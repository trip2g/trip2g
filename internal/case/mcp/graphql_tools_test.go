package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
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
func (e *gqlRequestEnv) Features() features.Features                      { panic("unexpected") }
func (e *gqlRequestEnv) LatestNoteViews() *model.NoteViews                { panic("unexpected") }
func (e *gqlRequestEnv) LatestNoteChunks() []model.NoteChunk              { panic("unexpected") }
func (e *gqlRequestEnv) CanReadNote(_ context.Context, _ *model.NoteView) (bool, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) FederationClient(_ context.Context, _ string) (model.Federation, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) SearchLatestNotes(_ string) ([]model.SearchResult, error) {
	panic("unexpected")
}
func (e *gqlRequestEnv) OpenAI() *openai.Client           { panic("unexpected") }
func (e *gqlRequestEnv) PublicURL() string                { panic("unexpected") }
func (e *gqlRequestEnv) NoteURL(_ *model.NoteView) string { panic("unexpected") }
func (e *gqlRequestEnv) Logger() logger.Logger            { return &logger.DummyLogger{} }
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
func (e *gqlRequestEnv) ResolveAPIKey(_ context.Context, _, _ string) (*db.ApiKey, error) {
	panic("unexpected")
}

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
	var want any
	require.NoError(t, json.Unmarshal(canned, &want))
	wantJSON, err := json.Marshal(want)
	require.NoError(t, err)
	gotJSON, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.Equal(t, string(wantJSON), string(gotJSON))
}

// --- validateReadOnlyQuery tests ---

func TestValidateReadOnlyQuery_RejectsMutation(t *testing.T) {
	err := validateReadOnlyQuery(`mutation { createNote { id } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutation")
}

func TestValidateReadOnlyQuery_RejectsDisallowedRootField(t *testing.T) {
	err := validateReadOnlyQuery(`{ admin { users { id } } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin")
}

func TestValidateReadOnlyQuery_RejectsMultiOpWithMutation(t *testing.T) {
	err := validateReadOnlyQuery(`query A { note(path: "x") { title } } mutation B { createNote { id } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutation")
}

func TestValidateReadOnlyQuery_AllowsNote(t *testing.T) {
	err := validateReadOnlyQuery(`{ note(path: "x") { title } }`)
	require.NoError(t, err)
}

func TestValidateReadOnlyQuery_AllowsSearch(t *testing.T) {
	err := validateReadOnlyQuery(`{ search(query: "y") { title notePath } }`)
	require.NoError(t, err)
}

func TestValidateReadOnlyQuery_RejectsSubscription(t *testing.T) {
	err := validateReadOnlyQuery(`subscription { noteUpdated { id } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "subscription")
}

func TestValidateReadOnlyQuery_RejectsPublicUrl(t *testing.T) {
	err := validateReadOnlyQuery(`{ publicUrl }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "publicUrl")
}

func TestValidateReadOnlyQuery_ParseError(t *testing.T) {
	err := validateReadOnlyQuery(`{{{not valid graphql`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse error")
}

// --- validateReadOnlyQuery guard-bypass regression tests ---
// These vectors are security-relevant: they confirm that inline fragments,
// fragment spreads, and field aliases cannot be used to bypass the allowlist.

func TestValidateReadOnlyQuery_RejectsRootInlineFragment(t *testing.T) {
	// Inline fragment at root level must be rejected — the selector is not a *ast.Field.
	err := validateReadOnlyQuery(`query { ... on Query { admin { id } } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only field selections are allowed at query root")
}

func TestValidateReadOnlyQuery_RejectsRootFragmentSpread(t *testing.T) {
	// Fragment spread at root level must be rejected — same reason.
	err := validateReadOnlyQuery(`query Q { ...F } fragment F on Query { admin { id } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only field selections are allowed at query root")
}

func TestValidateReadOnlyQuery_RejectsAliasedBannedField(t *testing.T) {
	// Field.Name (not Field.Alias) is checked; an alias must not bypass the allowlist.
	err := validateReadOnlyQuery(`query { n: admin { id } }`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin")
}

func TestValidateReadOnlyQuery_AllowsAliasedAllowedField(t *testing.T) {
	// An alias on an allowed field must be accepted.
	err := validateReadOnlyQuery(`query { n: note(path:"x") { title } }`)
	require.NoError(t, err)
}

// --- handleGraphQLRequest Option A edge-case tests ---

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

// TestHandleGraphQLRequest_MutationRejectedByHandler: handler-level check that a
// mutation query yields an InvalidParams error without reaching the env.
func TestHandleGraphQLRequest_MutationRejectedByHandler(t *testing.T) {
	env := &gqlRequestEnv{
		graphqlFunc: func(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
			panic("must not be called")
		},
	}
	argsRaw, err := json.Marshal(GraphQLRequestArguments{Query: `mutation { createNote { id } }`})
	require.NoError(t, err)

	resp := handleGraphQLRequest(context.Background(), env, 1, json.RawMessage(argsRaw))
	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "mutation")
}
