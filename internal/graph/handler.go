package graph

import (
	"context"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/logger"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// buildSkipTxMap collects all field names (at any nesting level under Mutation)
// that carry the @skipTx directive, so shouldSkipTx can match them recursively.
func buildSkipTxMap(schema graphql.ExecutableSchema) map[string]struct{} {
	skipTxMutations := make(map[string]struct{})

	if schema.Schema().Mutation == nil {
		return skipTxMutations
	}

	visited := make(map[string]bool)
	var scan func(typeName string)
	scan = func(typeName string) {
		if visited[typeName] {
			return
		}
		visited[typeName] = true

		typeDef := schema.Schema().Types[typeName]
		if typeDef == nil {
			return
		}
		for _, field := range typeDef.Fields {
			if field.Directives.ForName("skipTx") != nil {
				skipTxMutations[field.Name] = struct{}{}
			}
			scan(field.Type.Name())
		}
	}
	scan(schema.Schema().Mutation.Name)

	return skipTxMutations
}

// sseTransport returns the SSE transport for subscriptions. The keepalive ping
// is required: over fasthttp (internal/fastgql) a client disconnect is only
// observable through a failed write/flush, so without periodic pings an idle
// subscription stream never learns the client is gone — its context is never
// cancelled and the resolver goroutine (plus its notebus subscriber) leaks.
func sseTransport() transport.SSE {
	return transport.SSE{KeepAlivePingInterval: 30 * time.Second}
}

func NewHandler(env Env) *handler.Server {
	log := env.Logger()

	resolver := Resolver{DefaultEnv: env}

	config := Config{
		Resolvers: &resolver,
	}

	config.Directives.SkipTx = func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		return next(ctx)
	}

	schema := NewExecutableSchema(config)

	srv := handler.New(schema)

	maxBodySize := int64(env.MaxRequestBodySize() * 1024 * 1024)

	srv.AddTransport(sseTransport())
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: maxBodySize,
		MaxMemory:     10 * 1024 * 1024,
	})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.FixedComplexityLimit(30))
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	logger := logger.WithPrefix(env.Logger(), "GraphQL:")
	skipTxMutations := buildSkipTxMap(schema)

	srv.AroundOperations(makeAroundOperations(logger, skipTxMutations, env, makeGraphqlErr(log)))

	return srv
}

func makeGraphqlErr(log logger.Logger) func(err error) graphql.ResponseHandler {
	return func(err error) graphql.ResponseHandler {
		log.Error("graphql error", "error", err)

		return func(ctx context.Context) *graphql.Response {
			return graphql.ErrorResponse(ctx, "%s", err.Error())
		}
	}
}

func disableIntrospection(ctx context.Context, opCtx *graphql.OperationContext, env operationsEnv) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		env.Logger().Warn("failed to get app request from context", "error", err)
		return
	}

	userToken, err := req.UserToken()
	if err != nil || userToken == nil {
		opCtx.DisableIntrospection = true
	}
}

// operationsEnv is the narrow subset of Env that makeAroundOperations requires.
// Defining it as a separate interface makes the function independently testable
// without a full Env mock.
type operationsEnv interface {
	IsDevMode() bool
	ShortAPITokenSecret() string
	Logger() logger.Logger
	AcquireTxEnvInRequest(ctx context.Context, label string) error
	ReleaseTxEnvInRequest(ctx context.Context, commit bool) error
}

