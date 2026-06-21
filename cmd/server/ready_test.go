package main

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsReady covers the readiness decision backing the /ready endpoint without
// booting the server: ready is true only when warmup finished (ready=true) and
// shutdown has not begun (stopped=false).
func TestIsReady(t *testing.T) {
	tests := []struct {
		name    string
		stopped bool
		ready   bool
		want    bool
	}{
		{name: "warming up (not ready, not stopped)", stopped: false, ready: false, want: false},
		{name: "fully ready", stopped: false, ready: true, want: true},
		{name: "shutting down after ready", stopped: true, ready: true, want: false},
		{name: "shutting down during warmup", stopped: true, ready: false, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{
				stopped: &atomic.Bool{},
				ready:   &atomic.Bool{},
			}
			a.stopped.Store(tc.stopped)
			a.ready.Store(tc.ready)

			require.Equal(t, tc.want, a.isReady())
		})
	}
}
