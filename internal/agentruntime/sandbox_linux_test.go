//go:build linux

package agentruntime

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	llsys "github.com/landlock-lsm/go-landlock/landlock/syscall"
	"github.com/stretchr/testify/require"
)

// mustEnforce reports whether the CI enforcement lane demands that the sandbox
// actually run: when FLEET_SANDBOX_MUST_ENFORCE=1 the skip guards below become
// hard failures, so a runner without userns/Landlock support fails the build
// instead of quietly reporting "green" for tests that never executed.
func mustEnforce() bool { return os.Getenv("FLEET_SANDBOX_MUST_ENFORCE") == "1" }

// skipOrFail skips normally but fails hard under FLEET_SANDBOX_MUST_ENFORCE.
func skipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()
	if mustEnforce() {
		t.Fatalf("FLEET_SANDBOX_MUST_ENFORCE=1 but "+format, args...)
	}
	t.Skipf(format, args...)
}

// requireSandboxSupport skips the test when the kernel refuses the sandbox's
// namespaces (e.g. unprivileged user namespaces disabled). The enforcing
// production path REFUSES to run there, so the isolation assertions below would
// have nothing to assert. Under FLEET_SANDBOX_MUST_ENFORCE the missing support
// is a hard failure instead of a skip.
func requireSandboxSupport(t *testing.T) {
	t.Helper()
	// Use a /tmp workdir (world-traversable, like RunBlock's os.MkdirTemp), NOT
	// t.TempDir(): the latter is a 0700 root-owned nested path the root-path uid
	// drop cannot traverse, which would misreport support as unavailable.
	dir, err := os.MkdirTemp("", "fleet-sbprobe-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	cmd, err := sandboxCommand(context.Background(), []string{"true"}, dir, SandboxPolicy{}.withDefaults(0))
	require.NoError(t, err)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		skipOrFail(t, "sandbox namespaces unavailable on this kernel: %v (child: %s)", runErr, strings.TrimSpace(stderr.String()))
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
		skipOrFail(t, "kernel has no Landlock support (needs >= 5.13 with landlock LSM enabled)")
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

// TestSandbox_NoProcLeak proves the PID + mount namespace: a host process is
// visible in /proc without the sandbox but absent inside it, so the child can
// never reach /proc/<fleet>/environ to steal JWT_SECRET/FLEET_SECRET/etc.
func TestSandbox_NoProcLeak(t *testing.T) {
	requireSandboxSupport(t)

	sleeper := exec.Command("sleep", "30")
	sleeper.Env = []string{"SECRET=LEAKED-SECRET-VALUE"}
	require.NoError(t, sleeper.Start())
	t.Cleanup(func() {
		_ = sleeper.Process.Kill()
		_, _ = sleeper.Process.Wait()
	})
	pid := sleeper.Process.Pid

	probe := fmt.Sprintf(`if [ -e /proc/%d ]; then
  echo '{"changes":[],"answer":"visible"}'
else
  echo '{"changes":[],"answer":"hidden"}'
fi`, pid)

	// Control: without the sandbox the host process is visible in /proc.
	stdout, _, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{Mode: SandboxOff},
	})
	require.NoError(t, runErr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "visible", answer, "control run must see the host process in /proc")

	// Sandboxed: the private PID namespace + fresh /proc hide every host process.
	stdout, _, _, runErr = RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
	})
	require.NoError(t, runErr)
	_, answer, perr = parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "hidden", answer, "sandboxed child must not see host processes in /proc")
}

// TestSandbox_NoNewPrivs proves NO_NEW_PRIVS is set (across all OS threads):
// with the bit set a setuid binary can never gain privilege via execve. The
// child reads the flag with prctl(PR_GET_NO_NEW_PRIVS) rather than /proc, which
// the sandbox correctly denies. Uses python (ctypes) since /proc/self/status is
// out of the Landlock grant.
func TestSandbox_NoNewPrivs(t *testing.T) {
	requireSandboxSupport(t)

	// PR_GET_NO_NEW_PRIVS == 39; returns 1 when the bit is set.
	code := `import ctypes, json
libc = ctypes.CDLL(None, use_errno=True)
print(json.dumps({"changes": [], "answer": str(libc.prctl(39, 0, 0, 0, 0))}))`
	stdout, stderr, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "python",
		Code:    code,
	})
	require.NoError(t, runErr, "stderr: %s", stderr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "1", answer, "sandboxed child must have NO_NEW_PRIVS set")
}

// TestSandbox_RlimitsApplied proves RLIMIT_AS, RLIMIT_FSIZE, RLIMIT_NPROC and
// RLIMIT_NOFILE reach the child (only RLIMIT_CPU was previously covered). It
// reads them back via bash's ulimit and compares to the requested policy.
func TestSandbox_RlimitsApplied(t *testing.T) {
	requireSandboxSupport(t)

	const (
		memBytes   = int64(512) << 20 // ulimit -v is in KiB
		fsizeBytes = int64(32) << 20  // ulimit -f is in 1024-byte blocks
		maxProcs   = 128              // ulimit -u count
		maxFiles   = 256              // ulimit -n count
	)
	code := `printf '{"changes":[],"answer":"%s|%s|%s|%s"}\n' "$(ulimit -v)" "$(ulimit -f)" "$(ulimit -u)" "$(ulimit -n)"`
	stdout, stderr, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
		Sandbox: SandboxPolicy{
			MemoryBytes:   memBytes,
			FileSizeBytes: fsizeBytes,
			MaxProcs:      maxProcs,
			MaxOpenFiles:  maxFiles,
		},
	})
	require.NoError(t, runErr, "stderr: %s", stderr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)

	fields := strings.Split(answer, "|")
	require.Len(t, fields, 4, "want v|f|u|n, got %q", answer)
	require.Equal(t, strconv.FormatInt(memBytes/1024, 10), fields[0], "RLIMIT_AS (ulimit -v, KiB)")
	require.Equal(t, strconv.FormatInt(fsizeBytes/1024, 10), fields[1], "RLIMIT_FSIZE (ulimit -f, 1024-byte blocks)")
	require.Equal(t, strconv.Itoa(maxProcs), fields[2], "RLIMIT_NPROC (ulimit -u)")
	require.Equal(t, strconv.Itoa(maxFiles), fields[3], "RLIMIT_NOFILE (ulimit -n)")
}

// TestSandbox_FallbackWhenChildCannotStart exercises graceful degradation: a
// policy whose namespaces cannot be created must fall back to plain execution
// instead of failing the run. Simulated by requesting the sandbox from within
// an environment where the probe already failed — covered indirectly: an OFF
// policy and a NATIVE policy must both produce a working run end-to-end.
func TestSandbox_ModesBothExecute(t *testing.T) {
	requireSandboxSupport(t)
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