func makeAroundOperations(
	log logger.Logger,
	skipTxMutations map[string]struct{},
	env operationsEnv,
	graphqlErr func(err error) graphql.ResponseHandler,
) graphql.OperationMiddleware {
	devMode := env.IsDevMode()

	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		operationContext := graphql.GetOperationContext(ctx)

		op := operationContext.Operation

		log.Debug("process", "operotion", op.Operation, "name", op.Name)

		// Stamp scoped-token claims (read/write patterns, depth, delivery identity)
		// once per operation, BEFORE resolvers run. This covers both Query and
		// Mutation paths: checkapikey.Resolve stamps mutations via X-API-Key /
		// Bearer handling inside the resolver, but query resolvers (note, search,
		// notePaths) never call checkapikey.Resolve, so without this hook the
		// Bearer shortapitoken is parsed as anonymous and read_patterns are never
		// set — making scope enforcement dead on the read path.
		if req, reqErr := appreq.FromCtx(ctx); reqErr == nil {
			stampShortAPIToken(req, env.ShortAPITokenSecret())
		}

		if !devMode {
			disableIntrospection(ctx, operationContext, env)
		}

		if shouldSkipTx(op, skipTxMutations) {
			return next(ctx)
		}

		err := env.AcquireTxEnvInRequest(ctx, op.Name)
		if err != nil {
			log.Error("failed to acquire transactioned env", "error", err)
			return graphqlErr(err)
		}

		rh := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			resp := rh(ctx)

			// А тут интересно, нужно ли отказывать транзакции только в случае ошибок
			// или в случае ErrorPayload так же нужно? Похоже нужно откатывать в случае
			// непредвиденных ошибок и дополнительно вводить специальную ошибку для rollback.
			if len(resp.Errors) > 0 {
				rollbackErr := env.ReleaseTxEnvInRequest(ctx, false)
				if rollbackErr != nil {
					log.Error("failed to release transactioned env with rollback", "error", rollbackErr)
				} else {
					log.Info("released transactioned env with rollback")
				}

				return resp
			}

			commitErr := env.ReleaseTxEnvInRequest(ctx, true)
			if commitErr != nil {
				log.Error("failed to release transactioned env with commit", "error", commitErr)
				// The writes are lost; the caller must not see a success response.
				resp.Errors = append(resp.Errors, gqlerror.Errorf("failed to commit transaction: %s", commitErr))
			} else {
				log.Debug("released transactioned env with commit")
			}

			return resp
		}
	}
}

func shouldSkipTx(op *ast.OperationDefinition, skipTxMutations map[string]struct{}) bool {
	if op.Operation != ast.Mutation {
		return true
	}
	return selectionHasSkipTx(op.SelectionSet, skipTxMutations)
}

// selectionHasSkipTx walks the selection set recursively. A single @skipTx
// field anywhere in the tree is enough to skip the transaction for the whole
// operation — nested mutations like AdminMutation.runCronJob are covered.
func selectionHasSkipTx(sel ast.SelectionSet, skipTx map[string]struct{}) bool {
	for _, s := range sel {
		switch sel := s.(type) {
		case *ast.Field:
			if staticallyExcluded(sel.Directives) {
				continue
			}
			if _, skip := skipTx[sel.Name]; skip {
				return true
			}
			if selectionHasSkipTx(sel.SelectionSet, skipTx) {
				return true
			}
		case *ast.InlineFragment:
			if staticallyExcluded(sel.Directives) {
				continue
			}
			if selectionHasSkipTx(sel.SelectionSet, skipTx) {
				return true
			}
		case *ast.FragmentSpread:
			if staticallyExcluded(sel.Directives) {
				continue
			}
			// Definition is resolved by the validator; fragment cycles are
			// rejected before this runs, so plain recursion is safe.
			if sel.Definition != nil && selectionHasSkipTx(sel.Definition.SelectionSet, skipTx) {
				return true
			}
		}
	}
	return false
}

// staticallyExcluded reports whether @skip(if: true) or @include(if: false)
// with a literal boolean removes the selection from execution. Variable-driven
// conditions can't be evaluated here (no variables at this layer), so they are
// treated as potentially executing.
func staticallyExcluded(directives ast.DirectiveList) bool {
	for _, d := range directives {
		var excludeWhen bool
		switch d.Name {
		case "skip":
			excludeWhen = true
		case "include":
			excludeWhen = false
		default:
			continue
		}
		arg := d.Arguments.ForName("if")
		if arg == nil || arg.Value == nil || arg.Value.Kind != ast.BooleanValue {
			continue
		}
		if (arg.Value.Raw == "true") == excludeWhen {
			return true
		}
	}
	return false
}

// NewExecutor builds an executable schema for env and returns a low-level
// executor suitable for programmatic GraphQL dispatch (no HTTP transport).
func NewExecutor(env Env) *executor.Executor {
	config := Config{
		Resolvers: &Resolver{DefaultEnv: env},
	}
	config.Directives.SkipTx = func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		return next(ctx)
	}
	schema := NewExecutableSchema(config)
	exec := executor.New(schema)
	exec.Use(extension.Introspection{})

	// Programmatic (MCP) dispatch must share the HTTP path's tx lifecycle:
	// without this middleware, multi-step mutations run outside a transaction
	// and a mid-way failure leaves partial writes committed.
	log := logger.WithPrefix(env.Logger(), "GraphQL:")
	exec.AroundOperations(makeAroundOperations(log, buildSkipTxMap(schema), env, makeGraphqlErr(env.Logger())))

	return exec
}
