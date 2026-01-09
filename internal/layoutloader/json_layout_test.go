package layoutloader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertJSONLayout_Block(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "block", "name": "header"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, "{{ yield header() }}", result)
}

func TestConvertJSONLayout_BlockWithArgs(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "block",
				"name": "cta_section",
				"args": {
					"title": "Ready?",
					"level": 2
				}
			}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	// Args are sorted alphabetically
	require.Equal(t, `{{ yield cta_section(level=2, title="Ready?") }}`, result)
}

func TestConvertJSONLayout_BlockWithContent(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "block",
				"name": "card",
				"args": {"title": "Hello"},
				"content": [
					{"type": "html", "html": "<p>Content</p>"}
				]
			}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ yield card(title="Hello") content }}<p>Content</p>{{ end }}`, result)
}

func TestConvertJSONLayout_If(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "if",
				"condition": "note.M().GetBool(\"show\")",
				"content": [
					{"type": "html", "html": "<div>Shown</div>"}
				]
			}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ if note.M().GetBool("show") }}<div>Shown</div>{{ end }}`, result)
}

func TestConvertJSONLayout_Range(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "range",
				"iterator": "i, post",
				"collection": "nvs.ByGlob(\"blog/*.md\").All()",
				"content": [
					{"type": "html", "html": "<li>"},
					{"type": "expr", "expr": "post.Title()"},
					{"type": "html", "html": "</li>"}
				]
			}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ range i, post := nvs.ByGlob("blog/*.md").All() }}<li>{{ post.Title() }}</li>{{ end }}`, result)
}

func TestConvertJSONLayout_RangeWithoutIterator(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "range",
				"collection": "items",
				"content": [
					{"type": "html", "html": "<li>item</li>"}
				]
			}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ range items }}<li>item</li>{{ end }}`, result)
}

func TestConvertJSONLayout_Expr(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "expr", "expr": "note.Title()"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, "{{ note.Title() }}", result)
}

func TestConvertJSONLayout_HTML(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "html", "html": "<div class=\"container\">"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `<div class="container">`, result)
}

func TestConvertJSONLayout_Asset(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "asset", "path": "style.css"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ asset("style.css") }}`, result)
}

func TestConvertJSONLayout_NoteContent(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "note_content"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, "{{ note.HTMLString() | unsafe }}", result)
}

func TestConvertJSONLayout_NoteContentWithPath(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "note_content", "path": "_sidebar.md"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ _note0 := nvs.ByPath("_sidebar.md") }}{{ if _note0 }}{{ _note0.HTMLString() | unsafe }}{{ end }}`, result)
}

func TestConvertJSONLayout_MultipleNoteContentWithPath(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "note_content", "path": "_header.md"},
			{"type": "note_content"},
			{"type": "note_content", "path": "_footer.md"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	// Each path-based note_content gets a unique variable
	require.Contains(t, result, "_note0")
	require.Contains(t, result, "_note1")
	require.Contains(t, result, "note.HTMLString()")
}

func TestConvertJSONLayout_Import(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "import", "name": "blocks"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, `{{ import "blocks" }}`, result)
}

func TestConvertJSONLayout_ComplexLayout(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "import", "name": "blocks"},
			{"type": "block", "name": "header", "args": {"level": 2}},
			{
				"type": "if",
				"condition": "note.M().GetBool(\"show_sidebar\")",
				"content": [
					{"type": "note_content", "path": "_sidebar.md"}
				]
			},
			{"type": "note_content"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)

	require.Contains(t, result, `{{ import "blocks" }}`)
	require.Contains(t, result, `{{ yield header(level=2) }}`)
	require.Contains(t, result, `{{ if note.M().GetBool("show_sidebar") }}`)
	require.Contains(t, result, `nvs.ByPath("_sidebar.md")`)
	require.Contains(t, result, `{{ note.HTMLString() | unsafe }}`)
}

// Error tests

func TestConvertJSONLayout_Error_InvalidJSON(t *testing.T) {
	json := `{invalid json`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JSON")
}

