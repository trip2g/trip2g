package graph

// The programmatic executor (NewExecutor, used by the MCP graphql_request
// tools) must run operations through the same tx middleware as the HTTP
// handler. Without it, multi-step mutations dispatched over MCP run outside a
// transaction and a mid-way failure leaves partial writes committed.

import (
	"context"
	"errors"
	"testing"

	graphql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"trip2g/internal/logger"
)

// recordingTxEnv implements the tx-lifecycle subset of Env used by the
// operation middleware. The embedded Env stays nil: resolvers touching
// anything else panic, which gqlgen recovers into a response error — enough
// to drive the rollback path.
type recordingTxEnv struct {
	Env
	acquired      bool
	released      bool
	releaseCommit bool
}

func (e *recordingTxEnv) Logger() logger.Logger       { return &logger.DummyLogger{} }
func (e *recordingTxEnv) IsDevMode() bool             { return true }
func (e *recordingTxEnv) ShortAPITokenSecret() string { return "test-secret" }

func (e *recordingTxEnv) AcquireTxEnvInRequest(_ context.Context, _ string) error {
	e.acquired = true
	return nil
}

func (e *recordingTxEnv) ReleaseTxEnvInRequest(_ context.Context, commit bool) error {
	e.released = true
	e.releaseCommit = commit
	return nil
}

func dispatchProgrammatic(t *testing.T, exec *executor.Executor, query string) *graphql.Response {
	t.Helper()

	ctx := graphql.StartOperationTrace(context.Background())
	now := graphql.Now()

	opCtx, errList := exec.CreateOperationContext(ctx, &graphql.RawParams{
		Query:    query,
		ReadTime: graphql.TraceTiming{Start: now, End: now},
	})
	require.Empty(t, errList)

	rh, respCtx := exec.DispatchOperation(ctx, opCtx)
	return rh(respCtx)
}

func TestNewExecutor_MutationRunsInsideTxWrapper(t *testing.T) {
	env := &recordingTxEnv{}
	exec := NewExecutor(env)

	resp := dispatchProgrammatic(t, exec, `mutation { signOut { __typename } }`)

	require.NotEmpty(t, resp.Errors, "resolver must fail without an appreq in context")
	require.True(t, env.acquired,
		"programmatic executor must open the request tx for mutations")
	require.True(t, env.released,
		"programmatic executor must release the request tx")
	require.False(t, env.releaseCommit,
		"a failed mutation must roll back, not commit")
}

func TestNewExecutor_SkipTxMutationDoesNotOpenTx(t *testing.T) {
	env := &recordingTxEnv{}
	exec := NewExecutor(env)

	// runCronJob waits on the single writer: opening a tx first would deadlock.
	_ = dispatchProgrammatic(t, exec,
		`mutation { admin { runCronJob(input: {id: 1}) { __typename } } }`)

	require.False(t, env.acquired, "@skipTx mutation must not open a transaction")
}

// commitFailOpsEnv acquires fine but fails on commit.
type commitFailOpsEnv struct {
	stubOpsEnv
}

func (e *commitFailOpsEnv) AcquireTxEnvInRequest(_ context.Context, _ string) error { return nil }

func (e *commitFailOpsEnv) ReleaseTxEnvInRequest(_ context.Context, commit bool) error {
	if commit {
		return errors.New("disk full")
	}
	return nil
}

func TestMakeAroundOperations_CommitErrorBecomesGraphQLError(t *testing.T) {
	env := &commitFailOpsEnv{stubOpsEnv: stubOpsEnv{devMode: true}}
	middleware := makeAroundOperations(&logger.DummyLogger{}, map[string]struct{}{}, env, nil)

	opCtx := &graphql.OperationContext{
		Operation: &ast.OperationDefinition{
			Operation: ast.Mutation,
			Name:      "TestMutation",
		},
	}
	ctx := graphql.WithOperationContext(context.Background(), opCtx)

	rh := middleware(ctx, func(_ context.Context) graphql.ResponseHandler {
		return func(_ context.Context) *graphql.Response { return &graphql.Response{} }
	})
	resp := rh(ctx)

	require.NotEmpty(t, resp.Errors,
		"a failed commit must surface as a GraphQL error, not a silent success")
}
