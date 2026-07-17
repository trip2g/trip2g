package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFleetHash_MatchesNamespacedSHA256 pins the hash derivation:
// h = hex(sha256("fleet:" + fleetID)), stable and namespaced by the "fleet:"
// prefix.
func TestFleetHash_MatchesNamespacedSHA256(t *testing.T) {
	want := func(id string) string {
		sum := sha256.Sum256([]byte("fleet:" + id))
		return hex.EncodeToString(sum[:])
	}
	cases := []string{"", "fleet1", "codellm", "prod-eu"}
	for _, id := range cases {
		require.Equal(t, want(id), fleetHash(id), "fleet_id %q", id)
	}
	// Distinct fleet_ids get distinct hashes (the partition is meaningful).
	require.NotEqual(t, fleetHash("a"), fleetHash("b"))
	// 64 hex chars (sha256).
	require.Len(t, fleetHash("x"), 64)
}

// TestFleetHash_NamespacePrefixMatters asserts the "fleet:" prefix is folded in,
// so h is not a bare sha256 of the id (namespacing against other sha256 uses).
func TestFleetHash_NamespacePrefixMatters(t *testing.T) {
	bare := sha256.Sum256([]byte("fleet1"))
	require.NotEqual(t, hex.EncodeToString(bare[:]), fleetHash("fleet1"))
}

// TestDeliveryURLs_FoldInHashSegment asserts both change and cron delivery URLs
// carry the identity segment /_fleet/<h>/webhook/ and the role's urlKey.
func TestDeliveryURLs_FoldInHashSegment(t *testing.T) {
	const base, id, path = "https://fleet.example", "fleet1", "roles/triage.md"
	h := fleetHash(id)

	change := changeDeliveryURL(base, id, path)
	require.Equal(t, base+"/_fleet/"+h+"/webhook/"+urlKey(path), change)
	require.Contains(t, change, "/"+h+"/", "change url must contain the /<h>/ segment")

	cron := cronDeliveryURL(base, id, path)
	require.Equal(t, base+"/_fleet/"+h+"/webhook/cron/"+urlKey(path), cron)
	require.Contains(t, cron, "/"+h+"/", "cron url must contain the /<h>/ segment")

	// The path prefix the receive handler strips matches what the URL builders emit.
	require.True(t, strings.HasPrefix(change, base+webhookPathPrefix(id)))
	require.True(t, strings.HasPrefix(cron, base+webhookPathPrefix(id)))
}
