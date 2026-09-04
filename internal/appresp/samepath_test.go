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
		{"protocol relative", "//evil.com", "/"},
		{"protocol relative with path", "//evil.com/steal", "/"},
		{"backslash browsers normalise", "/\\evil.com", "/"},
		{"absolute", "https://evil.com/steal", "/"},
		{"no leading slash", "evil.com", "/"},
		{"scheme relative word", "relative/path", "/"},
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
