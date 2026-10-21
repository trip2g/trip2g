package appconfig

import (
	"flag"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// The internal listener serves unauthenticated /metrics, pprof and debug
// endpoints. Its default must bind loopback only, never all interfaces.
func TestDefaultInternalListenAddrIsLoopback(t *testing.T) {
	t.Parallel()

	addr := DefaultConfig().InternalListenAddr
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("internal listen default must be loopback (127.0.0.1:port), got %q", addr)
	}
}

// Webhook deliveries go through the SSRF-safe dialer by default; only an
// operator setting -webhook-allow-private opts a deployment out of it.
func TestDefaultConfigWebhookAllowPrivateIsFalse(t *testing.T) {
	t.Parallel()

	if DefaultConfig().WebhookAllowPrivate {
		t.Fatal("webhook-allow-private must default to false")
	}
}

// applyStorageEnvFallback must accept the canonical S3 secret-key env name
// (MINIO_SECRET_ACCESS_KEY) when -minio-secret-key was never explicitly set, and
// must never clobber an explicitly set -minio-secret-key — even one whose value
// happens to equal the built-in default — so precedence stays flag/env > default.
// Subtests share one Config/defineFlags() call: flag.StringVar binds to
// flag.CommandLine (a process global), so defining the same flag twice panics.
func TestApplyStorageEnvFallbackSecretKey(t *testing.T) {
	c := DefaultConfig()
	c.defineFlags()

	t.Run("applies canonical env when flag was never set", func(t *testing.T) {
		t.Setenv("MINIO_SECRET_ACCESS_KEY", "s3cr3t")

		c.applyStorageEnvFallback()

		if c.Storage.SecretKey != "s3cr3t" {
			t.Fatalf("canonical secret fallback not applied: got %q", c.Storage.SecretKey)
		}
	})

	t.Run("does not clobber an explicitly set flag equal to the default", func(t *testing.T) {
		if err := flag.CommandLine.Set("minio-secret-key", DefaultMinIOSecretKey); err != nil {
			t.Fatalf("failed to set minio-secret-key flag: %v", err)
		}
		t.Setenv("MINIO_SECRET_ACCESS_KEY", "should-not-apply")

		c.applyStorageEnvFallback()

		if c.Storage.SecretKey != DefaultMinIOSecretKey {
			t.Fatalf("explicitly set flag was clobbered by fallback: got %q", c.Storage.SecretKey)
		}
	})

	t.Run("trusted-proxies parses a list and refuses a bad entry", func(t *testing.T) {
		require.NoError(t, flag.CommandLine.Set("trusted-proxies", "127.0.0.1,172.16.0.0/12"))
		require.Equal(t, "127.0.0.1/32,172.16.0.0/12", c.TrustedProxies.String())

		require.Error(t, flag.CommandLine.Set("trusted-proxies", "172.16.0.0/40"))
	})

	t.Run("webhook-allow-private flag can be set", func(t *testing.T) {
		if err := flag.CommandLine.Set("webhook-allow-private", "true"); err != nil {
			t.Fatalf("failed to set webhook-allow-private flag: %v", err)
		}

		if !c.WebhookAllowPrivate {
			t.Fatal("webhook-allow-private flag did not set Config.WebhookAllowPrivate")
		}
	})
}

// OWNER_PASSWORD_HASH takes a bcrypt hash only: a plaintext password set there
// by mistake must stop boot instead of becoming a password nobody can match.
func TestValidateOwnerPasswordHash(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.MinCost)
	require.NoError(t, err)

	tests := []struct {
		name    string
		hash    string
		email   string
		wantErr string
	}{
		{name: "unset", hash: "", email: ""},
		{name: "bcrypt hash with owner email", hash: string(hash), email: "owner@example.com"},
		{name: "plaintext password", hash: "correct horse", email: "owner@example.com", wantErr: "OwnerPasswordHash: must be a bcrypt hash"},
		{name: "hash without owner email", hash: string(hash), email: "", wantErr: "OwnerEmail: is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := DefaultConfig()
			c.DevMode = true
			c.OwnerEmail = tt.email
			c.OwnerPasswordHash = tt.hash

			validateErr := c.validate()
			if tt.wantErr == "" {
				require.NoError(t, validateErr)
				return
			}
			require.ErrorContains(t, validateErr, tt.wantErr)
		})
	}
}
