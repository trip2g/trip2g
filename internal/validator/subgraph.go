package validator

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

var ErrNameOnlyValid = errors.New("name must be only [a-zA-Z0-9_]")

var subgraphNamesRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func NormalizeSubgraphNames(input string) (string, error) {
	// split by |
	parts := strings.Split(input, "|")

	for i, part := range parts {
		part = strings.TrimSpace(part)

		if !subgraphNamesRegexp.MatchString(part) {
			return "", ErrNameOnlyValid
		}

		parts[i] = part
	}

	sort.StringSlice(parts).Sort()

	return strings.Join(parts, "|"), nil
}
