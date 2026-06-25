package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"trip2g/internal/appreq"
	"trip2g/internal/case/listactiveusersubgraphs"
	"trip2g/internal/db"
	"trip2g/internal/fastgql"
	"trip2g/internal/graph"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type txEnvKeyType struct{}

//nolint:gochecknoglobals // Context key for transactional env
var txEnvKey = txEnvKeyType{}

type graphTransactions struct {
	sync.Mutex
	EnvMap map[*app]*sql.Tx
}

func (a *app) ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error) {
	// TODO: add caching for this method
	return listactiveusersubgraphs.Resolve(ctx, a, userID)
}

func (a *app) AcquireTxEnvInRequest(ctx context.Context, label string) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get request from context: %w", err)
	}

	tx, err := a.writeConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	logLabel := fmt.Sprintf("tx %s", label+":")
	queries := db.NewWriteQueries(db.WithLogger(tx, logger.WithPrefix(a.log, logLabel)))

	newEnv := *a
	newEnv.queries = queries.Queries
	newEnv.Queries = queries.Queries
	newEnv.WriteQueries = queries
	newEnv.currentTx = tx

	// override the context with the new tx env
	req.Env = &newEnv

	a.graphTxs.Lock()
	defer a.graphTxs.Unlock()

	a.graphTxs.EnvMap[&newEnv] = tx

	return nil
}

func (a *app) ReleaseTxEnvInRequest(ctx context.Context, commit bool) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get request from context: %w", err)
	}

	a.graphTxs.Lock()
	defer a.graphTxs.Unlock()

	envPtr, ok := req.Env.(*app)
	if !ok {
		return errors.New("failed to cast env to *app")
	}
	tx, ok := a.graphTxs.EnvMap[envPtr]
	if !ok {
		return fmt.Errorf("transactioned env not found for request: %v", req.Env)
	}

	// Clean up the map entry regardless of commit/rollback
	delete(a.graphTxs.EnvMap, envPtr)

	if commit {
		commitErr := tx.Commit()
		if commitErr != nil {
			return fmt.Errorf("failed to commit transaction: %w", commitErr)
		}

		return nil
	}

	err = tx.Rollback()
	if err != nil {
		a.log.Error("failed to rollback transaction", "error", err)
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

func (a *app) prepareGraphQLHandler() func(prefix string) func(ctx *fasthttp.RequestCtx, path string) bool {
	// graphql.
	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(playground.Handler("GraphQL playground", "/_system/graphql"))

	gqlMetrics := metrics.NewGraphQLMetrics()

	a.gqlServer = graph.NewHandler(a)
	a.gqlServer.Use(gqlMetrics)
	a.gqlServer.AroundOperations(gqlMetrics.Middleware())
	a.gqlExecutor = graph.NewExecutor(a)
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(a.gqlServer)
	compressedGraphqlHandler := fasthttp.CompressHandler(graphqlHandler)

	// SSE uses a custom handler that does not pool the response writer,
	// avoiding a data race in fasthttpadaptor where the pooled writer
	// is recycled while the SSE goroutine is still writing to it.
	sseHandler := fastgql.NewSSEHandler(a.gqlServer, fastgql.WithBaseContext(func(fctx *fasthttp.RequestCtx) context.Context {
		req, err := appreq.FromCtx(fctx)
		if err != nil {
			return context.Background()
		}
		return appreq.NewContext(context.Background(), req.Snapshot())
	}))

	return func(prefix string) func(ctx *fasthttp.RequestCtx, path string) bool {
		return func(ctx *fasthttp.RequestCtx, path string) bool {
			if !strings.HasPrefix(path, prefix) {
				return false
			}
			switch {
			case string(ctx.Method()) == "GET":
				playgroundHandler(ctx)
			case strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream"):
				sseHandler(ctx)
			default:
				compressedGraphqlHandler(ctx)
			}
			return true
		}
	}
}

// getEnvOrDefault retrieves the environment from the request context or returns a default environment.
// the context from the request wrapped all queries in a transaction.
func getEnvOrDefault[T any](ctx context.Context, defaultEnv *app) (T, error) {
	var zero T

	req, err := appreq.FromCtx(ctx)
	if err != nil {
		if errors.Is(err, appreq.ErrNotFound) {
			env, ok := any(defaultEnv).(T)
			if ok {
				return env, nil
			}

			return zero, fmt.Errorf("app does not implement required type: %T", zero)
		}

		return zero, fmt.Errorf("failed to get request from context: %w", err)
	}

	env, ok := req.Env.(T)
	if ok {
		return env, nil
	}

	return zero, fmt.Errorf("request env does not implement required type: %T", zero)
}

// GraphQLRequest executes a GraphQL query programmatically with admin privileges.
func (a *app) GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error) {
	ctx = appreq.WithAdminToken(ctx)
	// gqlgen requires StartOperationTrace before CreateOperationContext;
	// normally handler.Server.ServeHTTP does this — we bypass HTTP transport,
	// so set it up manually.
	ctx = graphql.StartOperationTrace(ctx)
	now := graphql.Now()

	// gqlgen's Int64 / int scalar UnmarshalGQL rejects float64 with
	// "float64 is not an int64". gqlgen's transport.POST uses json.Decoder
	// with UseNumber so values arrive as json.Number; reproduce that here.
	convertedVars := variables
	if len(variables) > 0 {
		rawVars, marshalErr := json.Marshal(variables)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal variables: %w", marshalErr)
		}
		dec := json.NewDecoder(bytes.NewReader(rawVars))
		dec.UseNumber()
		convertedVars = map[string]any{}
		if decodeErr := dec.Decode(&convertedVars); decodeErr != nil {
			return nil, fmt.Errorf("decode variables: %w", decodeErr)
		}
	}

	params := &graphql.RawParams{
		Query:     query,
		Variables: convertedVars,
		ReadTime: graphql.TraceTiming{
			Start: now,
			End:   now,
		},
	}

	opCtx, errList := a.gqlExecutor.CreateOperationContext(ctx, params)
	if errList != nil {
		return nil, fmt.Errorf("graphql operation context: %w", errList)
	}

	responseHandler, respCtx := a.gqlExecutor.DispatchOperation(ctx, opCtx)
	resp := responseHandler(respCtx)
	return json.Marshal(resp)
}

