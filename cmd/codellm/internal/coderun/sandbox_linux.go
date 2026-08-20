//go:build linux

package coderun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/landlock-lsm/go-landlock/landlock"
	llsys "github.com/landlock-lsm/go-landlock/landlock/syscall"
	"golang.org/x/sys/unix"
)

// sandboxSupportedOS gates the sandbox at compile time per GOOS.
const sandboxSupportedOS = true

// sandboxChildArg is the hidden argv[1] marking a re-exec'd sandbox child.
const sandboxChildArg = "__fleet-coderun-child"

// sandboxSpecFD is the inherited pipe fd (via cmd.ExtraFiles) the parent uses to
// hand the child its spec. The re-exec child branch activates ONLY when this fd
// carries a valid spec, so the confined-child launcher can never be driven by
// argv alone (a stray `binary __fleet-coderun-child` with no inherited pipe
// simply errors out instead of exec'ing an attacker-chosen command).
const sandboxSpecFD = 3

// sandboxNobodyID is the uid/gid the child drops to when the daemon is root.
const sandboxNobodyID = 65534

// MaybeRunSandboxChild must be called first in main() (and in TestMain of any
// test binary exercising the sandbox): when the process was re-exec'd as the
// confined child it applies its limits and execs the interpreter, never
// returning. In the normal parent process it is a no-op.
//
// The child spec arrives on the inherited pipe (sandboxSpecFD), not argv: the
// argv sentinel alone is not enough to trigger the launcher.
func MaybeRunSandboxChild() {
	if len(os.Args) != 2 || os.Args[1] != sandboxChildArg {
		return
	}
	spec, err := readChildSpec()
	if err != nil {
		fmt.Fprintln(os.Stderr, "coderun sandbox child: read spec:", err)
		os.Exit(125)
	}
	if runErr := runSandboxChild(spec); runErr != nil {
		fmt.Fprintln(os.Stderr, "coderun sandbox child:", runErr)
		os.Exit(125)
	}
}

// readChildSpec reads and parses the spec from the inherited handshake pipe.
// A missing/closed fd (i.e. not launched by the parent) is an error, not an
// exec — this is the handshake that keeps the child off the argv-only path.
func readChildSpec() (sandboxChildSpec, error) {
	f := os.NewFile(uintptr(sandboxSpecFD), "sandbox-spec")
	if f == nil {
		return sandboxChildSpec{}, errors.New("no handshake pipe on fd 3")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 1<<20))
	if err != nil {
		return sandboxChildSpec{}, fmt.Errorf("read handshake pipe: %w", err)
	}
	if len(raw) == 0 {
		return sandboxChildSpec{}, errors.New("empty handshake pipe")
	}
	var spec sandboxChildSpec
	if uerr := json.Unmarshal(raw, &spec); uerr != nil {
		return sandboxChildSpec{}, fmt.Errorf("parse spec: %w", uerr)
	}
	return spec, nil
}

