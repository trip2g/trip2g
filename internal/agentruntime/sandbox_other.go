//go:build !linux

package agentruntime

import (
	"context"
	"errors"
	"os/exec"
)

// sandboxSupportedOS gates the sandbox at compile time per GOOS.
const sandboxSupportedOS = false

// MaybeRunSandboxChild is a no-op on non-Linux platforms.
func MaybeRunSandboxChild() {}

// sandboxCommand is never reached on non-Linux (sandboxSupportedOS is false);
// it exists so coderun.go compiles everywhere.
func sandboxCommand(context.Context, []string, string, SandboxPolicy) (*exec.Cmd, error) {
	return nil, errors.New("sandbox: not supported on this OS")
}
