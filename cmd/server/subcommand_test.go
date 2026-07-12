package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnrecognizedSubcommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantName string
		wantOK   bool
	}{
		{"no args", []string{"trip2g-server"}, "", false},
		{"flag invocation", []string{"trip2g-server", "-listen-addr=:8080"}, "", false},
		{"known: lint", []string{"trip2g-server", "lint", "docs"}, "", false},
		{"known: login-link", []string{"trip2g-server", "login-link"}, "", false},
		{"unknown bare word", []string{"trip2g-server", "bogus"}, "bogus", true},
		{"typo of login-link", []string{"trip2g-server", "login-lnk"}, "login-lnk", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOK := unrecognizedSubcommand(tt.args)
			require.Equal(t, tt.wantOK, gotOK)
			require.Equal(t, tt.wantName, gotName)
		})
	}
}
