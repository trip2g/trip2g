package coderun

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// childWaitDelay bounds how long Wait keeps copying a child's output after the
// child is gone, when something it left behind still holds the pipes open.
const childWaitDelay = 500 * time.Millisecond

// plainCommand builds an unsandboxed child in its own process group, so a
// timeout, a cancel or the child's own exit can kill whatever it backgrounded.
// The sandboxed child needs none of this: it is PID 1 of its own PID namespace,
// and the kernel kills the namespace when it exits.
func plainCommand(ctx context.Context, argv []string) *exec.Cmd {
	// gosec G204: operator-allowlisted interpreter on the role's rendered code —
	// executing that code IS the feature, off by default.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // allowlisted interpreter
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killProcessGroup(cmd) }
	cmd.WaitDelay = childWaitDelay
	return cmd
}

// killProcessGroup kills the process group plainCommand started; a no-op for
// any other command.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil || cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		return nil
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

// finishChild runs after Wait: it kills whatever the child left running in its
// group and drops exec.ErrWaitDelay, which only means the child itself exited
// cleanly while a leftover descendant still held its output pipes.
func finishChild(cmd *exec.Cmd, waitErr error) error {
	_ = killProcessGroup(cmd)
	if errors.Is(waitErr, exec.ErrWaitDelay) {
		return nil
	}
	return waitErr
}
