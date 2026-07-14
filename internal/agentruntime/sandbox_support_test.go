package agentruntime

import (
	"os"
	"testing"

	"trip2g/internal/coderun"
)

// sandboxAvailable is set once in TestMain before any test runs.
var sandboxAvailable bool //nolint:gochecknoglobals // probed once before test suite; shared across all tests in the package

// TestMain hooks the sandbox re-exec protocol: sandboxed code execution (RunCode
// and the exec tool) re-execs this test binary as the confined child, so the
// child branch must run before the test framework takes over. The sandbox itself
// now lives in the coderun package.
func TestMain(m *testing.M) {
	coderun.MaybeRunSandboxChild()
	sandboxAvailable = coderun.SandboxSupported()
	os.Exit(m.Run())
}

// skipIfSandboxUnsupported skips t when the native sandbox cannot be used on
// this kernel. Tests that exercise RunCode or the exec tool with the default
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

// mustEnforce reports whether the strict sandbox-required test mode is on.
func mustEnforce() bool { return os.Getenv("FLEET_SANDBOX_MUST_ENFORCE") == "1" }
