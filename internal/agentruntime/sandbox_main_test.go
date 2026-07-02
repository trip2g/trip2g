package agentruntime

import (
	"os"
	"testing"
)

// TestMain hooks the sandbox re-exec protocol: sandboxed RunBlock re-execs
// this test binary as the confined child, so the child branch must run before
// the test framework takes over.
func TestMain(m *testing.M) {
	MaybeRunSandboxChild()
	os.Exit(m.Run())
}

// TestSandboxDefaultIsEnforcing guards the fail-closed invariant: the zero
// policy resolves to the enforcing native mode, and only native enforces —
// besteffort (degrade-to-unsandboxed) must never be the default.
func TestSandboxDefaultIsEnforcing(t *testing.T) {
	if got := (SandboxPolicy{}).withDefaults(0).Mode; got != SandboxNative {
		t.Fatalf("default sandbox mode = %q, want %q (fail-closed default)", got, SandboxNative)
	}
	if !SandboxNative.enforcing() {
		t.Fatal("SandboxNative must be enforcing (fail-closed)")
	}
	if SandboxBestEffort.enforcing() {
		t.Fatal("SandboxBestEffort must NOT be enforcing (it degrades to unsandboxed)")
	}
	if SandboxOff.enforcing() {
		t.Fatal("SandboxOff must NOT be enforcing")
	}
}
