package appconfig

import (
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

// applyStorageEnvFallback must fill the endpoint from MINIO_ENDPOINT when it was
// left at the localhost default, so a co-located box instance never silently
// starts against localhost:9000 and panics.
func TestApplyStorageEnvFallbackEndpoint(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "172.26.64.1:9000")

	c := DefaultConfig()
	c.applyStorageEnvFallback()

	if c.Storage.Endpoint != "172.26.64.1:9000" {
		t.Fatalf("endpoint fallback not applied: got %q", c.Storage.Endpoint)
	}
}

// The canonical S3 secret-key env name (MINIO_SECRET_ACCESS_KEY) is not covered
// by the flag binding; the fallback must still apply it.
func TestApplyStorageEnvFallbackCanonicalSecret(t *testing.T) {
	t.Setenv("MINIO_SECRET_ACCESS_KEY", "s3cr3t")

	c := DefaultConfig()
	c.applyStorageEnvFallback()

	if c.Storage.SecretKey != "s3cr3t" {
		t.Fatalf("canonical secret fallback not applied: got %q", c.Storage.SecretKey)
	}
}

// An explicitly configured (non-default) endpoint must win over the env fallback.
func TestApplyStorageEnvFallbackDoesNotClobberExplicit(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "172.26.64.1:9000")

	c := DefaultConfig()
	c.Storage.Endpoint = "minio.internal:9000"
	c.applyStorageEnvFallback()

	if c.Storage.Endpoint != "minio.internal:9000" {
		t.Fatalf("explicit endpoint clobbered by fallback: got %q", c.Storage.Endpoint)
	}
}
