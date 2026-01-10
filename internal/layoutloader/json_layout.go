package layoutloader

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// JSONLayout represents the root structure of .html.json files.
type JSONLayout struct {
	Meta map[string]any `json:"meta"`
	Body []JSONNode     `json:"body"`
}

// JSONNode represents a single node in the layout tree.
type JSONNode struct {
	Type string `json:"type"`

	// block, import
	Name string `json:"name,omitempty"`

	// block
	Args map[string]any `json:"args,omitempty"`

	// block, if, range
	Content []JSONNode `json:"content,omitempty"`

	// if
	Condition string `json:"condition,omitempty"`

	// note_content, asset
	Path string `json:"path,omitempty"`

	// html
	HTML string `json:"html,omitempty"`

	// expr
	Expr string `json:"expr,omitempty"`

	// range
	Iterator   string `json:"iterator,omitempty"`
	Collection string `json:"collection,omitempty"`
}

// ConvertError represents an error during JSON layout conversion.
type ConvertError struct {
	Path    string // JSON path: "body[2].content[0]"
	Type    string // node type that caused the error
	Field   string // problematic field
	Message string // human-readable description
}

func (e *ConvertError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: type '%s' %s (field: %s)", e.Path, e.Type, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

var validNodeTypes = []string{"block", "if", "range", "expr", "html", "asset", "note_content", "include_note", "import"}

// ConvertOptions contains options for JSON layout conversion.
type ConvertOptions struct {
	// Preview wraps each node with a div containing mol_view_root attribute
	// for visual editor integration.
	Preview bool
}

// converter holds state during JSON to Jet conversion.
type converter struct {
	sb         strings.Builder
	varCounter int
	preview    bool
}

// ConvertJSONLayout converts JSON layout to Jet template string.
func ConvertJSONLayout(jsonContent []byte) (string, error) {
	return ConvertJSONLayoutWithOptions(jsonContent, ConvertOptions{})
}

// ConvertJSONLayoutWithOptions converts JSON layout to Jet template string with options.
func ConvertJSONLayoutWithOptions(jsonContent []byte, opts ConvertOptions) (string, error) {
	var layout JSONLayout
	err := json.Unmarshal(jsonContent, &layout)
	if err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	return ConvertJSONLayoutFromStruct(layout, opts)
}

// ConvertJSONLayoutFromStruct converts a parsed JSONLayout to Jet template string.
func ConvertJSONLayoutFromStruct(layout JSONLayout, opts ConvertOptions) (string, error) {
	c := &converter{preview: opts.Preview}
	err := c.convertNodes(layout.Body, "body")
	if err != nil {
		return "", err
	}

	return c.sb.String(), nil
}

func (c *converter) convertNodes(nodes []JSONNode, basePath string) error {
	for i, node := range nodes {
		path := fmt.Sprintf("%s[%d]", basePath, i)
		err := c.convertNode(node, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *converter) convertNode(node JSONNode, path string) error {
	if c.preview {
		c.writePreviewOpen(node, path)
	}

	var err error
	switch node.Type {
	case "block":
		err = c.convertBlock(node, path)
	case "if":
		err = c.convertIf(node, path)
	case "range":
		err = c.convertRange(node, path)
	case "expr":
		err = c.convertExpr(node, path)
	case "html":
		err = c.convertHTML(node)
	case "asset":
		err = c.convertAsset(node, path)
	case "note_content":
		err = c.convertNoteContent(node)
	case "include_note":
		err = c.convertIncludeNote(node, path)
	case "import":
		err = c.convertImport(node, path)
	case "":
		return &ConvertError{
			Path:    path,
			Message: "missing 'type' field",
		}
	default:
		return &ConvertError{
			Path:    path,
			Type:    node.Type,
			Message: fmt.Sprintf("unknown type '%s', expected one of: %s", node.Type, strings.Join(validNodeTypes, ", ")),
		}
	}

	if err != nil {
		return err
	}

	if c.preview {
		c.writePreviewClose()
	}

	return nil
}

func (c *converter) writePreviewOpen(node JSONNode, path string) {
	c.sb.WriteString(`<div id="jlb-`)
	c.sb.WriteString(normalizePathToID(path))
	c.sb.WriteString(`" class="json-layout-block">{{ layoutBlockEditor("`)
	c.sb.WriteString(path)
	c.sb.WriteString(`", "`)
	c.sb.WriteString(node.Type)
	c.sb.WriteString(`"`)
	if node.Name != "" {
		c.sb.WriteString(`, "`)
		c.sb.WriteString(node.Name)
		c.sb.WriteString(`"`)
	}
	c.sb.WriteString(`) | unsafe }}`)
}

func (c *converter) writePreviewClose() {
	c.sb.WriteString(`</div>`)
}

// normalizePathToID converts a JSON path to a valid HTML id.
// "body[0]" → "body-0"
// "body[0].content[1]" → "body-0-content-1"
func normalizePathToID(path string) string {
	result := strings.ReplaceAll(path, "[", "-")
	result = strings.ReplaceAll(result, "]", "")
	result = strings.ReplaceAll(result, ".", "-")
	return result
}

// convertBlock: {{ yield name(args...) }} or {{ yield name(args...) content }}...{{ end }}
func (c *converter) convertBlock(node JSONNode, path string) error {
	if node.Name == "" {
		return &ConvertError{
			Path:    path,
			Type:    "block",
			Field:   "name",
			Message: "requires 'name' field",
		}
	}

	c.sb.WriteString("{{ yield ")
	c.sb.WriteString(node.Name)
	c.sb.WriteString("(")

	// Sort args keys for deterministic output
	if len(node.Args) > 0 {
		keys := make([]string, 0, len(node.Args))
		for k := range node.Args {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				c.sb.WriteString(", ")
			}
			c.sb.WriteString(k)
			c.sb.WriteString("=")
			c.sb.WriteString(formatArg(node.Args[k]))
		}
	}

	c.sb.WriteString(")")

	if len(node.Content) > 0 {
		c.sb.WriteString(" content }}")
		err := c.convertNodes(node.Content, path+".content")
		if err != nil {
			return err
		}
		c.sb.WriteString("{{ end }}")
	} else {
		c.sb.WriteString(" }}")
	}

	return nil
}

// convertIf: {{ if condition }}...{{ end }}
func (c *converter) convertIf(node JSONNode, path string) error {
	if node.Condition == "" {
		return &ConvertError{
			Path:    path,
			Type:    "if",
			Field:   "condition",
			Message: "requires 'condition' field",
		}
	}

	c.sb.WriteString("{{ if ")
	c.sb.WriteString(node.Condition)
	c.sb.WriteString(" }}")

	if len(node.Content) > 0 {
		err := c.convertNodes(node.Content, path+".content")
		if err != nil {
			return err
		}
	}

	c.sb.WriteString("{{ end }}")
	return nil
}

// convertRange: {{ range iterator := collection }}...{{ end }}
func (c *converter) convertRange(node JSONNode, path string) error {
	if node.Collection == "" {
		return &ConvertError{
			Path:    path,
			Type:    "range",
			Field:   "collection",
			Message: "requires 'collection' field",
		}
	}

	c.sb.WriteString("{{ range ")

	if node.Iterator != "" {
		c.sb.WriteString(node.Iterator)
		c.sb.WriteString(" := ")
	}

	c.sb.WriteString(node.Collection)
	c.sb.WriteString(" }}")

	if len(node.Content) > 0 {
		err := c.convertNodes(node.Content, path+".content")
		if err != nil {
			return err
		}
	}

	c.sb.WriteString("{{ end }}")
	return nil
}

// convertExpr: {{ expr }}
func (c *converter) convertExpr(node JSONNode, path string) error {
	if node.Expr == "" {
		return &ConvertError{
			Path:    path,
			Type:    "expr",
			Field:   "expr",
			Message: "requires 'expr' field",
		}
	}

	c.sb.WriteString("{{ ")
	c.sb.WriteString(node.Expr)
	c.sb.WriteString(" }}")
	return nil
}

// convertHTML: raw HTML content
func (c *converter) convertHTML(node JSONNode) error {
	c.sb.WriteString(node.HTML)
	return nil
}

// convertAsset: {{ asset("path") }}
func (c *converter) convertAsset(node JSONNode, path string) error {
	if node.Path == "" {
		return &ConvertError{
			Path:    path,
			Type:    "asset",
			Field:   "path",
			Message: "requires 'path' field",
		}
	}

	c.sb.WriteString("{{ asset(\"")
	c.sb.WriteString(node.Path)
	c.sb.WriteString("\") }}")
	return nil
}

// convertNoteContent: {{ note.HTMLString() | unsafe }} or {{ nvs.ByPath("path").HTMLString() | unsafe }}
func (c *converter) convertNoteContent(node JSONNode) error {
	if node.Path == "" {
		c.sb.WriteString("{{ note.HTMLString() | unsafe }}")
		return nil
	}

	c.sb.WriteString("{{ nvs.ByPath(\"")
	c.sb.WriteString(node.Path)
	c.sb.WriteString("\").HTMLString() | unsafe }}")
	return nil
}

// convertIncludeNote: {{ _var := nvs.ByPath("path") }}{{ if _var }}{{ _var.HTMLString() | unsafe }}{{ else }}Create file: path{{ end }}
func (c *converter) convertIncludeNote(node JSONNode, path string) error {
	if node.Path == "" {
		return &ConvertError{
			Path:    path,
			Type:    "include_note",
			Field:   "path",
			Message: "requires 'path' field",
		}
	}

	varName := fmt.Sprintf("_note%d", c.varCounter)
	c.varCounter++

	c.sb.WriteString("{{ ")
	c.sb.WriteString(varName)
	c.sb.WriteString(" := nvs.ByPath(\"")
	c.sb.WriteString(node.Path)
	c.sb.WriteString("\") }}{{ if ")
	c.sb.WriteString(varName)
	c.sb.WriteString(" }}{{ ")
	c.sb.WriteString(varName)
	c.sb.WriteString(".HTMLString() | unsafe }}{{ else }}Create file: ")
	c.sb.WriteString(node.Path)
	c.sb.WriteString("{{ end }}")

	return nil
}

// convertImport: {{ import "name" }}
func (c *converter) convertImport(node JSONNode, path string) error {
	if node.Name == "" {
		return &ConvertError{
			Path:    path,
			Type:    "import",
			Field:   "name",
			Message: "requires 'name' field",
		}
	}

	c.sb.WriteString("{{ import \"")
	c.sb.WriteString(node.Name)
	c.sb.WriteString("\" }}")
	return nil
}

// formatArg converts a value to Jet argument format.
func formatArg(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", val)
	case float64:
		// JSON numbers are float64
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
