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
func (e *gqlRequestEnv) OpenAI() *openai.Client                      { panic("unexpected") }
func (e *gqlRequestEnv) PublicURL() string                           { panic("unexpected") }
func (e *gqlRequestEnv) NoteURL(_ *model.NoteView) string            { panic("unexpected") }
func (e *gqlRequestEnv) Logger() logger.Logger                       { panic("unexpected") }
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
