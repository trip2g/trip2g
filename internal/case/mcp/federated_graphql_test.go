package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
	"trip2g/internal/openai"

	"github.com/stretchr/testify/require"
)

// fedGQLEnv is a minimal Env stub for federated GraphQL handler tests.
// Unexpected method calls panic to catch security violations.
type fedGQLEnv struct {
	federatedEnabled bool
	scopedFunc       func(ctx context.Context, query string, variables map[string]any, allowedSubgraphs []string) ([]byte, error)
	graphqlFunc      func(ctx context.Context, query string, variables map[string]any) ([]byte, error)
}

func (e *fedGQLEnv) FederatedGraphQLEnabled() bool { return e.federatedEnabled }
func (e *fedGQLEnv) GraphQLRequestScoped(ctx context.Context, query string, variables map[string]any, allowedSubgraphs []string) ([]byte, error) {
	if e.scopedFunc == nil {
		panic("unexpected GraphQLRequestScoped call")
	}
	return e.scopedFunc(ctx, query, variables, allowedSubgraphs)
}
func (e *fedGQLEnv) GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error) {
	if e.graphqlFunc != nil {
		return e.graphqlFunc(ctx, query, variables)
	}
	panic("unexpected admin GraphQLRequest call — security violation: must NOT reach admin path under fedAuth")
}
func (e *fedGQLEnv) MCPMetrics() *metrics.MCPMetrics     { return nil }
func (e *fedGQLEnv) Features() features.Features         { panic("unexpected") }
func (e *fedGQLEnv) LatestNoteViews() *model.NoteViews   { panic("unexpected") }
func (e *fedGQLEnv) LatestNoteChunks() []model.NoteChunk { panic("unexpected") }
func (e *fedGQLEnv) CanReadNote(_ context.Context, _ *model.NoteView) (bool, error) {
	panic("unexpected")
}
func (e *fedGQLEnv) FederationClient(_ context.Context, _ string) (model.Federation, error) {
	panic("unexpected")
}
func (e *fedGQLEnv) SearchLatestNotes(_ string) ([]model.SearchResult, error) { panic("unexpected") }
func (e *fedGQLEnv) SearchLiveNotes(_ string) ([]model.SearchResult, error)   { panic("unexpected") }
func (e *fedGQLEnv) LiveNoteChunks() []model.NoteChunk                        { panic("unexpected") }
func (e *fedGQLEnv) LiveNoteViews() *model.NoteViews                          { panic("unexpected") }
func (e *fedGQLEnv) OpenAI() *openai.Client                                   { panic("unexpected") }
func (e *fedGQLEnv) SiteConfig(_ context.Context) model.SiteConfig            { panic("unexpected") }
func (e *fedGQLEnv) PublicURL() string                                        { panic("unexpected") }
func (e *fedGQLEnv) NoteURL(_ *model.NoteView) string                         { panic("unexpected") }
func (e *fedGQLEnv) Logger() logger.Logger                                    { return &logger.DummyLogger{} }
func (e *fedGQLEnv) FederationSecretByKBURL(_ context.Context, _ string) (db.FederationSecret, bool, error) {
	panic("unexpected")
}
func (e *fedGQLEnv) FederationSecretByKID(_ context.Context, _ string) (db.FederationSecret, bool, error) {
	panic("unexpected")
}
func (e *fedGQLEnv) ListFederationSecretSubgraphsByKID(_ context.Context, _ string) ([]string, error) {
	panic("unexpected")
}
func (e *fedGQLEnv) DecryptData(_ []byte) ([]byte, error) { panic("unexpected") }
func (e *fedGQLEnv) FederationMaxDepth() int              { panic("unexpected") }
func (e *fedGQLEnv) FederatedFanoutConcurrency() int      { panic("unexpected") }
func (e *fedGQLEnv) FederatedFanoutLimit() int            { panic("unexpected") }
func (e *fedGQLEnv) FederatedFanoutTimeout() time.Duration {
	panic("unexpected")
}
func (e *fedGQLEnv) ResolveAPIKey(_ context.Context, _, _ string) (*db.ApiKey, error) {
	panic("unexpected")
}

// --- handleGraphQLRequestScoped tests ---

