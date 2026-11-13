package model_test

import (
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestSiteOriginIsSchemeHostAndANonDefaultPort(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "https://hub.test", want: "https://hub.test", ok: true},
		{raw: "HTTPS://Hub.Test:443/base/", want: "https://hub.test", ok: true},
		{raw: "http://hub.test:80", want: "http://hub.test", ok: true},
		{raw: "http://localhost:18081", want: "http://localhost:18081", ok: true},
		{raw: "https://hub.test:8443", want: "https://hub.test:8443", ok: true},
		{raw: "null", ok: false},
		{raw: "", ok: false},
		{raw: "ftp://hub.test", ok: false},
		{raw: "//hub.test", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, ok := model.SiteOrigin(tt.raw)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

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
