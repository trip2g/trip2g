package webhookutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequireHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https OK", "https://example.com/hook", false},
		{"http external rejects", "http://example.com/hook", true},
		{"http localhost allows", "http://localhost/hook", false},
		{"http 127.0.0.1 allows", "http://127.0.0.1/hook", false},
		{"http ::1 allows", "http://[::1]/hook", false},
		{"other scheme silent", "ftp://example.com", false},
		{"HTTP uppercase rejects", "HTTP://example.com/hook", true},
		{"HtTp mixed case rejects", "HtTp://example.com/hook", true},
		{"http with leading whitespace rejects", "  http://example.com/hook", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := RequireHTTPS(tt.url)
			if tt.wantErr {
				require.NotEmpty(t, msg)
			} else {
				require.Empty(t, msg)
			}
		})
	}
}

func TestRequireHTTPSUnlessDevMode(t *testing.T) {
	// Dev mode bypasses the check (Docker-internal compose URLs are not loopback).
	require.Empty(t, RequireHTTPSUnlessDevMode("http://fleet:9099/deliver", true))
	require.Empty(t, RequireHTTPSUnlessDevMode("http://example.com/hook", true))
	// Production (devMode=false) delegates to RequireHTTPS.
	require.NotEmpty(t, RequireHTTPSUnlessDevMode("http://example.com/hook", false))
	require.Empty(t, RequireHTTPSUnlessDevMode("https://example.com/hook", false))
	require.Empty(t, RequireHTTPSUnlessDevMode("http://localhost/hook", false))
}
