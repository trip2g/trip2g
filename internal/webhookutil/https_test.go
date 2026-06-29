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