// FederatedGraphQLEnabled reports whether the federated_graphql_request MCP tool is enabled.
func (a *app) FederatedGraphQLEnabled() bool { return a.config.MCPFederatedGraphQLEnabled }

// GraphQLRequestScoped executes a GraphQL query with a federated-scoped token (never admin).
func (a *app) GraphQLRequestScoped(ctx context.Context, query string, variables map[string]any, allowedSubgraphs []string) ([]byte, error) {
	ctx = appreq.WithFederatedToken(ctx, allowedSubgraphs)
	// gqlgen requires StartOperationTrace before CreateOperationContext;
	// normally handler.Server.ServeHTTP does this — we bypass HTTP transport,
	// so set it up manually.
	ctx = graphql.StartOperationTrace(ctx)
	now := graphql.Now()

	// gqlgen's Int64 / int scalar UnmarshalGQL rejects float64 with
	// "float64 is not an int64". gqlgen's transport.POST uses json.Decoder
	// with UseNumber so values arrive as json.Number; reproduce that here.
	convertedVars := variables
	if len(variables) > 0 {
		rawVars, marshalErr := json.Marshal(variables)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal variables: %w", marshalErr)
		}
		dec := json.NewDecoder(bytes.NewReader(rawVars))
		dec.UseNumber()
		convertedVars = map[string]any{}
		if decodeErr := dec.Decode(&convertedVars); decodeErr != nil {
			return nil, fmt.Errorf("decode variables: %w", decodeErr)
		}
	}

	params := &graphql.RawParams{
		Query:     query,
		Variables: convertedVars,
		ReadTime: graphql.TraceTiming{
			Start: now,
			End:   now,
		},
	}

	opCtx, errList := a.gqlExecutor.CreateOperationContext(ctx, params)
	if errList != nil {
		return nil, fmt.Errorf("graphql operation context: %w", errList)
	}

	responseHandler, respCtx := a.gqlExecutor.DispatchOperation(ctx, opCtx)
	resp := responseHandler(respCtx)
	return json.Marshal(resp)
}
