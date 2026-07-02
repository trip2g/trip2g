package agentruntime

import (
	"log/slog"
	"sync"
	"time"
)

// SandboxMode selects the OS-level isolation posture for code execution.
type SandboxMode string

const (
	// SandboxOff disables OS-level isolation (pre-sandbox behavior: scrubbed
	// env + throwaway workdir + timeout only).
	SandboxOff SandboxMode = "off"
	// SandboxNative is the Tier-0 posture (see docs/dev/process_isolation.md
	// and the 2026-07-02 sandbox research): empty network namespace, Landlock
	// filesystem confinement to the run workdir, rlimits, NO_NEW_PRIVS, and a
	// drop to an unprivileged uid when the daemon runs as root. Pure-Go,
	// Linux-only; unsupported platforms/kernels fall back with a warning.
	SandboxNative SandboxMode = "native"
)

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
	RWDirs        []string `json:"rw_dirs"`
	RWFiles       []string `json:"rw_files"`
	CPUSeconds    int      `json:"cpu_seconds"`
	MemoryBytes   int64    `json:"memory_bytes"`
	FileSizeBytes int64    `json:"file_size_bytes"`
	MaxProcs      int      `json:"max_procs"`
	MaxOpenFiles  int      `json:"max_open_files"`
}

//nolint:gochecknoglobals // one-shot warning gate; package-level by design
var sandboxWarnOnce sync.Once

// warnSandboxFallback logs (once per process) that code execution is running
// WITHOUT the sandbox so operators see the degraded posture in the logs.
func warnSandboxFallback(reason string, err error) {
	sandboxWarnOnce.Do(func() {
		slog.Warn("coderun: sandbox unavailable, executing without OS-level isolation", "reason", reason, "err", err)
	})
}
