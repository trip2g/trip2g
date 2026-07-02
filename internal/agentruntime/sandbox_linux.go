//go:build linux

package agentruntime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/landlock-lsm/go-landlock/landlock"
	"golang.org/x/sys/unix"
)

// sandboxSupportedOS gates the sandbox at compile time per GOOS.
const sandboxSupportedOS = true

// sandboxChildArg is the hidden argv[1] marking a re-exec'd sandbox child.
const sandboxChildArg = "__fleet-coderun-child"

// sandboxNobodyID is the uid/gid the child drops to when the daemon is root.
const sandboxNobodyID = 65534

// MaybeRunSandboxChild must be called first in main() (and in TestMain of any
// test binary exercising the sandbox): when the process was re-exec'd as the
// confined child it applies its limits and execs the interpreter, never
// returning. In the normal parent process it is a no-op.
func MaybeRunSandboxChild() {
	if len(os.Args) != 3 || os.Args[1] != sandboxChildArg {
		return
	}
	if err := runSandboxChild(os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "coderun sandbox child:", err)
		os.Exit(125)
	}
}

// runSandboxChild applies rlimits, Landlock (BestEffort — no-op on kernels
// without it), and NO_NEW_PRIVS, then execs the interpreter. The namespaces
// and credential drop were already applied by the parent via SysProcAttr.
func runSandboxChild(encoded string) error {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode spec: %w", err)
	}
	var spec sandboxChildSpec
	if uerr := json.Unmarshal(raw, &spec); uerr != nil {
		return fmt.Errorf("parse spec: %w", uerr)
	}
	if len(spec.Argv) == 0 {
		return errors.New("empty argv")
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

	// go-landlock also sets NO_NEW_PRIVS internally, but keep it explicit so
	// the guarantee holds even when Landlock degrades to a no-op.
	if pErr := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); pErr != nil {
		return fmt.Errorf("prctl no_new_privs: %w", pErr)
	}

	var rules []landlock.Rule
	if len(spec.RODirs) > 0 {
		rules = append(rules, landlock.RODirs(spec.RODirs...).IgnoreIfMissing())
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

	bin, err := exec.LookPath(spec.Argv[0])
	if err != nil {
		return fmt.Errorf("lookup %s: %w", spec.Argv[0], err)
	}
	return unix.Exec(bin, spec.Argv, os.Environ())
}

// sandboxCommand builds the re-exec child command implementing the Tier-0
// posture: the parent contributes the namespaces + credential drop through
// SysProcAttr; the child stub (runSandboxChild) applies rlimits, Landlock and
// NO_NEW_PRIVS before exec'ing the interpreter.
func sandboxCommand(ctx context.Context, argv []string, workDir string, p SandboxPolicy) (*exec.Cmd, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("sandbox: resolve self: %w", err)
	}
	spec := sandboxChildSpec{
		Argv:          argv,
		WorkDir:       workDir,
		RODirs:        sandboxRODirs(argv[0]),
		RWDirs:        []string{workDir},
		RWFiles:       []string{"/dev/null", "/dev/zero", "/dev/urandom", "/dev/random", "/dev/full"},
		CPUSeconds:    p.CPUSeconds,
		MemoryBytes:   p.MemoryBytes,
		FileSizeBytes: p.FileSizeBytes,
		MaxProcs:      p.MaxProcs,
		MaxOpenFiles:  p.MaxOpenFiles,
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("sandbox: marshal spec: %w", err)
	}

	cmd := exec.CommandContext(ctx, self, sandboxChildArg, base64.StdEncoding.EncodeToString(raw))
	attr := &syscall.SysProcAttr{}
	if !p.Network {
		attr.Cloneflags |= syscall.CLONE_NEWNET
	}
	if os.Geteuid() == 0 {
		// Root can create the netns directly; drop the child to an
		// unprivileged uid. The workdir must follow the ownership change.
		attr.Credential = &syscall.Credential{Uid: sandboxNobodyID, Gid: sandboxNobodyID}
		if chErr := chownTree(workDir, sandboxNobodyID, sandboxNobodyID); chErr != nil {
			return nil, fmt.Errorf("sandbox: chown workdir: %w", chErr)
		}
	} else if !p.Network {
		// Unprivileged netns creation requires a user namespace
		// (the unshare --map-root-user pattern).
		attr.Cloneflags |= syscall.CLONE_NEWUSER
		attr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}}
		attr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}}
	}
	cmd.SysProcAttr = attr
	return cmd, nil
}

// sandboxRODirs returns the read-only system paths the interpreter needs
// (missing ones are ignored via IgnoreIfMissing), plus the resolved directory
// of the interpreter binary for non-standard installs.
func sandboxRODirs(program string) []string {
	dirs := []string{"/usr", "/bin", "/sbin", "/lib", "/lib32", "/lib64", "/etc", "/opt"}
	if bin, err := exec.LookPath(program); err == nil {
		if resolved, rerr := filepath.EvalSymlinks(bin); rerr == nil {
			bin = resolved
		}
		dirs = append(dirs, filepath.Dir(bin))
	}
	return dirs
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
