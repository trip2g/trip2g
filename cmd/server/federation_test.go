package main

import (
	"testing"
	"time"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

func secretRow(kid, kbURL string, revoked bool) db.ListFederationSecretsRow {
	row := db.ListFederationSecretsRow{Kid: kid}
	if kbURL != "" {
		row.KbUrl = &kbURL
	}
	if revoked {
		at := time.Unix(1, 0)
		row.RevokedAt = &at
	}
	return row
}

func TestPickOutboundKID(t *testing.T) {
	const peer = "https://keeper.example/_system/mcp"

	tests := []struct {
		name    string
		rows    []db.ListFederationSecretsRow
		kbID    string
		wantKID string
		wantOK  bool
		wantErr string
	}{
		{
			name: "the key named like the kb_id wins over a newer key to the same peer",
			rows: []db.ListFederationSecretsRow{
				secretRow("masha-kp-writer", peer, false),
				secretRow("masha-baserole", peer, false),
			},
			kbID:    "masha-baserole",
			wantKID: "masha-baserole",
			wantOK:  true,
		},
		{
			name:    "a single live key serves a kb_id that names no key",
			rows:    []db.ListFederationSecretsRow{secretRow("partner-2026", peer, false)},
			kbID:    "keeper.example",
			wantKID: "partner-2026",
			wantOK:  true,
		},
		{
			name: "a revoked key named like the kb_id is an error, never a fall back to another key",
			rows: []db.ListFederationSecretsRow{
				secretRow("masha-baserole", peer, true),
				secretRow("partner-2026", peer, false),
			},
			kbID:    "masha-baserole",
			wantErr: "is revoked",
		},
		{
			name: "keys to other peers and inbound keys are ignored",
			rows: []db.ListFederationSecretsRow{
				secretRow("other", "https://other.example/_system/mcp", false),
				secretRow("inbound", "", false),
				secretRow("partner-2026", peer, false),
			},
			kbID:    "keeper.example",
			wantKID: "partner-2026",
			wantOK:  true,
		},
		{
			name:   "no key for the peer means an unauthenticated peer",
			rows:   []db.ListFederationSecretsRow{secretRow("other", "https://other.example/_system/mcp", false)},
			kbID:   "keeper.example",
			wantOK: false,
		},
		{
			name:    "only revoked keys for the peer is an error",
			rows:    []db.ListFederationSecretsRow{secretRow("partner-2026", peer, true)},
			kbID:    "keeper.example",
			wantErr: "may be revoked",
		},
		{
			name: "several live keys and none named like the kb_id is an error, not a guess",
			rows: []db.ListFederationSecretsRow{
				secretRow("masha-kp-writer", peer, false),
				secretRow("masha-baserole", peer, false),
			},
			kbID:    "keeper.example",
			wantErr: "masha-kp-writer, masha-baserole",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kid, ok, err := pickOutboundKID(tt.rows, tt.kbID, peer)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantKID, kid)
		})
	}
}
