package frontmatterpatch

import "github.com/bmatcuk/doublestar/v4"

// MatchPath checks if a path matches the patch's include/exclude patterns.
func MatchPath(patch CompiledPatch, path string) bool {
	// Must match at least one include pattern
	includeMatch := false
	for _, pattern := range patch.IncludePatterns {
		match, _ := doublestar.Match(pattern, path)
		if match {
			includeMatch = true
			break
		}
	}
	if !includeMatch {
		return false
	}

	// Must NOT match any exclude pattern
	for _, pattern := range patch.ExcludePatterns {
		match, _ := doublestar.Match(pattern, path)
		if match {
			return false
		}
	}

	return true
}
