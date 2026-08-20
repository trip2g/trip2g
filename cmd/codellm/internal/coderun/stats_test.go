package coderun

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sandboxOff runs the stats tests unsandboxed: what they assert (exit codes,
// truncation, per-block observation) is sandbox-independent, and an off policy
// keeps them running on kernels where the native sandbox is unavailable.
func sandboxOff() SandboxPolicy {
	return SandboxPolicy{Mode: SandboxOff}
}

// TestExec_ObserverReportsExitCode is the exit-code plumbing proof: a failing
// block must report its real exit status, not a collapsed zero.
func TestExec_ObserverReportsExitCode(t *testing.T) {
	var seen []BlockStats
	_, err := Exec(context.Background(), CodeInput{
		Body:            "```bash\necho boom >&2; exit 3\n```",
		AllowedPrograms: []string{"bash"},
		Sandbox:         sandboxOff(),
		Observe:         func(s BlockStats) { seen = append(seen, s) },
	}, false)

	require.Error(t, err)
	require.Equal(t, KindNonZeroExit, ErrorKind(err))
	require.Len(t, seen, 1)
	require.Equal(t, 3, seen[0].ExitCode)
	require.Equal(t, BlockNonZeroExit, seen[0].Outcome)
	require.Equal(t, "bash", seen[0].Program)
}

// TestExec_ObserverReportsSuccess asserts a clean block reports exit 0, the ok
// outcome, and its captured stdout size.
func TestExec_ObserverReportsSuccess(t *testing.T) {
	var seen []BlockStats
	_, _, err := ExecCode(context.Background(), CodeInput{
		Body:            "```bash\necho '{\"changes\":[],\"answer\":\"ok\"}'\n```",
		AllowedPrograms: []string{"bash"},
		Sandbox:         sandboxOff(),
		Observe:         func(s BlockStats) { seen = append(seen, s) },
	})

	require.NoError(t, err)
	require.Len(t, seen, 1)
	require.Equal(t, BlockOK, seen[0].Outcome)
	require.Equal(t, 0, seen[0].ExitCode)
	require.Positive(t, seen[0].StdoutBytes)
	require.False(t, seen[0].StdoutTruncated)
}

// TestExec_ObserverReportsTruncation asserts stdout hitting MaxStdoutBytes is
// reported as truncated — today the overflow is dropped silently and only
// surfaces later as a confusing parse error.
func TestExec_ObserverReportsTruncation(t *testing.T) {
	var seen []BlockStats
	_, _, err := ExecCode(context.Background(), CodeInput{
		Body:            "```bash\nhead -c 4096 /dev/zero | tr '\\0' 'x'\n```",
		AllowedPrograms: []string{"bash"},
		Sandbox:         sandboxOff(),
		MaxStdoutBytes:  64,
		Observe:         func(s BlockStats) { seen = append(seen, s) },
	})

	require.Error(t, err) // truncated stdout is no longer valid JSON
	require.Equal(t, KindParseError, ErrorKind(err))
	require.Len(t, seen, 1)
	require.True(t, seen[0].StdoutTruncated)
	require.Equal(t, 64, seen[0].StdoutBytes)
}

// TestExec_ObserverReportsEveryPipelineBlock asserts the multi-block path
// observes each block, not just the last one.
func TestExec_ObserverReportsEveryPipelineBlock(t *testing.T) {
	var seen []BlockStats
	body := "```bash\necho hi\n```\n```bash\ncat >/dev/null; echo '{\"changes\":[],\"answer\":\"ok\"}'\n```"
	_, _, err := ExecCode(context.Background(), CodeInput{
		Body:            body,
		AllowedPrograms: []string{"bash"},
		Sandbox:         sandboxOff(),
		Observe:         func(s BlockStats) { seen = append(seen, s) },
	})

	require.NoError(t, err)
	require.Len(t, seen, 2)
	require.Equal(t, 0, seen[0].Index)
	require.Equal(t, 1, seen[1].Index)
	for _, s := range seen {
		require.Equal(t, BlockOK, s.Outcome)
		require.Equal(t, 0, s.ExitCode)
	}
}

// TestExec_ObserverReportsTimeout asserts a killed block is reported as a
// timeout rather than a plain non-zero exit. It runs sandboxed on purpose: only
// the sandbox's pid namespace reaps the interpreter's own children, so an
// unsandboxed sleep would hold the captured stdout pipe open past the deadline.
func TestExec_ObserverReportsTimeout(t *testing.T) {
	skipIfSandboxUnsupported(t)
	var seen []BlockStats
	_, _, err := ExecCode(context.Background(), CodeInput{
		Body:            "```bash\nsleep 60\n```",
		AllowedPrograms: []string{"bash"},
		Timeout:         150 * time.Millisecond,
		Observe:         func(s BlockStats) { seen = append(seen, s) },
	})

	require.Error(t, err)
	require.Equal(t, KindTimeout, ErrorKind(err))
	require.Len(t, seen, 1)
	require.Equal(t, BlockTimeout, seen[0].Outcome)
}

// TestErrorKind_ClassifiesSetupFailures asserts the failure kinds a caller
// counts on are distinguishable without matching message text.
func TestErrorKind_ClassifiesSetupFailures(t *testing.T) {
	tests := []struct {
		name string
		in   CodeInput
		want string
	}{
		{
			name: "no fenced block",
			in:   CodeInput{Body: "plain prose", AllowedPrograms: []string{"bash"}},
			want: KindNoBlocks,
		},
		{
			name: "unknown fence language",
			in:   CodeInput{Body: "```brainfuck\n+++\n```", AllowedPrograms: []string{"bash"}},
			want: KindUnknownFence,
		},
		{
			name: "program not allowed",
			in:   CodeInput{Body: "```bash\necho hi\n```", AllowedPrograms: []string{"python"}},
			want: KindDisallowedProgram,
		},
		{
			name: "execution disabled",
			in:   CodeInput{Body: "```bash\necho hi\n```"},
			want: KindDisallowedProgram,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			in.Sandbox = sandboxOff()
			_, _, err := ExecCode(context.Background(), in)
			require.Error(t, err)
			require.Equal(t, tt.want, ErrorKind(err))
		})
	}
}

// TestErrorKind_Fallbacks asserts the helper is nil-safe and does not invent a
// kind for errors that carry none.
func TestErrorKind_Fallbacks(t *testing.T) {
	require.Empty(t, ErrorKind(nil))
	require.Equal(t, KindUnclassified, ErrorKind(errors.New("boom")))
}
