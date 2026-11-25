package rendernotepage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrailingSlashRedirectURL_RejectsHostileTargets(t *testing.T) {
	for _, notePath := range []string{"//x", "///x", "\\x", "", "/", "http://x", "x", "//x/y"} {
		t.Run(notePath, func(t *testing.T) {
			require.Empty(t, trailingSlashRedirectURL(notePath, nil))
		})
	}
}

func TestTrailingSlashRedirectURL_EscapesInsteadOfPassingThrough(t *testing.T) {
	for _, notePath := range []string{"/\\x", "/a\r\nb", "/a\nb", "/a\rb"} {
		t.Run(notePath, func(t *testing.T) {
			target := trailingSlashRedirectURL(notePath, nil)

			require.NotEmpty(t, target)
			require.Equal(t, byte('/'), target[0])
			require.NotContains(t, []byte{'/', '\\'}, target[1])
			require.False(t, strings.ContainsAny(target, "\r\n\\"))
		})
	}
}

func TestTrailingSlashRedirectURL_Query(t *testing.T) {
	require.Equal(t, "/blog?a=1&b=2", trailingSlashRedirectURL("/blog", []byte("a=1&b=2")))
	require.Equal(t, "/blog", trailingSlashRedirectURL("/blog", []byte("a=1\r\nX: y")))
	require.Equal(t, "/blog", trailingSlashRedirectURL("/blog", []byte("a=1\nb")))
	require.Equal(t, "/blog", trailingSlashRedirectURL("/blog", nil))
}
