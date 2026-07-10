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
