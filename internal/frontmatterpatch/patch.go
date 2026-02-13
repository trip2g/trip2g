package frontmatterpatch

// CompiledPatch represents a compiled frontmatter patch ready for evaluation.
type CompiledPatch struct {
	ID              int
	IncludePatterns []string
	ExcludePatterns []string
	Priority        int
	Description     string
	WrappedSource   string // full jsonnet with auto-wrapping
}

// AppliedPatch represents a patch that was successfully applied.
type AppliedPatch struct {
	PatchID     int
	Description string
}

// ApplyResult represents the result of applying patches to frontmatter.
type ApplyResult struct {
	RawMeta        map[string]interface{}
	AppliedPatches []AppliedPatch
	Warnings       []string // Runtime errors that were handled gracefully
}

// Compile creates a compiled patch with auto-wrapped jsonnet source.
func Compile(id int, includePatterns, excludePatterns []string, jsonnetBody string, priority int, description string) CompiledPatch {
	return CompiledPatch{
		ID:              id,
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
		Priority:        priority,
		Description:     description,
		WrappedSource:   WrapSource(jsonnetBody),
	}
}

// WrapSource wraps jsonnet body with meta and path external variables.
func WrapSource(jsonnetBody string) string {
	return `local meta = std.parseJson(std.extVar("meta"));
local path = std.extVar("path");
` + jsonnetBody
}
