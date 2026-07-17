package coderun

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"
)

// SandboxSupported reports whether the native per-block sandbox can be built and
// started on this host/kernel (unprivileged user namespaces + MS_REC|MS_PRIVATE
// remount). It probes by launching a trivial confined child, so it must be
// called only AFTER MaybeRunSandboxChild in the same binary (the probe re-execs
// this binary). Non-Linux always returns false.
func SandboxSupported() bool {
	dir, err := os.MkdirTemp("", "coderun-sbprobe-*")
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

// SandboxMode selects the OS-level isolation posture for code execution.
type SandboxMode string

const (
	// SandboxOff disables OS-level isolation (pre-sandbox behavior: scrubbed
	// env + throwaway workdir + timeout only). Untrusted code runs unconfined —
	// only choose this for fully trusted code.
	SandboxOff SandboxMode = "off"
	// SandboxNative is the enforcing Tier-0 posture (see
	// docs/dev/process_isolation.md and the 2026-07-02 sandbox research):
	// empty network + PID + mount namespaces, a private /proc, Landlock
	// filesystem confinement to the run workdir, rlimits, NO_NEW_PRIVS, and a
	// drop to an unprivileged uid. Pure-Go, Linux-only. FAIL-CLOSED: if the
	// sandbox cannot be built or the child fails to start, the run is REFUSED —
	// untrusted code is never executed unconfined.
	SandboxNative SandboxMode = "native"
	// SandboxBestEffort attempts the native posture but degrades to UNSANDBOXED
	// execution (with a per-run warning) when the sandbox cannot be built or
	// started. This runs untrusted code without isolation on unsupported
	// kernels — it is NEVER the default and must be opted into explicitly.
	SandboxBestEffort SandboxMode = "besteffort"
)

// enforcing reports whether the mode refuses to run when the sandbox cannot be
// applied (fail-closed). Only SandboxNative enforces; SandboxBestEffort falls
// back and SandboxOff never sandboxes.
func (m SandboxMode) enforcing() bool { return m == SandboxNative }

// SandboxPolicy configures the sandbox applied around one code run. The zero
// value means the safe default (SandboxNative where supported, network off).
type SandboxPolicy struct {
	Mode    SandboxMode // "" → SandboxNative; anything but "off" sandboxes
	Network bool        // opt-in: keep host network access inside the sandbox

	// rlimits for the child; zero values take the defaults below.
	CPUSeconds    int   // RLIMIT_CPU; 0 → derived from the run timeout, else 300s
	MemoryBytes   int64 // RLIMIT_AS (virtual); 0 → 4 GiB (V8 reserves a lot)
	FileSizeBytes int64 // RLIMIT_FSIZE; 0 → 64 MiB
	MaxProcs      int   // RLIMIT_NPROC; 0 → 256
	MaxOpenFiles  int   // RLIMIT_NOFILE; 0 → 512
}

const (
	defaultSandboxCPUSeconds    = 300
	defaultSandboxMemoryBytes   = 4 << 30
	defaultSandboxFileSizeBytes = 64 << 20
	defaultSandboxMaxProcs      = 256
	defaultSandboxMaxOpenFiles  = 512
	sandboxCPUGraceSeconds      = 5 // headroom over the wall-clock timeout
)

// withDefaults resolves the effective policy: empty mode means the safe
// default, and zero rlimits take the package defaults. RLIMIT_CPU tracks the
// run's wall-clock timeout (plus grace) so a fork-bombed child that escapes
// the context kill still dies.
func (p SandboxPolicy) withDefaults(timeout time.Duration) SandboxPolicy {
	if p.Mode == "" {
		p.Mode = SandboxNative
	}
	if p.CPUSeconds == 0 {
		if timeout > 0 {
			p.CPUSeconds = int(timeout/time.Second) + sandboxCPUGraceSeconds
		} else {
			p.CPUSeconds = defaultSandboxCPUSeconds
		}
	}
	if p.MemoryBytes == 0 {
		p.MemoryBytes = defaultSandboxMemoryBytes
	}
	if p.FileSizeBytes == 0 {
		p.FileSizeBytes = defaultSandboxFileSizeBytes
	}
	if p.MaxProcs == 0 {
		p.MaxProcs = defaultSandboxMaxProcs
	}
	if p.MaxOpenFiles == 0 {
		p.MaxOpenFiles = defaultSandboxMaxOpenFiles
	}
	return p
}

// sandboxChildSpec is the JSON contract between RunBlock and the re-exec
// child launcher: everything the child needs to confine itself before it
// execs the interpreter.
type sandboxChildSpec struct {
	Argv          []string `json:"argv"`
	WorkDir       string   `json:"workdir"`
	RODirs        []string `json:"ro_dirs"`
	ROFiles       []string `json:"ro_files"`
	RWDirs        []string `json:"rw_dirs"`
	RWFiles       []string `json:"rw_files"`
	CPUSeconds    int      `json:"cpu_seconds"`
	MemoryBytes   int64    `json:"memory_bytes"`
	FileSizeBytes int64    `json:"file_size_bytes"`
	MaxProcs      int      `json:"max_procs"`
	MaxOpenFiles  int      `json:"max_open_files"`
	DropUID       int      `json:"drop_uid"` // >0 → setuid to this uid after mounts (root path)
	DropGID       int      `json:"drop_gid"` // >0 → setgid to this gid after mounts (root path)
}

// warnSandboxFallback logs (per run, NOT once) that a SandboxBestEffort run is
// executing WITHOUT OS-level isolation so operators see every degraded run in
// the logs. SandboxNative never reaches this path — it fails closed.
func warnSandboxFallback(reason string, err error) {
	//nolint:sloglint // pure helper with no scoped logger; operator-facing degraded-posture warning
	slog.Warn("coderun: sandbox unavailable, executing WITHOUT OS-level isolation (besteffort mode)", "reason", reason, "err", err)
}
