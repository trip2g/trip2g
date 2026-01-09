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

var validNodeTypes = []string{"block", "if", "range", "expr", "html", "asset", "note_content", "import"}

// ConvertJSONLayout converts JSON layout to Jet template string.
func ConvertJSONLayout(jsonContent []byte) (string, error) {
	var layout JSONLayout
	err := json.Unmarshal(jsonContent, &layout)
	if err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	var sb strings.Builder
	err = convertNodes(&sb, layout.Body, "body", 0)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func convertNodes(sb *strings.Builder, nodes []JSONNode, basePath string, varCounter int) error {
	for i, node := range nodes {
		path := fmt.Sprintf("%s[%d]", basePath, i)
		err := convertNode(sb, node, path, &varCounter)
		if err != nil {
			return err
		}
	}
	return nil
}

func convertNode(sb *strings.Builder, node JSONNode, path string, varCounter *int) error {
	switch node.Type {
	case "block":
		return convertBlock(sb, node, path, varCounter)
	case "if":
		return convertIf(sb, node, path, varCounter)
	case "range":
		return convertRange(sb, node, path, varCounter)
	case "expr":
		return convertExpr(sb, node, path)
	case "html":
		return convertHTML(sb, node, path)
	case "asset":
		return convertAsset(sb, node, path)
	case "note_content":
		return convertNoteContent(sb, node, path, varCounter)
	case "import":
		return convertImport(sb, node, path)
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
}

// convertBlock: {{ yield name(args...) }} or {{ yield name(args...) content }}...{{ end }}
func convertBlock(sb *strings.Builder, node JSONNode, path string, varCounter *int) error {
	if node.Name == "" {
		return &ConvertError{
			Path:    path,
			Type:    "block",
			Field:   "name",
			Message: "requires 'name' field",
		}
	}

	sb.WriteString("{{ yield ")
	sb.WriteString(node.Name)
	sb.WriteString("(")

	// Sort args keys for deterministic output
	if len(node.Args) > 0 {
		keys := make([]string, 0, len(node.Args))
		for k := range node.Args {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(formatArg(node.Args[k]))
		}
	}

	sb.WriteString(")")

	if len(node.Content) > 0 {
		sb.WriteString(" content }}")
		err := convertNodes(sb, node.Content, path+".content", *varCounter)
		if err != nil {
			return err
		}
		sb.WriteString("{{ end }}")
	} else {
		sb.WriteString(" }}")
	}

	return nil
}

// convertIf: {{ if condition }}...{{ end }}
func convertIf(sb *strings.Builder, node JSONNode, path string, varCounter *int) error {
	if node.Condition == "" {
		return &ConvertError{
			Path:    path,
			Type:    "if",
			Field:   "condition",
			Message: "requires 'condition' field",
		}
	}

	sb.WriteString("{{ if ")
	sb.WriteString(node.Condition)
	sb.WriteString(" }}")

	if len(node.Content) > 0 {
		err := convertNodes(sb, node.Content, path+".content", *varCounter)
		if err != nil {
			return err
		}
	}

	sb.WriteString("{{ end }}")
	return nil
}

// convertRange: {{ range iterator := collection }}...{{ end }}
func convertRange(sb *strings.Builder, node JSONNode, path string, varCounter *int) error {
	if node.Collection == "" {
		return &ConvertError{
			Path:    path,
			Type:    "range",
			Field:   "collection",
			Message: "requires 'collection' field",
		}
	}

	sb.WriteString("{{ range ")

	if node.Iterator != "" {
		sb.WriteString(node.Iterator)
		sb.WriteString(" := ")
	}

	sb.WriteString(node.Collection)
	sb.WriteString(" }}")

	if len(node.Content) > 0 {
		err := convertNodes(sb, node.Content, path+".content", *varCounter)
		if err != nil {
			return err
		}
	}

	sb.WriteString("{{ end }}")
	return nil
}

// convertExpr: {{ expr }}
func convertExpr(sb *strings.Builder, node JSONNode, path string) error {
	if node.Expr == "" {
		return &ConvertError{
			Path:    path,
			Type:    "expr",
			Field:   "expr",
			Message: "requires 'expr' field",
		}
	}

	sb.WriteString("{{ ")
	sb.WriteString(node.Expr)
	sb.WriteString(" }}")
	return nil
}

// convertHTML: raw HTML content
func convertHTML(sb *strings.Builder, node JSONNode, path string) error {
	sb.WriteString(node.HTML)
	return nil
}

// convertAsset: {{ asset("path") }}
func convertAsset(sb *strings.Builder, node JSONNode, path string) error {
	if node.Path == "" {
		return &ConvertError{
			Path:    path,
			Type:    "asset",
			Field:   "path",
			Message: "requires 'path' field",
		}
	}

	sb.WriteString("{{ asset(\"")
	sb.WriteString(node.Path)
	sb.WriteString("\") }}")
	return nil
}

// convertNoteContent: {{ note.HTMLString() | unsafe }} or {{ _var := nvs.ByPath("path") }}{{ if _var }}{{ _var.HTMLString() | unsafe }}{{ end }}
func convertNoteContent(sb *strings.Builder, node JSONNode, path string, varCounter *int) error {
	if node.Path == "" {
		// Current note content
		sb.WriteString("{{ note.HTMLString() | unsafe }}")
		return nil
	}

	// Include another note by path
	// Generate unique variable name
	varName := fmt.Sprintf("_note%d", *varCounter)
	*varCounter++

	sb.WriteString("{{ ")
	sb.WriteString(varName)
	sb.WriteString(" := nvs.ByPath(\"")
	sb.WriteString(node.Path)
	sb.WriteString("\") }}{{ if ")
	sb.WriteString(varName)
	sb.WriteString(" }}{{ ")
	sb.WriteString(varName)
	sb.WriteString(".HTMLString() | unsafe }}{{ end }}")

	return nil
}

// convertImport: {{ import "name" }}
func convertImport(sb *strings.Builder, node JSONNode, path string) error {
	if node.Name == "" {
		return &ConvertError{
			Path:    path,
			Type:    "import",
			Field:   "name",
			Message: "requires 'name' field",
		}
	}

	sb.WriteString("{{ import \"")
	sb.WriteString(node.Name)
	sb.WriteString("\" }}")
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
