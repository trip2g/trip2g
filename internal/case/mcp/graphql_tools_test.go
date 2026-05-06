package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
