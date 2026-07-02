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
