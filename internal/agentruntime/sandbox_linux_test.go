//go:build linux

package agentruntime

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	llsys "github.com/landlock-lsm/go-landlock/landlock/syscall"
	"github.com/stretchr/testify/require"
)

// requireSandboxSupport skips the test when the kernel refuses the sandbox's
// namespaces (e.g. unprivileged user namespaces disabled) — the production
// path falls back to unsandboxed execution there, so the isolation assertions
// below would be meaningless.
func requireSandboxSupport(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cmd, err := sandboxCommand(context.Background(), []string{"true"}, dir, SandboxPolicy{}.withDefaults(0))
	require.NoError(t, err)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	if runErr := cmd.Run(); runErr != nil {
		t.Skipf("sandbox namespaces unavailable on this kernel: %v", runErr)
	}
}

// landlockABI returns the kernel's Landlock ABI version (0 = unsupported).
func landlockABI() int {
	v, err := llsys.LandlockGetABIVersion()
	if err != nil {
		return 0
	}
	return v
}

// TestSandbox_NetworkBlocked proves the empty net namespace: a loopback TCP
// server reachable from an unsandboxed child must be unreachable from the
// sandboxed one.
func TestSandbox_NetworkBlocked(t *testing.T) {
	requireSandboxSupport(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			c, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port

	probe := fmt.Sprintf(`if (exec 3<>/dev/tcp/127.0.0.1/%d) 2>/dev/null; then
  echo '{"changes":[],"answer":"reachable"}'
else
  echo '{"changes":[],"answer":"unreachable"}'
fi`, port)

	// Control: without the sandbox the server is reachable (proves the probe).
	stdout, _, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{Mode: SandboxOff},
	})
	require.NoError(t, runErr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "reachable", answer, "control run must reach the loopback server")

	// Sandboxed (default policy): the new netns has no route to the host loopback.
	stdout, _, _, runErr = RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
	})
	require.NoError(t, runErr)
	_, answer, perr = parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "unreachable", answer, "sandboxed child must have no network access")
}

// TestSandbox_FSConfinedToWorkdir proves Landlock confinement: a file outside
// the run workdir (and outside the read-only system dirs) is readable without
// the sandbox but not inside it.
func TestSandbox_FSConfinedToWorkdir(t *testing.T) {
	requireSandboxSupport(t)
	if landlockABI() < 1 {
		t.Skip("kernel has no Landlock support (needs >= 5.13 with landlock LSM enabled)")
	}

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("OUTSIDE-SECRET"), 0o644))

	probe := fmt.Sprintf(`if cat %s >/dev/null 2>&1; then
  echo '{"changes":[],"answer":"readable"}'
else
  echo '{"changes":[],"answer":"denied"}'
fi`, secret)

	stdout, _, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{Mode: SandboxOff},
	})
	require.NoError(t, runErr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "readable", answer, "control run must read the outside file")

	stdout, _, _, runErr = RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
	})
	require.NoError(t, runErr)
	_, answer, perr = parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "denied", answer, "sandboxed child must not read outside its workdir")
}

// TestSandbox_WorkdirStaysWritable proves the flip side of confinement: the
// sandboxed child can still create and read files in its own workdir.
func TestSandbox_WorkdirStaysWritable(t *testing.T) {
	requireSandboxSupport(t)

	code := `echo data > out.txt && cat out.txt >/dev/null && echo '{"changes":[],"answer":"wrote"}'`
	stdout, stderr, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
	})
	require.NoError(t, runErr, "stderr: %s", stderr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "wrote", answer)
}

// TestSandbox_RlimitCPUTrips proves RLIMIT_CPU: a busy loop is killed by the
// kernel (SIGXCPU) well before the generous wall-clock timeout.
func TestSandbox_RlimitCPUTrips(t *testing.T) {
	requireSandboxSupport(t)

	start := time.Now()
	_, _, timedOut, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    "while :; do :; done",
		Timeout: 30 * time.Second,
		Sandbox: SandboxPolicy{CPUSeconds: 1},
	})
	require.Error(t, runErr, "busy loop must be killed by RLIMIT_CPU")
	require.False(t, timedOut, "the rlimit, not the wall-clock timeout, must trip")
	require.Less(t, time.Since(start), 15*time.Second, "SIGXCPU must fire near the 1s CPU limit")
}

// TestSandbox_FallbackWhenChildCannotStart exercises graceful degradation: a
// policy whose namespaces cannot be created must fall back to plain execution
// instead of failing the run. Simulated by requesting the sandbox from within
// an environment where the probe already failed — covered indirectly: an OFF
// policy and a NATIVE policy must both produce a working run end-to-end.
func TestSandbox_ModesBothExecute(t *testing.T) {
	for _, mode := range []SandboxMode{SandboxOff, SandboxNative} {
		stdout, stderr, _, runErr := RunBlock(context.Background(), CodeSpec{
			Program: "bash",
			Code:    `echo '{"changes":[],"answer":"ok"}'`,
			Sandbox: SandboxPolicy{Mode: mode},
		})
		require.NoError(t, runErr, "mode %s failed: %s", mode, stderr)
		_, answer, perr := parseCodeOutput(stdout)
		require.NoError(t, perr)
		require.Equal(t, "ok", answer, "mode %s", mode)
	}
}
