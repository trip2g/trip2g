package coderun

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunBlock_StdoutLimit(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		limit int
		size  int
	}{
		{name: "below limit", code: "printf 123456789", limit: 10, size: 9},
		{name: "exact limit", code: "printf 1234567890", limit: 10, size: 10},
		{name: "one byte over", code: "printf 12345678901", limit: 10, size: 11},
		{name: "separate writes", code: "printf 1234567890; printf 1", limit: 10, size: 11},
		{name: "default exact limit", code: "head -c 10485760 /dev/zero", size: 10 << 20},
		{name: "default overflow", code: "head -c 10485761 /dev/zero", size: (10 << 20) + 1},
		{name: "larger configured limit", code: "head -c 10485761 /dev/zero", limit: 11 << 20, size: (10 << 20) + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := RunBlockStats{}
			out, _, timedOut, err := RunBlock(context.Background(), CodeSpec{
				Program: "bash", Code: tt.code, MaxStdoutBytes: tt.limit,
				Sandbox: sandboxOff(), Timeout: 5 * time.Second, Stats: &stats,
			})
			limit := tt.limit
			if limit == 0 {
				limit = 10 << 20
			}
			require.False(t, timedOut)
			if tt.size > limit {
				require.ErrorContains(t, err, fmt.Sprintf("stdout limit exceeded (%d bytes)", limit))
				require.Equal(t, "stdout_limit_exceeded", ErrorKind(err))
				require.Equal(t, "stdout_limit_exceeded", stats.Outcome)
				require.True(t, stats.StdoutTruncated)
			} else {
				require.NoError(t, err)
				require.Equal(t, BlockOK, stats.Outcome)
				require.False(t, stats.StdoutTruncated)
			}
			require.Len(t, out, min(tt.size, limit))
		})
	}
}

func TestExecBlocksDebug_StdoutLimit(t *testing.T) {
	for _, pipeline := range []bool{false, true} {
		for _, size := range []int{9, 10, 11} {
			t.Run(fmt.Sprintf("pipeline=%t/bytes=%d", pipeline, size), func(t *testing.T) {
				body := fmt.Sprintf("```bash\nprintf %s\n```", strings.Repeat("x", size))
				if pipeline {
					body += "\n```bash\ncat\n```"
				}
				var seen []BlockStats
				out, debug, err := ExecBlocksDebug(context.Background(), CodeInput{
					Body: body, AllowedPrograms: []string{"bash"}, MaxStdoutBytes: 10,
					Sandbox: sandboxOff(), Timeout: 5 * time.Second,
					Observe: func(s BlockStats) { seen = append(seen, s) },
				}, 0)
				if size > 10 {
					require.ErrorContains(t, err, "stdout limit exceeded (10 bytes)")
					require.NotContains(t, err.Error(), "coderun: coderun:")
					require.Equal(t, "stdout_limit_exceeded", ErrorKind(err))
					require.Empty(t, out)
					require.Empty(t, debug)
					require.Equal(t, "stdout_limit_exceeded", seen[len(seen)-1].Outcome)
					if pipeline {
						require.ErrorContains(t, err, "block 2/2:")
					}
				} else {
					require.NoError(t, err)
					require.Equal(t, strings.Repeat("x", size), out)
				}
			})
		}
	}
}

func TestExec_IntermediateStdoutStreamsPastCaptureLimit(t *testing.T) {
	for _, capture := range []bool{false, true} {
		t.Run(fmt.Sprintf("capture=%t", capture), func(t *testing.T) {
			body := "```bash\nhead -c 1048576 /dev/zero\n```\n" +
				"```bash\nprintf '{\"answer\":\"%s\"}' \"$(wc -c | tr -d ' ')\"\n```"
			result, err := Exec(context.Background(), CodeInput{
				Body: body, AllowedPrograms: []string{"bash"}, MaxStdoutBytes: 64,
				Sandbox: sandboxOff(), Timeout: 5 * time.Second,
			}, capture)
			require.NoError(t, err)
			require.Equal(t, "1048576", result.Answer)
			if capture {
				require.Len(t, result.Debug, 2)
				require.Len(t, result.Debug[0].PipeBuffer, 64)
			}
		})
	}
}

func TestRunBlock_StderrLimitIsDiagnostic(t *testing.T) {
	out, stderr, _, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash", Code: "printf 12345678901 >&2; printf ok", MaxStdoutBytes: 10,
		Sandbox: sandboxOff(),
	})
	require.NoError(t, err)
	require.Equal(t, "ok", out)
	require.Equal(t, "1234567890", stderr)
}

func TestExecBlocksDebug_StdoutLimitPreservesExecutionFailure(t *testing.T) {
	for _, pipeline := range []bool{false, true} {
		t.Run(fmt.Sprintf("pipeline=%t", pipeline), func(t *testing.T) {
			body := "```bash\nprintf 12345678901; printf broken >&2; exit 3\n```"
			if pipeline {
				body = "```bash\ntrue\n```\n" + body
			}
			_, _, err := ExecBlocksDebug(context.Background(), CodeInput{
				Body: body, AllowedPrograms: []string{"bash"}, MaxStdoutBytes: 10,
				Sandbox: sandboxOff(), Timeout: 5 * time.Second,
			}, 0)
			require.Equal(t, KindNonZeroExit, ErrorKind(err))
			require.ErrorContains(t, err, "broken")
		})
	}
}
