package model_test

import (
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestMCPFederationSiteStripsTheMCPPath(t *testing.T) {
	tests := []struct {
		kbURL string
		want  string
		ok    bool
	}{
		{kbURL: "https://peer.test/_system/mcp", want: "https://peer.test", ok: true},
		{kbURL: "https://peer.test/_system/mcp/", want: "https://peer.test", ok: true},
		// Taken as given: an internal address yields an internal site address,
		// which a browser outside that network cannot open.
		{kbURL: "http://10.0.0.5:8081/_system/mcp", want: "http://10.0.0.5:8081", ok: true},
		{kbURL: "https://peer.test/mcp", ok: false},
		{kbURL: "ftp://peer.test/_system/mcp", ok: false},
		{kbURL: "/_system/mcp", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.kbURL, func(t *testing.T) {
			got, ok := model.MCPFederationSite(tt.kbURL)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}
