package mdloader

import (
	"bytes"
	"errors"

	"github.com/yuin/goldmark/ast"
)

// toStringSlice converts a YAML-parsed interface{} to []string.
// Handles []interface{} (common from YAML parsers) and []string.
func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	}

	return nil
}

// toInt converts a YAML-parsed interface{} to int. Returns 0 for nil or non-numeric.
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	}

	return 0
}

// extractJsonnetBlocks walks a goldmark AST and collects the content of all
// fenced code blocks with language "jsonnet".
func extractJsonnetBlocks(doc ast.Node, source []byte) ([]string, error) {
	var blocks []string

	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		fcb, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Check language info string.
		info := fcb.Info
		if info == nil {
			return ast.WalkContinue, nil
		}

		lang := string(info.Segment.Value(source))
		if lang != "jsonnet" {
			return ast.WalkContinue, nil
		}

		// Extract content from lines.
		var buf bytes.Buffer
		lines := fcb.Lines()
		for i := range lines.Len() {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}

		body := bytes.TrimSpace(buf.Bytes())
		if len(body) > 0 {
			blocks = append(blocks, string(body))
		}

		return ast.WalkContinue, nil
	})

	if len(blocks) == 0 {
		return nil, errors.New("no jsonnet code blocks found")
	}

	return blocks, nil
}
