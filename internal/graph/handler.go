package graph

import (
	"context"
	"trip2g/internal/logger"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"
)

func skipTx(operationContext *graphql.OperationContext) bool {
	for _, selection := range operationContext.Operation.SelectionSet {
		if field, ok := selection.(*ast.Field); ok {
			for _, directive := range field.Directives {
				if directive.Name == "skipTx" {
					return true
				}
			}
		}
	}

	return false
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

	srv := handler.New(NewExecutableSchema(config))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: 30 * 1024 * 1024,
		MaxMemory:     10 * 1024 * 1024,
	})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	graphqlErr := func(err error) graphql.ResponseHandler {
		log.Error("graphql error", "error", err)

		return func(ctx context.Context) *graphql.Response {
			return graphql.ErrorResponse(ctx, "%s", err.Error())
		}
	}

	logger := logger.WithPrefix(env.Logger(), "GraphQL:")

	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		operationContext := graphql.GetOperationContext(ctx)

		op := operationContext.Operation

		logger.Debug("process", "operotion", op.Operation, "name", op.Name)

		if op.Operation != ast.Mutation || skipTx(operationContext) {
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
			} else {
				log.Debug("released transactioned env with commit")
			}

			return resp
		}
	})

	return srv
}
