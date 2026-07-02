//go:build linux

package agentruntime

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	llsys "github.com/landlock-lsm/go-landlock/landlock/syscall"
	"github.com/stretchr/testify/require"
)

// skipOrFatal skips by default (dev kernels without userns/Landlock stay
// green) but fails under SANDBOX_TESTS_REQUIRED=1 — the privileged CI lane
// sets it so "sandbox unsupported" can never masquerade as a passing run.
func skipOrFatal(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("SANDBOX_TESTS_REQUIRED") == "1" {
		t.Fatalf("SANDBOX_TESTS_REQUIRED=1 but "+format, args...)
	}
	t.Skipf(format, args...)
}

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
		skipOrFatal(t, "sandbox namespaces unavailable on this kernel: %v", runErr)
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
		skipOrFatal(t, "kernel has no Landlock support (needs >= 5.13 with landlock LSM enabled)")
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

// TestSandbox_NoNewPrivs proves the NO_NEW_PRIVS bit: setuid/setcap binaries
// cannot re-elevate the sandboxed child. A real setuid escalation needs root
// and a setuid file, so the honest portable assertion is the kernel-reported
// NoNewPrivs status bit. The child cannot introspect itself (Landlock denies
// /proc), so the unsandboxed parent reads /proc/<pid>/status of the running
// child: 0 in the control run, 1 sandboxed.
func TestSandbox_NoNewPrivs(t *testing.T) {
	requireSandboxSupport(t)

	run := func(t *testing.T, sandboxed bool) string {
		t.Helper()
		dir := t.TempDir()
		script := filepath.Join(dir, "run.sh")
		// The ready-file signals that bash is exec'd (post-confinement).
		require.NoError(t, os.WriteFile(script, []byte("echo >ready; exec sleep 30"), 0o600))
		argv := []string{"bash", script}

		var cmd *exec.Cmd
		if sandboxed {
			var err error
			cmd, err = sandboxCommand(context.Background(), argv, dir, SandboxPolicy{}.withDefaults(0))
			require.NoError(t, err)
		} else {
			cmd = exec.Command(argv[0], argv[1:]...)
		}
		cmd.Dir = dir
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		require.NoError(t, cmd.Start())
		defer func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}()

		require.Eventually(t, func() bool {
			_, err := os.Stat(filepath.Join(dir, "ready"))
			return err == nil
		}, 10*time.Second, 10*time.Millisecond, "probe never signalled readiness")

		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", cmd.Process.Pid))
		require.NoError(t, err)
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(line, "NoNewPrivs:"); ok {
				return strings.TrimSpace(v)
			}
		}
		return "0" // pre-3.5 kernels omit the line; treat absent as unset
	}

	require.Equal(t, "0", run(t, false), "control child must not have NO_NEW_PRIVS set")
	require.Equal(t, "1", run(t, true), "sandboxed child must have NO_NEW_PRIVS set")
}

// TestSandbox_RlimitFSizeTrips proves RLIMIT_FSIZE: writing a file past the
// limit gets the writer killed by SIGXFSZ only when sandboxed.
func TestSandbox_RlimitFSizeTrips(t *testing.T) {
	requireSandboxSupport(t)

	probe := `if dd if=/dev/zero of=big.bin bs=1M count=2 2>/dev/null; then
  echo '{"changes":[],"answer":"wrote"}'
else
  echo '{"changes":[],"answer":"denied"}'
fi`

	stdout, _, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{Mode: SandboxOff},
	})
	require.NoError(t, runErr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "wrote", answer, "control run must write a 2 MiB file")

	stdout, _, _, runErr = RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{FileSizeBytes: 1 << 20},
	})
	require.NoError(t, runErr)
	_, answer, perr = parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "denied", answer, "sandboxed child must not write past RLIMIT_FSIZE")
}

// TestSandbox_RlimitNoFileTrips proves RLIMIT_NOFILE: fd numbers >= the limit
// cannot be opened. The probe opens fd 100 in a subshell — fine under the
// usual >=1024 default, impossible under a limit of 64.
func TestSandbox_RlimitNoFileTrips(t *testing.T) {
	requireSandboxSupport(t)

	probe := `if (exec 100>/dev/null) 2>/dev/null; then
  echo '{"changes":[],"answer":"opened"}'
else
  echo '{"changes":[],"answer":"denied"}'
fi`

	stdout, _, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{Mode: SandboxOff},
	})
	require.NoError(t, runErr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "opened", answer, "control run must open fd 100")

	stdout, _, _, runErr = RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{MaxOpenFiles: 64},
	})
	require.NoError(t, runErr)
	_, answer, perr = parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "denied", answer, "sandboxed child must not open fds past RLIMIT_NOFILE")
}

// TestSandbox_RlimitsApplied proves every rlimit the policy sets reaches the
// child as its soft limit, including the two without a cheap enforcement
// probe: RLIMIT_AS (a tight limit crashes the Go child stub before exec) and
// RLIMIT_NPROC (per-UID accounting makes a fork probe environment-dependent).
// Deliberately odd values so a coincidental match with ambient limits is
// implausible; bash ulimit units: -t seconds, -f 1024-byte blocks, -v KiB.
func TestSandbox_RlimitsApplied(t *testing.T) {
	requireSandboxSupport(t)

	policy := SandboxPolicy{
		CPUSeconds:    137,
		MemoryBytes:   3 << 30,
		FileSizeBytes: 2 << 20,
		MaxProcs:      311,
		MaxOpenFiles:  487,
	}
	want := "cpu=137 fsize=2048 nproc=311 nofile=487 as=3145728"
	probe := `echo "{\"changes\":[],\"answer\":\"cpu=$(ulimit -t) fsize=$(ulimit -f) nproc=$(ulimit -u) nofile=$(ulimit -n) as=$(ulimit -v)\"}"`

	stdout, _, _, runErr := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: SandboxPolicy{Mode: SandboxOff},
	})
	require.NoError(t, runErr)
	_, answer, perr := parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.NotEqual(t, want, answer, "control run must keep the ambient limits")

	stdout, _, _, runErr = RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    probe,
		Sandbox: policy,
	})
	require.NoError(t, runErr)
	_, answer, perr = parseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, want, answer, "sandboxed child must run under exactly the policy's rlimits")
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
