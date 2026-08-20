package coderun

import (
	"errors"
	"fmt"
)

// Block outcomes reported through CodeInput.Observe.
const (
	BlockOK          = "ok"
	BlockNonZeroExit = "nonzero_exit"
	BlockTimeout     = "timeout"
	BlockStartFailed = "start_failed"
)

// Failure kinds carried by ExecError. They classify WHY a run failed without
// the caller matching on message text.
const (
	KindNoBlocks          = "no_blocks"
	KindUnknownFence      = "unknown_fence"
	KindUnknownProgram    = "unknown_program"
	KindDisallowedProgram = "disallowed_program"
	KindSandboxRefused    = "sandbox_refused"
	KindSetupFailed       = "setup_failed"
	KindStartFailed       = "start_failed"
	KindTimeout           = "timeout"
	KindNonZeroExit       = "nonzero_exit"
	KindParseError        = "parse_error"
	KindInternal          = "internal"
	// KindUnclassified is what ErrorKind reports for an error that carries no
	// kind of its own — never returned by coderun itself.
	KindUnclassified = "unclassified"
)

// BlockStats is one executed block's measured outcome. coderun records no
// metrics itself: it reports what it measured and the caller (codellm) decides
// what to count, which keeps this package free of a metrics dependency.
type BlockStats struct {
	Index    int
	Program  string
	Outcome  string // BlockOK | BlockNonZeroExit | BlockTimeout | BlockStartFailed
	ExitCode int    // -1 when the child never produced an exit status

	DurationMs  int64
	MaxRSSBytes int64

	// StdoutBytes / StdoutTruncated describe the stdout this block's caller
	// captured. Only the block whose stdout is buffered reports them: in a
	// pipeline the intermediate blocks stream straight into the next block's
	// stdin, so their sizes are unknown (0/false) unless debug capture is on.
	StdoutBytes     int
	StdoutTruncated bool

	// SandboxFallback is non-empty when a besteffort policy degraded this block
	// to unsandboxed execution, carrying the reason. Enforcing mode refuses the
	// run instead and never reaches a BlockStats.
	SandboxFallback string
}

// Sentinel errors for the two failures that carry no per-run detail.
var (
	errNoFencedBlock     = errors.New("coderun: no fenced code block found in rendered body")
	errExecutionDisabled = errors.New("coderun: code execution disabled (set --allowed-programs)")
	errLastStdoutMissing = errors.New("coderun: internal error: last block stdout not captured")
)

// ExecError classifies an execution failure. The message is unchanged from the
// wrapped error, so callers that match on text keep working; Kind is the stable
// label to count on.
type ExecError struct {
	Kind string
	Err  error
}

func (e *ExecError) Error() string { return e.Err.Error() }

func (e *ExecError) Unwrap() error { return e.Err }

// ErrorKind returns the classification of err: "" for nil, the ExecError kind
// when one is present, KindUnclassified otherwise.
func ErrorKind(err error) string {
	if err == nil {
		return ""
	}
	var ee *ExecError
	if errors.As(err, &ee) {
		return ee.Kind
	}
	return KindUnclassified
}

// execErrf builds a classified error with a formatted message.
func execErrf(kind, format string, a ...any) error {
	return &ExecError{Kind: kind, Err: fmt.Errorf(format, a...)}
}

// observeSingleBlock reports the single-block path's measured stats. An empty
// Outcome means RunBlock failed before the child ran (sandbox refused, workdir
// or file setup) — there is no block to report, only an exec error.
func observeSingleBlock(observe func(BlockStats), program string, stats RunBlockStats) {
	if observe == nil || stats.Outcome == "" {
		return
	}
	observe(BlockStats{
		Index:           0,
		Program:         program,
		Outcome:         stats.Outcome,
		ExitCode:        stats.ExitCode,
		DurationMs:      stats.DurationMs,
		MaxRSSBytes:     stats.MaxRSSBytes,
		StdoutBytes:     stats.StdoutBytes,
		StdoutTruncated: stats.StdoutTruncated,
		SandboxFallback: stats.SandboxFallback,
	})
}
