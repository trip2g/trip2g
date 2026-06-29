package graph

// Tests in this file guard the F1 wiring: that makeAroundOperations stamps a
// Bearer shortapitoken onto the appreq BEFORE resolvers run, so that the Query
// path (note / search / notePaths) can enforce read_patterns.
//
// Pre-fix: stampShortAPIToken was never called for QUERY operations.
//   Deleting lines 136-138 in handler.go left all graph tests passing.
//
// Post-fix: makeAroundOperations calls stampShortAPIToken unconditionally.
//   These tests call makeAroundOperations directly and will FAIL if those
//   lines are removed.

import (
	"context"
	"testing"
	"time"

	graphql "github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/vektah/gqlparser/v2/ast"

	"trip2g/internal/appreq"
	"trip2g/internal/logger"
	"trip2g/internal/shortapitoken"
)

// stubOpsEnv is a minimal implementation of operationsEnv for handler tests.
// Only IsDevMode() and ShortAPITokenSecret() are meaningful; the rest panic
// to catch unexpected calls in Query-path tests (no transaction is needed for
// Query operations — shouldSkipTx returns true and skips AcquireTxEnvInRequest).
type stubOpsEnv struct {
	devMode bool
	secret  string
}

func (s *stubOpsEnv) IsDevMode() bool                                         { return s.devMode }
func (s *stubOpsEnv) ShortAPITokenSecret() string                             { return s.secret }
func (s *stubOpsEnv) Logger() logger.Logger                                   { return &logger.DummyLogger{} }
func (s *stubOpsEnv) AcquireTxEnvInRequest(_ context.Context, _ string) error { panic("unexpected") }
func (s *stubOpsEnv) ReleaseTxEnvInRequest(_ context.Context, _ bool) error   { panic("unexpected") }

// queryCtxWithBearer returns a context ready for makeAroundOperations:
//   - a fasthttp.RequestCtx (which implements context.Context) with an
//     Authorization: Bearer header and an appreq.Request stored in it
//   - a gqlgen OperationContext for a Query operation layered on top
//
// appreq.FromCtx works because fasthttp.RequestCtx.Value() reads from user
// values set via SetUserValue, and gqlgen's context.WithValue chain passes
// unknown keys through to the underlying fasthttp context.
func queryCtxWithBearer(t *testing.T, bearerToken string) context.Context {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.Set("Authorization", "Bearer "+bearerToken)
	fctx.Request.SetRequestURI("http://example.com/graphql")

	req := &appreq.Request{Req: fctx}
	req.StoreInContext() // stores req in fctx via SetUserValue(ctxKey, req)

	opCtx := &graphql.OperationContext{
		Operation: &ast.OperationDefinition{
			Operation: ast.Query,
			Name:      "TestQuery",
		},
	}

	// Wrap fctx (which carries the appreq) with the gqlgen operation context.
	return graphql.WithOperationContext(fctx, opCtx)
}

// TestMakeAroundOperations_StampsShortAPIToken is the integration guard for F1.
//
// It constructs the middleware returned by makeAroundOperations and invokes it
// with a context that carries both an appreq (Bearer shortapitoken) and a
// gqlgen OperationContext (Query). After the middleware runs it asserts that
// WebhookReadPatterns has been populated on the appreq.
//
// This test FAILS if lines 136-138 in handler.go are removed.
func TestMakeAroundOperations_StampsShortAPIToken(t *testing.T) {
	const secret = "wiring-test-secret"

	tok, err := shortapitoken.Sign(shortapitoken.Data{
		Depth:        1,
		ReadPatterns: []string{"boards/**"},
	}, secret, time.Hour)
	require.NoError(t, err)

	env := &stubOpsEnv{devMode: true, secret: secret}
	middleware := makeAroundOperations(&logger.DummyLogger{}, map[string]struct{}{}, env, nil)

	ctx := queryCtxWithBearer(t, tok)

	var nextCalled bool
	next := graphql.OperationHandler(func(_ context.Context) graphql.ResponseHandler {
		nextCalled = true
		return func(_ context.Context) *graphql.Response { return &graphql.Response{} }
	})

	_ = middleware(ctx, next)

	require.True(t, nextCalled, "next handler must be called for Query ops")

	// Retrieve the appreq from the same context and confirm read patterns were stamped.
	req, err := appreq.FromCtx(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"boards/**"}, req.WebhookReadPatterns,
		"makeAroundOperations must stamp WebhookReadPatterns before resolvers run")
	require.True(t, req.WebhookScoped,
		"makeAroundOperations must set WebhookScoped=true for scoped shortapitoken")
}

// TestMakeAroundOperations_UnscopedRequestUnchanged verifies that a request
// without a Bearer shortapitoken is not modified (no patterns injected).
// Admin / cookie-authed requests must be unaffected.
func TestMakeAroundOperations_UnscopedRequestUnchanged(t *testing.T) {
	env := &stubOpsEnv{devMode: true, secret: "any-secret"}
	middleware := makeAroundOperations(&logger.DummyLogger{}, map[string]struct{}{}, env, nil)

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")
	req := &appreq.Request{Req: fctx}
	req.StoreInContext()

	opCtx := &graphql.OperationContext{
		Operation: &ast.OperationDefinition{
			Operation: ast.Query,
			Name:      "AdminQuery",
		},
	}
	ctx := graphql.WithOperationContext(fctx, opCtx)

	_ = middleware(ctx, func(_ context.Context) graphql.ResponseHandler {
		return func(_ context.Context) *graphql.Response { return &graphql.Response{} }
	})

	req2, err := appreq.FromCtx(ctx)
	require.NoError(t, err)
	require.Nil(t, req2.WebhookReadPatterns, "unscoped request must not have read patterns injected")
}
