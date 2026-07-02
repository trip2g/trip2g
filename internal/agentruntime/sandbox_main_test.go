package agentruntime

import (
	"context"
	"os"
	"strings"
	"testing"
)

// sandboxAvailable is set once in TestMain before any test runs.
var sandboxAvailable bool //nolint:gochecknoglobals // probed once before test suite; shared across all tests in the package

// TestMain hooks the sandbox re-exec protocol: sandboxed RunBlock re-execs
// this test binary as the confined child, so the child branch must run before
// the test framework takes over.
func TestMain(m *testing.M) {
	MaybeRunSandboxChild()
	sandboxAvailable = probeSandbox()
	os.Exit(m.Run())
}

// probeSandbox returns true when the current kernel supports the native sandbox
// (user namespaces + MS_REC|MS_PRIVATE remount). Used to gate tests that call
// RunBlock/RunCode with the default SandboxNative policy.
func probeSandbox() bool {
	dir, err := os.MkdirTemp("", "fleet-sbprobe-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	cmd, err := sandboxCommand(context.Background(), []string{"true"}, dir, SandboxPolicy{}.withDefaults(0))
	if err != nil {
		return false
	}
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	var sb strings.Builder
	cmd.Stderr = &sb
	return cmd.Run() == nil
}

// skipIfSandboxUnsupported skips t when the native sandbox cannot be used on
// this kernel. Tests that call RunBlock or RunCode with the default
// SandboxNative policy must call this at their top so the regular test job
// (no FLEET_SANDBOX_MUST_ENFORCE) gets clean skips on non-privileged runners.
func skipIfSandboxUnsupported(t *testing.T) {
	t.Helper()
	if !sandboxAvailable {
		if mustEnforce() {
			t.Fatal("FLEET_SANDBOX_MUST_ENFORCE=1 but sandbox is unavailable on this kernel")
		}
		t.Skip("sandbox unavailable on this kernel (mount namespace or MS_PRIVATE not permitted)")
	}
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