// runSandboxChild confines the re-exec'd child, then execs the interpreter.
// Order matters: private /proc (while still privileged) → drop uid → rlimits →
// NO_NEW_PRIVS (all threads) → Landlock (fail-closed) → exec. Namespaces and
// uid mappings were already applied by the parent via SysProcAttr.
func runSandboxChild(spec sandboxChildSpec) error {
	if len(spec.Argv) == 0 {
		return errors.New("empty argv")
	}

	// Fresh private /proc so the child sees ONLY its own PID namespace: no
	// /proc/<fleet>/environ, no host process list. Must run before the uid drop
	// (mounting needs CAP_SYS_ADMIN). MS_REC|MS_PRIVATE detaches propagation and
	// unlocks the mount so the fresh procfs can shadow the inherited one.
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("remount / private: %w", err)
	}
	if err := unix.Mount("proc", "/proc", "proc", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("mount private /proc: %w", err)
	}

	limits := []struct {
		name string
		res  int
		cur  uint64
	}{
		{"cpu", unix.RLIMIT_CPU, uint64(spec.CPUSeconds)},
		{"as", unix.RLIMIT_AS, uint64(spec.MemoryBytes)},
		{"fsize", unix.RLIMIT_FSIZE, uint64(spec.FileSizeBytes)},
		{"nproc", unix.RLIMIT_NPROC, uint64(spec.MaxProcs)},
		{"nofile", unix.RLIMIT_NOFILE, uint64(spec.MaxOpenFiles)},
	}
	for _, l := range limits {
		if l.cur == 0 {
			continue
		}
		if rlErr := unix.Setrlimit(l.res, &unix.Rlimit{Cur: l.cur, Max: l.cur}); rlErr != nil {
			return fmt.Errorf("setrlimit %s: %w", l.name, rlErr)
		}
	}

	// NO_NEW_PRIVS across ALL OS threads: a single-thread prctl can be lost when
	// Go execs from a different thread, so a setuid binary could still elevate.
	if pErr := llsys.AllThreadsPrctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); pErr != nil {
		return fmt.Errorf("prctl no_new_privs: %w", pErr)
	}

	// Apply Landlock while still privileged so it can open every granted path
	// (notably the RW workdir) regardless of ownership; the ruleset is inherited
	// across the uid drop and the exec below.
	if err := applyLandlock(spec); err != nil {
		return err
	}

	// Root path: drop to an unprivileged uid LAST — after the privileged mount,
	// rlimits, NO_NEW_PRIVS and Landlock are in place. (In the unprivileged
	// userns path DropUID is 0 and the child already runs as a mapped
	// non-privileged identity.) NO_NEW_PRIVS still permits dropping privilege.
	if spec.DropUID > 0 {
		if err := syscall.Setgroups([]int{}); err != nil {
			return fmt.Errorf("setgroups: %w", err)
		}
		if err := syscall.Setgid(spec.DropGID); err != nil {
			return fmt.Errorf("setgid: %w", err)
		}
		if err := syscall.Setuid(spec.DropUID); err != nil {
			return fmt.Errorf("setuid: %w", err)
		}
	}

	bin, err := exec.LookPath(spec.Argv[0])
	if err != nil {
		return fmt.Errorf("lookup %s: %w", spec.Argv[0], err)
	}
	return unix.Exec(bin, spec.Argv, os.Environ())
}

// applyLandlock enforces filesystem confinement, failing CLOSED when Landlock
// is unavailable: BestEffort silently no-ops on kernels without the Landlock
// LSM, leaving the FS unconfined while the code looks confined. We probe the
// ABI first and refuse the run when it reports 0 (unavailable). With Landlock
// present, BestEffort still enforces at the highest supported ABI. Everything
// not granted here — including /proc, /sys and /dev — is denied by default.
func applyLandlock(spec sandboxChildSpec) error {
	abi, err := llsys.LandlockGetABIVersion()
	if err != nil {
		return fmt.Errorf("landlock probe failed; refusing to run without FS confinement: %w", err)
	}
	if abi < 1 {
		return errors.New("landlock unavailable (abi 0); refusing to run without FS confinement")
	}
	var rules []landlock.Rule
	if len(spec.RODirs) > 0 {
		rules = append(rules, landlock.RODirs(spec.RODirs...).IgnoreIfMissing())
	}
	if len(spec.ROFiles) > 0 {
		rules = append(rules, landlock.ROFiles(spec.ROFiles...).IgnoreIfMissing())
	}
	if len(spec.RWDirs) > 0 {
		rules = append(rules, landlock.RWDirs(spec.RWDirs...).IgnoreIfMissing())
	}
	if len(spec.RWFiles) > 0 {
		rules = append(rules, landlock.RWFiles(spec.RWFiles...).IgnoreIfMissing())
	}
	if llErr := landlock.V5.BestEffort().RestrictPaths(rules...); llErr != nil {
		return fmt.Errorf("landlock: %w", llErr)
	}
	return nil
}

