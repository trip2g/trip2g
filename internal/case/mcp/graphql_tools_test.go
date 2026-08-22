package mcp

import (
	"context"
	"encoding/json"
	"strings"
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

// gqlRequestEnv is a minimal Env stub for testing handleGraphQLRequest.
// All methods except GraphQLRequest panic to catch unexpected calls.
type gqlRequestEnv struct {
	graphqlFunc func(ctx context.Context, query string, variables map[string]any) ([]byte, error)
}

func (e *gqlRequestEnv) GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error) {
	return e.graphqlFunc(ctx, query, variables)
}
func (e *gqlRequestEnv) MCPMetrics() *metrics.MCPMetrics     { return nil }
func (e *gqlRequestEnv) Features() features.Features         { panic("unexpected") }
func (e *gqlRequestEnv) LatestNoteViews() *model.NoteViews   { panic("unexpected") }
func (e *gqlRequestEnv) LatestNoteChunks() []model.NoteChunk { panic("unexpected") }
func (e *gqlRequestEnv) CanReadNote(_ context.Context, _ *model.NoteView) (bool, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) FederationClient(_ context.Context, _ string) (model.Federation, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) SearchLatestNotes(_ string) ([]model.SearchResult, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) SearchLiveNotes(_ string) ([]model.SearchResult, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) LiveNoteChunks() []model.NoteChunk             { panic("unexpected") }
func (e *gqlRequestEnv) LiveNoteViews() *model.NoteViews               { panic("unexpected") }
func (e *gqlRequestEnv) OpenAI() *openai.Client                        { panic("unexpected") }
func (e *gqlRequestEnv) SiteConfig(_ context.Context) model.SiteConfig { panic("unexpected") }
func (e *gqlRequestEnv) PublicURL() string                             { panic("unexpected") }
func (e *gqlRequestEnv) NoteURL(_ *model.NoteView) string              { panic("unexpected") }
func (e *gqlRequestEnv) Logger() logger.Logger                         { return &logger.DummyLogger{} }
func (e *gqlRequestEnv) AuditLogger() logger.Logger                    { return &logger.DummyLogger{} }
func (e *gqlRequestEnv) EncryptData(_ []byte) ([]byte, error)          { panic("unexpected") }
func (e *gqlRequestEnv) ClearFederationSecretPrev(_ context.Context, _ db.ClearFederationSecretPrevParams) error {
	panic("unexpected")
}
func (e *gqlRequestEnv) FederationSecretByID(_ context.Context, _ int64) (db.FederationSecret, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) RotateFederationSecret(_ context.Context, _ db.RotateFederationSecretParams) error {
	panic("unexpected")
}
func (e *gqlRequestEnv) FederationSecretByKBURL(_ context.Context, _ string) (db.FederationSecret, bool, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) FederationSecretByKID(_ context.Context, _ string) (db.FederationSecret, bool, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) ListFederationSecretSubgraphsByKID(_ context.Context, _ string) ([]string, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) DecryptData(_ []byte) ([]byte, error) { panic("unexpected") }
func (e *gqlRequestEnv) FederationMaxDepth() int              { panic("unexpected") }
func (e *gqlRequestEnv) FederatedFanoutConcurrency() int      { panic("unexpected") }
func (e *gqlRequestEnv) FederatedFanoutLimit() int            { panic("unexpected") }
func (e *gqlRequestEnv) FederatedFanoutTimeout() time.Duration {
	panic("unexpected")
}
func (e *gqlRequestEnv) CachedFederatedInstructions(_ string) (model.FederationResult, bool) {
	panic("unexpected")
}
func (e *gqlRequestEnv) StoreFederatedInstructions(_ string, _ model.FederationResult) {
	panic("unexpected")
}
func (e *gqlRequestEnv) ResolveAPIKey(_ context.Context, _, _ string) (*db.ApiKey, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) GraphQLRequestScoped(_ context.Context, _ string, _ map[string]any, _ []string) ([]byte, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) FederatedGraphQLEnabled() bool { panic("unexpected") }