func TestConvertJSONLayout_Error_MissingType(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"name": "header"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "body[0]", convertErr.Path)
	require.Contains(t, convertErr.Message, "missing 'type'")
}

func TestConvertJSONLayout_Error_UnknownType(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "unknown_type"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "body[0]", convertErr.Path)
	require.Equal(t, "unknown_type", convertErr.Type)
	require.Contains(t, convertErr.Message, "unknown type")
	require.Contains(t, convertErr.Message, "block, if, range")
}

func TestConvertJSONLayout_Error_BlockMissingName(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "block"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "body[0]", convertErr.Path)
	require.Equal(t, "block", convertErr.Type)
	require.Equal(t, "name", convertErr.Field)
}

func TestConvertJSONLayout_Error_IfMissingCondition(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "if", "content": []}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "if", convertErr.Type)
	require.Equal(t, "condition", convertErr.Field)
}

func TestConvertJSONLayout_Error_RangeMissingCollection(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "range", "iterator": "i, item"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "range", convertErr.Type)
	require.Equal(t, "collection", convertErr.Field)
}

func TestConvertJSONLayout_Error_ExprMissingExpr(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "expr"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "expr", convertErr.Type)
	require.Equal(t, "expr", convertErr.Field)
}

func TestConvertJSONLayout_Error_AssetMissingPath(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "asset"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "asset", convertErr.Type)
	require.Equal(t, "path", convertErr.Field)
}

func TestConvertJSONLayout_Error_ImportMissingName(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "import"}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "import", convertErr.Type)
	require.Equal(t, "name", convertErr.Field)
}

func TestConvertJSONLayout_Error_NestedError(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "if",
				"condition": "true",
				"content": [
					{
						"type": "block",
						"name": "wrapper",
						"content": [
							{"type": "unknown"}
						]
					}
				]
			}
		]
	}`

	_, err := ConvertJSONLayout([]byte(json))
	require.Error(t, err)

	convertErr, ok := err.(*ConvertError)
	require.True(t, ok)
	require.Equal(t, "body[0].content[0].content[0]", convertErr.Path)
}

func TestConvertJSONLayout_BoolArgs(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{
				"type": "block",
				"name": "toggle",
				"args": {
					"enabled": true,
					"visible": false
				}
			}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Contains(t, result, "enabled=true")
	require.Contains(t, result, "visible=false")
}

func TestConvertJSONLayout_EmptyBody(t *testing.T) {
	json := `{
		"meta": {"title": "Empty"},
		"body": []
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)
	require.Equal(t, "", result)
}

// Test that generated Jet is valid by checking it doesn't have obvious syntax errors
func TestConvertJSONLayout_ValidJetSyntax(t *testing.T) {
	json := `{
		"meta": {},
		"body": [
			{"type": "import", "name": "blocks"},
			{"type": "html", "html": "<!DOCTYPE html><html>"},
			{"type": "block", "name": "head"},
			{"type": "html", "html": "<body>"},
			{
				"type": "range",
				"iterator": "i, post",
				"collection": "nvs.ByGlob(\"*.md\").All()",
				"content": [
					{"type": "expr", "expr": "post.Title()"}
				]
			},
			{"type": "note_content"},
			{"type": "html", "html": "</body></html>"}
		]
	}`

	result, err := ConvertJSONLayout([]byte(json))
	require.NoError(t, err)

	// Check balanced braces
	openBraces := strings.Count(result, "{{")
	closeBraces := strings.Count(result, "}}")
	require.Equal(t, openBraces, closeBraces, "Jet template should have balanced {{ and }}")

	// Check balanced end tags
	endCount := strings.Count(result, "{{ end }}")
	rangeCount := strings.Count(result, "{{ range")
	ifCount := strings.Count(result, "{{ if")
	contentCount := strings.Count(result, " content }}")

	require.Equal(t, rangeCount+ifCount+contentCount, endCount,
		"Each range/if/content block should have matching {{ end }}")
}
