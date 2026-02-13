package frontmatterpatch

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
)

// ValidateJsonnet validates jsonnet code by evaluating it with test data.
func ValidateJsonnet(jsonnetBody string) error {
	vm := NewVM()
	testMeta := map[string]interface{}{"title": "test"}
	testPath := "test/page.md"

	_, err := Evaluate(vm, CompiledPatch{
		ID:            0,
		WrappedSource: WrapSource(jsonnetBody),
	}, testMeta, testPath)

	return err
}

// ValidatePatterns validates glob patterns for syntax errors.
func ValidatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		_, err := doublestar.Match(pattern, "test")
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
	}
	return nil
}
