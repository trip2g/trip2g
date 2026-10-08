package mcp_test

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/features"
	appmodel "trip2g/internal/model"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

type listedTool struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}

func listEveryTool(t *testing.T) []listedTool {
	t.Helper()

	env := buildDispatchEnv(t, false)
	adminTools := true
	env.ResolveAPIKeyFunc = func(context.Context, string, string) (*db.ApiKey, error) {
		return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
	}
	env.FederatedGraphQLEnabledFunc = func() bool {
		return true
	}
	env.FeaturesFunc = func() features.Features {
		var f features.Features
		f.VectorSearch.Reranker.Enabled = true
		return f
	}
	noteTool := &appmodel.NoteView{Path: "tools/hello.md", Title: "Hello", MCPMethod: "hello", Free: true}
	env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
		return &appmodel.NoteViews{
			List:    []*appmodel.NoteView{noteTool},
			PathMap: map[string]*appmodel.NoteView{noteTool.Path: noteTool},
		}
	}

	fasthttpCtx := buildMCPFasthttpCtx(mcpToolsListBody(t), "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "admin-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp struct {
		Result struct {
			Tools []listedTool `json:"tools"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}
	err = json.Unmarshal(fasthttpCtx.Response.Body(), &resp)
	require.NoError(t, err)
	require.Empty(t, resp.Error)

	return resp.Result.Tools
}

func TestToolsListAdvertisesEveryKindOfTool(t *testing.T) {
	names := make([]string, 0)
	for _, tool := range listEveryTool(t) {
		names = append(names, tool.Name)
	}

	for _, want := range []string{"expand", "federated_expand", "graphql_request", "federated_graphql_request", "hello"} {
		require.Contains(t, names, want)
	}
}

func TestEveryToolInputSchemaIsValidJSONSchema(t *testing.T) {
	for _, tool := range listEveryTool(t) {
		t.Run(tool.Name, func(t *testing.T) {
			require.NotNil(t, tool.InputSchema)

			compiler := jsonschema.NewCompiler()
			compiler.DefaultDraft(jsonschema.Draft2020)
			err := compiler.AddResource("inputSchema.json", tool.InputSchema)
			require.NoError(t, err)
			_, err = compiler.Compile("inputSchema.json")
			require.NoError(t, err)

			require.Equal(t, "object", tool.InputSchema["type"])
			require.Empty(t, schemaShapeProblems("inputSchema", tool.InputSchema))
		})
	}
}

func TestExpandTocPathIsAnArrayOfStrings(t *testing.T) {
	for _, tool := range listEveryTool(t) {
		if tool.Name != "expand" && tool.Name != "federated_expand" {
			continue
		}

		properties, ok := tool.InputSchema["properties"].(map[string]any)
		require.True(t, ok)

		tocPath := properties["toc_path"].(map[string]any)
		require.Equal(t, "array", tocPath["type"])
		require.Equal(t, map[string]any{"type": "string"}, tocPath["items"])

		for _, bound := range []string{"first", "last"} {
			require.Equal(t, "number", properties[bound].(map[string]any)["type"])
			require.NotContains(t, properties[bound], "items")
		}
	}
}

func allowedKeywords(kind string) ([]string, bool) {
	switch kind {
	case "object":
		return []string{"type", "description", "properties", "required", "additionalProperties"}, true
	case "array":
		return []string{"type", "description", "items"}, true
	case "string", "number", "integer", "boolean":
		return []string{"type", "description"}, true
	default:
		return nil, false
	}
}

func schemaShapeProblems(at string, schema map[string]any) []string {
	kind, _ := schema["type"].(string)
	allowed, known := allowedKeywords(kind)
	if !known {
		return []string{at + ": unknown type " + strconv.Quote(kind)}
	}

	var problems []string

	keys := make([]string, 0, len(schema))
	for key := range schema {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if !slices.Contains(allowed, key) {
			problems = append(problems, at+": "+key+" does not belong on type "+kind)
		}
	}

	if kind == "array" {
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return append(problems, at+": array without items")
		}
		problems = append(problems, schemaShapeProblems(at+".items", items)...)
	}

	if kind == "object" {
		properties, _ := schema["properties"].(map[string]any)
		names := make([]string, 0, len(properties))
		for name := range properties {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			property, ok := properties[name].(map[string]any)
			if !ok {
				problems = append(problems, at+".properties."+name+": not a schema")
				continue
			}
			problems = append(problems, schemaShapeProblems(at+".properties."+name, property)...)
		}
	}

	return problems
}
