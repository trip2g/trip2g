package webhookutil

import (
	"encoding/json"
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
)

// ParseJSONStringArray parses a JSON string array like '["blog/**","docs/*"]'.
func ParseJSONStringArray(raw string) ([]string, error) {
	var result []string

	err := json.Unmarshal([]byte(raw), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON string array: %w", err)
	}

	return result, nil
}

// MatchesAny checks if path matches any of the glob patterns.
func MatchesAny(path string, patterns []string) bool {
	for _, p := range patterns {
		matched, err := doublestar.Match(p, path)
		if err != nil {
			// Invalid pattern — skip it.
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