func TestFilterIntrospection_MatchAndExpand(t *testing.T) {
	raw := []byte(`{
		"data": {
			"__schema": {
				"queryType": {"name": "Query"},
				"mutationType": {"name": "Mutation"},
				"types": [
					{"kind": "OBJECT", "name": "Note", "fields": [
						{"name": "id", "type": {"kind": "SCALAR", "name": "Int", "ofType": null}},
						{"name": "tags", "type": {"kind": "LIST", "name": "", "ofType": {"kind": "SCALAR", "name": "String", "ofType": null}}}
					]},
					{"kind": "OBJECT", "name": "User", "fields": []},
					{"kind": "SCALAR", "name": "Int"},
					{"kind": "SCALAR", "name": "String"}
				]
			}
		}
	}`)

	out, err := filterIntrospection(raw, "Note")
	require.NoError(t, err)

	s := string(out)
	require.Contains(t, s, `"name":"Note"`)
	require.Contains(t, s, `"name":"Int"`)
	require.Contains(t, s, `"name":"String"`)
	require.NotContains(t, s, `"name":"User"`)
}

func TestFilterIntrospection_MatchesByFieldName(t *testing.T) {
	raw := []byte(`{
		"data": {
			"__schema": {
				"queryType": {"name": "Query"},
				"types": [
					{"kind": "OBJECT", "name": "Mutation", "fields": [
						{"name": "createWebhook", "type": {"kind": "OBJECT", "name": "Webhook", "ofType": null}}
					]},
					{"kind": "OBJECT", "name": "Webhook", "fields": []},
					{"kind": "OBJECT", "name": "Unrelated", "fields": []}
				]
			}
		}
	}`)

	out, err := filterIntrospection(raw, "webhook")
	require.NoError(t, err)

	s := string(out)
	require.Contains(t, s, `"name":"Mutation"`)
	require.Contains(t, s, `"name":"Webhook"`)
	require.NotContains(t, s, `"name":"Unrelated"`)
}

func TestFilterIntrospection_SkipsBuiltinTypes(t *testing.T) {
	raw := []byte(`{
		"data": {
			"__schema": {
				"types": [
					{"kind": "OBJECT", "name": "__Schema"},
					{"kind": "OBJECT", "name": "Query"}
				]
			}
		}
	}`)

	out, err := filterIntrospection(raw, ".*")
	require.NoError(t, err)
	require.NotContains(t, string(out), `"__Schema"`)
}

func TestHandleGraphQLIntrospection_RequiresPattern(t *testing.T) {
	resp := handleGraphQLIntrospection(context.Background(), nil, 1, json.RawMessage(`{}`))
	require.NotNil(t, resp.Error)
	require.Contains(t, resp.Error.Message, "pattern is required")
}

func TestHandleGraphQLRequest_RequiresQuery(t *testing.T) {
	resp := handleGraphQLRequest(context.Background(), nil, 1, json.RawMessage(`{}`))
	require.NotNil(t, resp.Error)
	require.Contains(t, resp.Error.Message, "query is required")
}

// Sanity: ensure the JSON envelope is preserved for the agent.
func TestFilterIntrospection_PreservesSchemaEnvelope(t *testing.T) {
	raw := []byte(`{"data":{"__schema":{"queryType":{"name":"Query"},"types":[{"kind":"OBJECT","name":"Foo"}]}}}`)
	out, err := filterIntrospection(raw, "Foo")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(out), `{"data":{"__schema":`))
}

// TestHandleGraphQLRequest_StructuredContent: happy-path test verifying that the
// response envelope appears once in StructuredContent and the text stub is short.
func TestHandleGraphQLRequest_StructuredContent(t *testing.T) {
	canned := []byte(`{"data":{"note":{"title":"x"}}}`)
	env := &gqlRequestEnv{
		graphqlFunc: func(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
			return canned, nil
		},
	}
	argsRaw, err := json.Marshal(GraphQLRequestArguments{Query: `{ note(path: "x") { title } }`})
	require.NoError(t, err)

	resp := handleGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(CallToolResult)
	require.True(t, ok, "result must be CallToolResult")
	require.Len(t, result.Content, 1)
	require.Equal(t, "structured result", result.Content[0].Text, "text must be stub, not JSON payload")
	require.NotNil(t, result.StructuredContent, "StructuredContent must be set")

	// StructuredContent must deep-equal the parsed envelope.
	gotJSON, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, string(canned), string(gotJSON))
}

