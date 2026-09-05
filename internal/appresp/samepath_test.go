package appresp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSamePath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "/"},
		{"root", "/", "/"},
		{"path", "/boards/x", "/boards/x"},
		{"path with query", "/boards/x?tab=2", "/boards/x?tab=2"},
		{"path with fragment", "/boards/x#!userspace=open", "/boards/x#!userspace=open"},
		{"escaped backslash stays ours", "/%5Cevil.com", "/%5Cevil.com"},
		{"escaped slash stays ours", "/%2F/evil.com", "/%2F/evil.com"},
		{"protocol relative", "//evil.com", "/"},
		{"protocol relative with path", "//evil.com/steal", "/"},
		{"three slashes", "///evil.com", "/"},
		{"backslash is escaped, not passed on", "/\\evil.com", "/%5Cevil.com"},
		{"space is escaped", "/ /evil.com", "/%20/evil.com"},
		{"absolute", "https://evil.com/steal", "/"},
		{"opaque scheme", "javascript:alert(1)", "/"},
		{"no leading slash", "evil.com", "/"},
		{"scheme relative word", "relative/path", "/"},
		// A browser strips ASCII tab, LF and CR from a URL before it parses it,
		// so any of them beside the leading slash turns the value back into
		// "//evil.com" in the one place it matters. CR LF in a header value is
		// worse than a redirect. Refused as a class by parsing, rather than
		// enumerated one character at a time.
		{"tab the browser would strip", "/\t/evil.com", "/"},
		{"newline the browser would strip", "/\n/evil.com", "/"},
		{"header injection", "/\r\nSet-Cookie: a=1", "/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, SamePath(tc.in))
		})
	}
}

// An absolute-form request line is accepted by the server, and RequestURI()
// then hands back the path of that URL — which can itself be another origin.
// This is the shape that reached a Location header twice.
func TestSamePathRefusesAnAbsoluteFormRequestLine(t *testing.T) {
	parsed, err := url.Parse("http://ourhost//evil.com")
	require.NoError(t, err)
	require.Equal(t, "//evil.com", parsed.RequestURI())

	require.Equal(t, "/", SamePath(parsed.RequestURI()))
}
