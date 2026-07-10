package webhookutil

import (
	"encoding/json"
	"fmt"
	"strings"

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

// MatchesAny checks if path matches any of the glob patterns. Patterns are
// normalized first (leading "/" and "./" stripped, same semantics as
// agentruntime's scope-pattern normalization) so a stored pattern like
// "/concepts/**" matches the canonical slash-less note paths. A pattern that
// is empty after stripping matches nothing.
func MatchesAny(path string, patterns []string) bool {
	for _, p := range patterns {
		p = normalizePattern(p)
		if p == "" {
			continue
		}
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

// normalizePattern strips leading "/" and "./" segments (repeatedly, since
// "/./"-style prefixes can stack) so patterns compare against the slash-less
// canonical paths. Globs are not path.Clean-ed to keep segments like "**" intact.
func normalizePattern(p string) string {
	p = strings.TrimSpace(p)
	for {
		switch {
		case strings.HasPrefix(p, "./"):
			p = p[2:]
		case strings.HasPrefix(p, "/"):
			p = p[1:]
		default:
			return p
		}
	}
}
