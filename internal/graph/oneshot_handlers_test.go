package graph

// gqlgen's SSE transport calls the response handler until it returns nil, so
// handlers for one-shot operations must end after the first response.

import (
	"context"
	"errors"
	"testing"

	graphql "github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"trip2g/internal/logger"
)

const maxStreamCalls = 5

// countResponses mimics the SSE transport loop with a bound, so a handler that
// never ends fails the test instead of hanging it.
func countResponses(ctx context.Context, rh graphql.ResponseHandler) int {
	n := 0
	for n < maxStreamCalls {
		if rh(ctx) == nil {
			break
		}
		n++
	}
	return n
}

func TestMakeGraphqlErr_EndsAfterFirstResponse(t *testing.T) {
	rh := makeGraphqlErr(&logger.DummyLogger{})(errors.New("boom"))

	require.Equal(t, 1, countResponses(context.Background(), rh))
}

type failAcquireOpsEnv struct {
	stubOpsEnv
}

func (e *failAcquireOpsEnv) AcquireTxEnvInRequest(_ context.Context, _ string) error {
	return errors.New("no tx")
}

func TestMakeAroundOperations_AcquireFailureEndsAfterFirstResponse(t *testing.T) {
	env := &failAcquireOpsEnv{stubOpsEnv: stubOpsEnv{devMode: true}}
	middleware := makeAroundOperations(
		&logger.DummyLogger{}, map[string]struct{}{}, env, makeGraphqlErr(&logger.DummyLogger{}))

	ctx := graphql.WithOperationContext(context.Background(), &graphql.OperationContext{
		Operation: &ast.OperationDefinition{Operation: ast.Mutation, Name: "M"},
	})
	rh := middleware(ctx, func(_ context.Context) graphql.ResponseHandler {
		t.Fatal("next must not run when acquire fails")
		return nil
	})

	require.Equal(t, 1, countResponses(ctx, rh))
}

func TestMakeAroundOperations_TxWrapperPassesThroughEndOfStream(t *testing.T) {
	env := &commitFailOpsEnv{stubOpsEnv: stubOpsEnv{devMode: true}}
	middleware := makeAroundOperations(&logger.DummyLogger{}, map[string]struct{}{}, env, nil)

	ctx := graphql.WithOperationContext(context.Background(), &graphql.OperationContext{
		Operation: &ast.OperationDefinition{Operation: ast.Mutation, Name: "M"},
	})
	rh := middleware(ctx, func(_ context.Context) graphql.ResponseHandler {
		return graphql.OneShot(&graphql.Response{})
	})

	require.NotPanics(t, func() {
		require.Equal(t, 1, countResponses(ctx, rh))
	})
}
