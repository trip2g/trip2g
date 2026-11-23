package notepath_test

import (
	"testing"

	"trip2g/internal/notepath"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"a.md":            "a.md",
		"/a.md":           "a.md",
		"./a.md":          "a.md",
		"\ta/./b/../c\t":  "",
		" a.md":           "",
		"a.md ":           "",
		"a\\..\\..\\x.md": "",
		"a\\b.md":         "",
		"":                "",
		".":               "",
		"..":              "",
		"../x.md":         "",
		"a/../../x.md":    "",
		"/../x.md":        "",
		"a\x00b.md":       "",
		"..hidden/a.md":   "..hidden/a.md",
	}

	for raw, want := range cases {
		require.Equal(t, want, notepath.Normalize(raw), "path %q", raw)
	}
}
