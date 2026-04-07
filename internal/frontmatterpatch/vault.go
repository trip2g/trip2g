package frontmatterpatch

import (
	"fmt"
	"hash/fnv"
	"strconv"
)

// CompileVaultPatches creates compiled patches from vault file fields.
// Each jsonnet body becomes a separate patch sharing the same include/exclude/priority.
// IDs are deterministic negative integers derived from path + block index.
func CompileVaultPatches(path string, includePatterns, excludePatterns []string, jsonnetBodies []string, priority int) ([]CompiledPatch, error) {
	if len(includePatterns) == 0 {
		return nil, fmt.Errorf("include patterns required")
	}

	if err := ValidatePatterns(includePatterns); err != nil {
		return nil, fmt.Errorf("include: %w", err)
	}

	if err := ValidatePatterns(excludePatterns); err != nil {
		return nil, fmt.Errorf("exclude: %w", err)
	}

	if len(jsonnetBodies) == 0 {
		return nil, fmt.Errorf("no jsonnet blocks found")
	}

	patches := make([]CompiledPatch, 0, len(jsonnetBodies))

	for i, body := range jsonnetBodies {
		if err := ValidateJsonnet(body); err != nil {
			return nil, fmt.Errorf("jsonnet block #%d: %w", i, err)
		}

		id := vaultPatchID(path, i)
		desc := path
		if len(jsonnetBodies) > 1 {
			desc = path + "#" + strconv.Itoa(i)
		}

		patches = append(patches, Compile(id, includePatterns, excludePatterns, body, priority, desc))
	}

	return patches, nil
}

// vaultPatchID generates a deterministic negative ID from path and block index.
// Always returns a value in [-2147483648, -1].
func vaultPatchID(path string, blockIndex int) int {
	h := fnv.New32a()
	h.Write([]byte(path + "#" + strconv.Itoa(blockIndex)))
	return -(int(h.Sum32()&0x7FFFFFFF) + 1)
}
