package oauthstate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeRedirect(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/dashboard", "/dashboard"},
		{"/settings/profile", "/settings/profile"},
		// absolute URLs must be rejected
		{"https://evil.com", "/"},
		{"http://evil.com/steal", "/"},
		// protocol-relative
		{"//evil.com", "/"},
		{"//evil.com/path", "/"},
		// backslash variants some browsers normalise
		{"/\\evil.com", "/"},
		// no leading slash
		{"evil.com", "/"},
		{"relative/path", "/"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, safeRedirect(tc.in))
		})
	}
}