// TestHandleGraphQLRequestScoped_MutationRejected: fed ctx + mutation → rejected before scoped call.
func TestHandleGraphQLRequestScoped_MutationRejected(t *testing.T) {
	env := &fedGQLEnv{federatedEnabled: true}
	ctx := contextWithFederationAuth(context.Background(), "kid1", []string{"subA"})
	argsRaw, _ := json.Marshal(GraphQLRequestArguments{Query: `mutation { createNote { id } }`})

	resp := handleGraphQLRequestScoped(ctx, env, 1, json.RawMessage(argsRaw), []string{"subA"})

	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "mutation")
}

// TestHandleGraphQLRequestScoped_AdminFieldRejected: fed ctx + admin root field → rejected.
func TestHandleGraphQLRequestScoped_AdminFieldRejected(t *testing.T) {
	env := &fedGQLEnv{federatedEnabled: true}
	ctx := contextWithFederationAuth(context.Background(), "kid1", []string{"subA"})
	argsRaw, _ := json.Marshal(GraphQLRequestArguments{Query: `{ admin { users { id } } }`})

	resp := handleGraphQLRequestScoped(ctx, env, 1, json.RawMessage(argsRaw), []string{"subA"})

	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "admin")
}

// TestHandleGraphQLRequestScoped_ValidQueryCallsScopedNotAdmin: fed ctx + valid query →
// GraphQLRequestScoped called with ctx's AllowedSubgraphs; admin GraphQLRequest NEVER called.
func TestHandleGraphQLRequestScoped_ValidQueryCallsScopedNotAdmin(t *testing.T) {
	allowedSubgraphs := []string{"subA", "subB"}
	var gotSubgraphs []string
	scopedCalled := false

	env := &fedGQLEnv{
		federatedEnabled: true,
		scopedFunc: func(_ context.Context, query string, _ map[string]any, allowed []string) ([]byte, error) {
			scopedCalled = true
			gotSubgraphs = allowed
			return []byte(`{"data":{"note":{"title":"ok"}}}`), nil
		},
		// graphqlFunc is nil → panic if called (security assertion via fedGQLEnv.GraphQLRequest)
	}
	ctx := contextWithFederationAuth(context.Background(), "kid1", allowedSubgraphs)
	argsRaw, _ := json.Marshal(GraphQLRequestArguments{Query: `{ note(path: "x") { title } }`})

	resp := handleGraphQLRequestScoped(ctx, env, 1, json.RawMessage(argsRaw), allowedSubgraphs)

	require.Nil(t, resp.Error)
	require.True(t, scopedCalled, "GraphQLRequestScoped must be called")
	require.Equal(t, allowedSubgraphs, gotSubgraphs, "scoped call must receive fed auth AllowedSubgraphs")
}

// --- handleFederatedGraphQLRequest tests ---

// TestHandleFederatedGraphQLRequest_FlagOff: FederatedGraphQLEnabled=false → MethodNotFound.
func TestHandleFederatedGraphQLRequest_FlagOff(t *testing.T) {
	env := &fedGQLEnv{federatedEnabled: false}
	argsRaw, _ := json.Marshal(FederatedGraphQLRequestArguments{KBID: "kb1", Query: `{ note(path:"x"){title} }`})

	resp := handleFederatedGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))

	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeMethodNotFound, resp.Error.Code)
}

// TestHandleFederatedGraphQLRequest_MutationRejected: flag on + mutation → error before network.
func TestHandleFederatedGraphQLRequest_MutationRejected(t *testing.T) {
	env := &fedGQLEnv{federatedEnabled: true}
	// Build a minimal NoteViews with a federation KB so accessibleKBNotes succeeds.
	// Actually, validateReadOnlyQuery runs before callFederatedSingleKB, so no KB needed.
	argsRaw, _ := json.Marshal(FederatedGraphQLRequestArguments{KBID: "kb1", Query: `mutation { createNote { id } }`})

	resp := handleFederatedGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))

	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "mutation")
}

// TestHandleFederatedGraphQLRequest_AdminFieldRejected: flag on + admin root field → error.
func TestHandleFederatedGraphQLRequest_AdminFieldRejected(t *testing.T) {
	env := &fedGQLEnv{federatedEnabled: true}
	argsRaw, _ := json.Marshal(FederatedGraphQLRequestArguments{KBID: "kb1", Query: `{ admin { id } }`})

	resp := handleFederatedGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))

	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "admin")
}
