package tgrich_test

import (
	"testing"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

func TestAccountCapability(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		isPremium   bool
		wantAllowed bool
		wantReason  string
	}{
		{
			name:        "premium mode with a premium account",
			config:      map[string]interface{}{tgrich.KeyRichMessagePosting: tgrich.PostingPremium},
			isPremium:   true,
			wantAllowed: true,
		},
		{
			name:        "premium mode without premium",
			config:      map[string]interface{}{tgrich.KeyRichMessagePosting: tgrich.PostingPremium},
			isPremium:   false,
			wantAllowed: false,
			wantReason:  tgrich.ReasonNeedsPremium,
		},
		{
			name:        "disabled refuses even a premium account",
			config:      map[string]interface{}{tgrich.KeyRichMessagePosting: tgrich.PostingDisabled},
			isPremium:   true,
			wantAllowed: false,
			wantReason:  tgrich.ReasonDisabled,
		},
		{
			name:        "enabled allows a non-premium account",
			config:      map[string]interface{}{tgrich.KeyRichMessagePosting: tgrich.PostingEnabled},
			isPremium:   false,
			wantAllowed: true,
		},
		{
			// The key predates rich messages on an account whose app_config was
			// last refreshed before July 2026. Premium is the measured gate, so
			// the fallback reproduces it rather than guessing open.
			name:        "missing key falls back to the premium gate",
			config:      map[string]interface{}{},
			isPremium:   true,
			wantAllowed: true,
		},
		{
			name:        "missing key without premium is refused",
			config:      nil,
			isPremium:   false,
			wantAllowed: false,
			wantReason:  tgrich.ReasonNeedsPremium,
		},
		{
			// An unknown spelling must not silently open the gate: the send would
			// fail server-side with RICH_MESSAGE_UNSUPPORTED and lose the reason.
			name:        "unknown mode falls back to the premium gate",
			config:      map[string]interface{}{tgrich.KeyRichMessagePosting: "something_new"},
			isPremium:   false,
			wantAllowed: false,
			wantReason:  tgrich.ReasonNeedsPremium,
		},
		{
			name:        "non-string value falls back to the premium gate",
			config:      map[string]interface{}{tgrich.KeyRichMessagePosting: 42.0},
			isPremium:   true,
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tgrich.AccountCapability(tt.config, tt.isPremium)

			require.Equal(t, tt.wantAllowed, got.Allowed)

			if tt.wantAllowed {
				require.Empty(t, got.Reason)
				return
			}

			require.Equal(t, tt.wantReason, got.Reason)
		})
	}
}

// The reason is what an admin reads in place of a generic send failure, so it
// has to name the precondition rather than the error code.
func TestAccountCapabilityReasonsAreSpecific(t *testing.T) {
	for _, reason := range []string{tgrich.ReasonNeedsPremium, tgrich.ReasonDisabled} {
		require.Contains(t, reason, "rich")
		require.NotContains(t, reason, "RICH_MESSAGE_UNSUPPORTED")
	}

	require.Contains(t, tgrich.ReasonNeedsPremium, "Premium")
}