// sandboxCommand builds the re-exec child command implementing the enforcing
// Tier-0 posture. The parent contributes the namespaces + credential mapping
// through SysProcAttr and hands the spec over the inherited handshake pipe; the
// child stub (runSandboxChild) mounts a private /proc, drops privileges, and
// applies rlimits, NO_NEW_PRIVS and Landlock before exec'ing the interpreter.
func sandboxCommand(ctx context.Context, argv []string, workDir string, p SandboxPolicy) (*exec.Cmd, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("sandbox: resolve self: %w", err)
	}
	roDirs, roFiles := sandboxROPaths(argv[0], p.Network)
	spec := sandboxChildSpec{
		Argv:          argv,
		WorkDir:       workDir,
		RODirs:        roDirs,
		ROFiles:       roFiles,
		RWDirs:        []string{workDir},
		RWFiles:       []string{"/dev/null", "/dev/zero", "/dev/urandom", "/dev/random", "/dev/full"},
		CPUSeconds:    p.CPUSeconds,
		MemoryBytes:   p.MemoryBytes,
		FileSizeBytes: p.FileSizeBytes,
		MaxProcs:      p.MaxProcs,
		MaxOpenFiles:  p.MaxOpenFiles,
	}

	attr := &syscall.SysProcAttr{}
	// PID + mount namespaces isolate the process table and give us a private
	// /proc; the (optional) network namespace blocks host network access.
	attr.Cloneflags |= syscall.CLONE_NEWPID | syscall.CLONE_NEWNS
	if !p.Network {
		attr.Cloneflags |= syscall.CLONE_NEWNET
	}
	if os.Geteuid() == 0 {
		// Real root: create the namespaces directly and drop the child to a
		// distinct unprivileged uid (done in the child after the /proc mount).
		// The workdir must follow the ownership change so the dropped child can
		// still write it.
		spec.DropUID = sandboxNobodyID
		spec.DropGID = sandboxNobodyID
		if chErr := chownTree(workDir, sandboxNobodyID, sandboxNobodyID); chErr != nil {
			return nil, fmt.Errorf("sandbox: chown workdir: %w", chErr)
		}
	} else {
		// Unprivileged: a user namespace grants the caps needed to create the
		// other namespaces and mount /proc. The kernel only lets an unprivileged
		// writer map a single id whose host side is its own uid, so container 0
		// maps to the caller's uid; the PID + mount namespaces (private /proc)
		// are what actually close the /proc secret leak here.
		attr.Cloneflags |= syscall.CLONE_NEWUSER
		attr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}}
		attr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}}
	}

	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("sandbox: marshal spec: %w", err)
	}
	specR, specW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("sandbox: handshake pipe: %w", err)
	}
	// The spec is small (well under the pipe buffer); write and close the write
	// end now so the child reads to EOF. The parent's copy of the read end is
	// closed by os/exec once the child inherits it as fd 3.
	if _, werr := specW.Write(raw); werr != nil {
		_ = specW.Close()
		_ = specR.Close()
		return nil, fmt.Errorf("sandbox: write spec: %w", werr)
	}
	_ = specW.Close()

	cmd := exec.CommandContext(ctx, self, sandboxChildArg)
	cmd.ExtraFiles = []*os.File{specR} // → fd 3 in the child
	cmd.SysProcAttr = attr
	return cmd, nil
}

// sandboxROPaths returns the read-only paths the interpreter needs: the
// resolved interpreter directory, the core executable and library dirs, and a
// named handful of /etc files. Everything else (the blanket /etc, /opt and the
// rest of /usr) is denied so a compromised interpreter cannot read host config.
//
// The /etc files are individually named rather than granted as a directory:
// node refuses to start without openssl.cnf even for an offline block, and the
// resolver config is what makes DNS work — without it a block can only reach
// raw IPs. The resolver files are added only when the network is actually
// reachable, so a no-network block still sees nothing of the host's DNS setup.
// /etc/passwd, /etc/shadow and the rest of /etc stay denied in both cases.
func sandboxROPaths(program string, network bool) ([]string, []string) {
	dirs := []string{
		"/bin", "/sbin",
		"/usr/bin", "/usr/sbin",
		"/lib", "/lib32", "/lib64",
		"/usr/lib", "/usr/lib32", "/usr/lib64",
	}
	files := []string{
		"/etc/ld.so.cache",     // dynamic linker cache
		"/etc/ssl/openssl.cnf", // node aborts at startup without it
	}
	if network {
		files = append(files,
			"/etc/resolv.conf",
			"/etc/hosts",
			"/etc/nsswitch.conf",
			"/etc/services",
			"/etc/gai.conf",
		)
	}
	if bin, err := exec.LookPath(program); err == nil {
		if resolved, rerr := filepath.EvalSymlinks(bin); rerr == nil {
			bin = resolved
		}
		dirs = append(dirs, filepath.Dir(bin))
	}
	return dirs, files
}

// chownTree recursively chowns a directory tree (the per-run temp workdir).
func chownTree(root string, uid, gid int) error {
	return filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(path, uid, gid)
	})
}
