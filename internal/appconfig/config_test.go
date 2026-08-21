package appconfig

import (
	"flag"
	"strings"
	"testing"
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

	t.Run("webhook-allow-private flag can be set", func(t *testing.T) {
		if err := flag.CommandLine.Set("webhook-allow-private", "true"); err != nil {
			t.Fatalf("failed to set webhook-allow-private flag: %v", err)
		}

		if !c.WebhookAllowPrivate {
			t.Fatal("webhook-allow-private flag did not set Config.WebhookAllowPrivate")
		}
	})
}