// --- validateReadOnlyQuery tests (federated/scoped path) ---

func TestValidateReadOnlyQuery_RejectsMutation(t *testing.T) {
	err := validateReadOnlyQuery(`mutation { createNote { id } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutation")
}

func TestValidateReadOnlyQuery_RejectsDisallowedRootField(t *testing.T) {
	err := validateReadOnlyQuery(`{ admin { users { id } } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin")
}

func TestValidateReadOnlyQuery_RejectsMultiOpWithMutation(t *testing.T) {
	err := validateReadOnlyQuery(`query A { note(path: "x") { title } } mutation B { createNote { id } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutation")
}

func TestValidateReadOnlyQuery_AllowsNote(t *testing.T) {
	err := validateReadOnlyQuery(`{ note(path: "x") { title } }`, graphqlFederatedRootFields)
	require.NoError(t, err)
}

func TestValidateReadOnlyQuery_AllowsSearch(t *testing.T) {
	err := validateReadOnlyQuery(`{ search(query: "y") { title notePath } }`, graphqlFederatedRootFields)
	require.NoError(t, err)
}

func TestValidateReadOnlyQuery_RejectsSubscription(t *testing.T) {
	err := validateReadOnlyQuery(`subscription { noteUpdated { id } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "subscription")
}

func TestValidateReadOnlyQuery_RejectsPublicUrl(t *testing.T) {
	err := validateReadOnlyQuery(`{ publicUrl }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "publicUrl")
}

func TestValidateReadOnlyQuery_ParseError(t *testing.T) {
	err := validateReadOnlyQuery(`{{{not valid graphql`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse error")
}

// --- validateReadOnlyQuery guard-bypass regression tests ---
// These vectors are security-relevant: they confirm that inline fragments,
// fragment spreads, and field aliases cannot be used to bypass the allowlist
// on the federated/scoped path.

func TestValidateReadOnlyQuery_RejectsRootInlineFragment(t *testing.T) {
	// Inline fragment at root level must be rejected — the selector is not a *ast.Field.
	err := validateReadOnlyQuery(`query { ... on Query { admin { id } } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only field selections are allowed at query root")
}

func TestValidateReadOnlyQuery_RejectsRootFragmentSpread(t *testing.T) {
	// Fragment spread at root level must be rejected — same reason.
	err := validateReadOnlyQuery(`query Q { ...F } fragment F on Query { admin { id } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only field selections are allowed at query root")
}

func TestValidateReadOnlyQuery_RejectsAliasedBannedField(t *testing.T) {
	// Field.Name (not Field.Alias) is checked; an alias must not bypass the allowlist.
	err := validateReadOnlyQuery(`query { n: admin { id } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin")
}

func TestValidateReadOnlyQuery_AllowsAliasedAllowedField(t *testing.T) {
	// An alias on an allowed field must be accepted.
	err := validateReadOnlyQuery(`query { n: note(path:"x") { title } }`, graphqlFederatedRootFields)
	require.NoError(t, err)
}

// --- Federated allowlist tests ---
// notePaths and resolveWikilinks must be blocked on the federated path.

func TestValidateReadOnlyQuery_Federated_RejectsNotePaths(t *testing.T) {
	// notePaths is NOT in the federated allowlist.
	err := validateReadOnlyQuery(`{ notePaths }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "notePaths")
}

func TestValidateReadOnlyQuery_Federated_RejectsResolveWikilinks(t *testing.T) {
	// resolveWikilinks is NOT in the federated allowlist.
	err := validateReadOnlyQuery(`{ resolveWikilinks(paths: ["x"]) { path href } }`, graphqlFederatedRootFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolveWikilinks")
}

// --- rejectSubscription tests (admin path) ---

func TestRejectSubscription_AllowsQuery(t *testing.T) {
	require.NoError(t, rejectSubscription(`{ note(path: "x") { title } }`))
}

func TestRejectSubscription_AllowsMutation(t *testing.T) {
	require.NoError(t, rejectSubscription(`mutation { createNote { id } }`))
}

func TestRejectSubscription_RejectsSubscription(t *testing.T) {
	err := rejectSubscription(`subscription { noteUpdated { id } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "subscription")
}

func TestRejectSubscription_ParseError(t *testing.T) {
	err := rejectSubscription(`{{{not valid graphql`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse error")
}

// --- handleGraphQLRequest edge-case tests ---

// TestHandleGraphQLRequest_ErrorsPreserved: a GraphQL errors-bearing response
// (data: null, errors: [...]) must be forwarded as StructuredContent so the
// agent can inspect both the errors key and the data key.
func TestHandleGraphQLRequest_ErrorsPreserved(t *testing.T) {
	canned := []byte(`{"data":null,"errors":[{"message":"boom"}]}`)
	env := &gqlRequestEnv{
		graphqlFunc: func(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
			return canned, nil
		},
	}
	argsRaw, err := json.Marshal(GraphQLRequestArguments{Query: `{ note(path: "x") { title } }`})
	require.NoError(t, err)

	resp := handleGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(CallToolResult)
	require.True(t, ok, "result must be CallToolResult")
	require.Len(t, result.Content, 1)
	require.Equal(t, "structured result", result.Content[0].Text, "text must be stub")
	require.NotNil(t, result.StructuredContent, "StructuredContent must be set for errors-bearing response")

	// The errors key must survive the round-trip.
	gotJSON, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.Contains(t, string(gotJSON), `"errors"`)
	require.Contains(t, string(gotJSON), `"boom"`)
}

// TestHandleGraphQLRequest_MalformedJSONFallback: when GraphQLRequest returns
// bytes that are not valid JSON, the handler must fall back to textToolResult
// (Content[0].Text contains the raw bytes; StructuredContent is nil).
func TestHandleGraphQLRequest_MalformedJSONFallback(t *testing.T) {
	raw := []byte("not json")
	env := &gqlRequestEnv{
		graphqlFunc: func(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
			return raw, nil
		},
	}
	argsRaw, err := json.Marshal(GraphQLRequestArguments{Query: `{ note(path: "x") { title } }`})
	require.NoError(t, err)

	resp := handleGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(CallToolResult)
	require.True(t, ok, "result must be CallToolResult")
	require.Len(t, result.Content, 1)
	require.Contains(t, result.Content[0].Text, "not json", "text must contain raw bytes")
	require.Nil(t, result.StructuredContent, "StructuredContent must be nil for malformed JSON")
}

// TestHandleGraphQLRequest_MutationDispatched: admin handler must forward mutations
// to env.GraphQLRequest without rejection.
func TestHandleGraphQLRequest_MutationDispatched(t *testing.T) {
	canned := []byte(`{"data":{"adminMutation":{"createNote":{"id":"1"}}}}`)
	called := false
	env := &gqlRequestEnv{
		graphqlFunc: func(_ context.Context, query string, _ map[string]any) ([]byte, error) {
			called = true
			require.Contains(t, query, "mutation")
			return canned, nil
		},
	}
	argsRaw, err := json.Marshal(GraphQLRequestArguments{Query: `mutation { adminMutation { createNote { id } } }`})
	require.NoError(t, err)

	resp := handleGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))
	require.Nil(t, resp.Error, "admin mutation must succeed, got: %v", resp.Error)
	require.True(t, called, "GraphQLRequest must be invoked for admin mutation")

	result, ok := resp.Result.(CallToolResult)
	require.True(t, ok, "result must be CallToolResult")
	require.Len(t, result.Content, 1)
	require.Equal(t, "structured result", result.Content[0].Text)
	require.NotNil(t, result.StructuredContent)
}

// TestHandleGraphQLRequest_SubscriptionRejectedByHandler: subscriptions must be
// rejected before reaching env.GraphQLRequest (unsupported transport).
func TestHandleGraphQLRequest_SubscriptionRejectedByHandler(t *testing.T) {
	env := &gqlRequestEnv{
		graphqlFunc: func(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
			panic("must not be called")
		},
	}
	argsRaw, err := json.Marshal(GraphQLRequestArguments{Query: `subscription { noteUpdated { id } }`})
	require.NoError(t, err)

	resp := handleGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))
	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "subscription")
}
