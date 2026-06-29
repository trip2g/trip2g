package webhookutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/webhookutil"
)

func TestValidateConcurrencyMode(t *testing.T) {
	valid := []string{
		webhookutil.ConcurrencyAllowOverlap,
		webhookutil.ConcurrencySkip,
		webhookutil.ConcurrencyQueueOne,
	}
	for _, mode := range valid {
		require.NoError(t, webhookutil.ValidateConcurrencyMode(mode), "expected %q to be valid", mode)
	}
	// Empty string normalises to allow_overlap — also valid.
	require.NoError(t, webhookutil.ValidateConcurrencyMode(""))
}

func TestValidateConcurrencyMode_Invalid(t *testing.T) {
	cases := []string{"bogus", "SKIP", "overlap", "queue", "none"}
	for _, mode := range cases {
		require.Error(t, webhookutil.ValidateConcurrencyMode(mode), "expected %q to be rejected", mode)
	}
}

func TestNormalizeConcurrencyMode(t *testing.T) {
	require.Equal(t, webhookutil.ConcurrencyAllowOverlap, webhookutil.NormalizeConcurrencyMode(""))
	require.Equal(t, webhookutil.ConcurrencySkip, webhookutil.NormalizeConcurrencyMode(webhookutil.ConcurrencySkip))
	require.Equal(t, webhookutil.ConcurrencyQueueOne, webhookutil.NormalizeConcurrencyMode(webhookutil.ConcurrencyQueueOne))
	require.Equal(t, webhookutil.ConcurrencyAllowOverlap, webhookutil.NormalizeConcurrencyMode(webhookutil.ConcurrencyAllowOverlap))
}
