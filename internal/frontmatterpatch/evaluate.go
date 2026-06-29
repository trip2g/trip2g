package frontmatterpatch

import (
	"encoding/json"
	"fmt"

	jsonnet "github.com/google/go-jsonnet"

	"trip2g/internal/jsonneteval"
)

// NewVM creates a new jsonnet VM with safe stack limits.
// Delegates to jsonneteval so the MaxStack limit has one source of truth.
func NewVM() *jsonnet.VM {
	return jsonneteval.NewVM()
}

// normalizeYAML converts yaml.v2-style map[interface{}]interface{} values
// (produced by goldmark-meta for nested frontmatter blocks) into
// map[string]interface{} so the meta can be JSON-marshalled.
func normalizeYAML(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, item := range val {
			m[fmt.Sprint(k)] = normalizeYAML(item)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, item := range val {
			m[k] = normalizeYAML(item)
		}
		return m
	case []interface{}:
		s := make([]interface{}, len(val))
		for i, item := range val {
			s[i] = normalizeYAML(item)
		}
		return s
	default:
		return v
	}
}

// Evaluate evaluates a compiled patch against raw frontmatter.
func Evaluate(vm *jsonnet.VM, patch CompiledPatch, rawMeta map[string]interface{}, path string) (map[string]interface{}, error) {
	// Marshal meta to JSON string for ExtVar
	metaJSON, err := json.Marshal(normalizeYAML(rawMeta))
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}

	vm.ExtVar("meta", string(metaJSON))
	vm.ExtVar("path", path)

	// Evaluate jsonnet
	result, err := vm.EvaluateAnonymousSnippet("patch", patch.WrappedSource)
	if err != nil {
		return nil, fmt.Errorf("evaluate jsonnet: %w", err)
	}

	// Unmarshal result
	var merged map[string]interface{}
	err = json.Unmarshal([]byte(result), &merged)
	if err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return merged, nil
}

// ApplyPatches applies multiple patches to frontmatter in priority order.
func ApplyPatches(vm *jsonnet.VM, patches []CompiledPatch, path string, rawMeta map[string]interface{}) ApplyResult {
	if rawMeta == nil {
		rawMeta = map[string]interface{}{}
	}

	result := ApplyResult{
		RawMeta:        rawMeta,
		AppliedPatches: []AppliedPatch{},
		Warnings:       []string{},
	}

	for _, patch := range patches {
		// Check if path matches patterns
		if !MatchPath(patch, path) {
			continue
		}

		// Evaluate patch
		merged, err := Evaluate(vm, patch, result.RawMeta, path)
		if err != nil {
			// Runtime error - add warning, don't fail
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Patch %d (%s) failed: %v", patch.ID, patch.Description, err))
			continue
		}

		// Shallow merge
		for k, v := range merged {
			result.RawMeta[k] = v
		}

		// Track applied patch
		result.AppliedPatches = append(result.AppliedPatches, AppliedPatch{
			PatchID:     patch.ID,
			Description: patch.Description,
		})
	}

	return result
}
