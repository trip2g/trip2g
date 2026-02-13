package frontmatterpatch

import (
	"encoding/json"
	"fmt"

	jsonnet "github.com/google/go-jsonnet"
)

// NewVM creates a new jsonnet VM with safe stack limits.
func NewVM() *jsonnet.VM {
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500 // Prevent stack overflow from recursive jsonnet
	return vm
}

// Evaluate evaluates a compiled patch against raw frontmatter.
func Evaluate(vm *jsonnet.VM, patch CompiledPatch, rawMeta map[string]interface{}, path string) (map[string]interface{}, error) {
	// Marshal meta to JSON string for ExtVar
	metaJSON, err := json.Marshal(rawMeta)
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
