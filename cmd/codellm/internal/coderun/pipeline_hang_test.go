package coderun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const answerJSON = `echo '{"changes":[],"answer":"x"}'`

// execWithin runs ExecCode and fails the test if it does not return within
// limit, instead of hanging the whole test binary.
func execWithin(t *testing.T, limit time.Duration, in CodeInput) (string, error) {
	t.Helper()
	type result struct {
		answer string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		_, answer, err := ExecCode(context.Background(), in)
		done <- result{answer, err}
	}()
	select {
	case r := <-done:
		return r.answer, r.err
	case <-time.After(limit):
		t.Fatalf("ExecCode did not return within %s", limit)
		return "", nil
	}
}

func pipelineModes(t *testing.T) []SandboxPolicy {
	t.Helper()
	modes := []SandboxPolicy{sandboxOff()}
	if sandboxAvailable {
		modes = append(modes, SandboxPolicy{Mode: SandboxNative})
	}
	return modes
}

func TestExecCode_PipelineDownstreamStopsReading(t *testing.T) {
	flood := "head -c 2000000 /dev/zero | tr '\\0' a"
	tests := []struct {
		name    string
		blocks  []string
		wantErr string
	}{
		{name: "downstream ignores stdin", blocks: []string{flood, answerJSON}},
		{name: "downstream reads one line", blocks: []string{"yes", "head -n 1 >/dev/null\n" + answerJSON}},
		{name: "downstream fails at once", blocks: []string{flood, "echo boom >&2\nexit 3"}, wantErr: "block 2/2: non-zero exit: boom"},
		{name: "upstream fails on its own", blocks: []string{"exit 4", answerJSON}, wantErr: "block 1/2: non-zero exit"},
		{name: "middle block ignores stdin", blocks: []string{flood, "echo hi", "cat >/dev/null\n" + answerJSON}},
	}
	for _, mode := range pipelineModes(t) {
		for _, tt := range tests {
			t.Run(string(mode.Mode)+"/"+tt.name, func(t *testing.T) {
				var body strings.Builder
				for _, b := range tt.blocks {
					body.WriteString("```bash\n" + b + "\n```\n")
				}
				answer, err := execWithin(t, 10*time.Second, CodeInput{
					Body: body.String(), AllowedPrograms: []string{"bash"},
					Timeout: 2 * time.Second, Sandbox: mode,
				})
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr)
					return
				}
				require.NoError(t, err)
				require.Equal(t, "x", answer)
			})
		}
	}
}

func TestExecCode_PipelineFailureStopsOtherBlocks(t *testing.T) {
	body := "```bash\nsleep 30\n```\n```bash\nexit 3\n```\n```bash\n" + answerJSON + "\n```\n"
	start := time.Now()
	_, err := execWithin(t, 10*time.Second, CodeInput{
		Body: body, AllowedPrograms: []string{"bash"}, Timeout: 20 * time.Second, Sandbox: sandboxOff(),
	})
	require.ErrorContains(t, err, "block 2/3: non-zero exit")
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestExecCode_BackgroundedGrandchildSandboxOff(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "orphan-alive")
	// The grandchild touches marker only if it outlives the call.
	orphan := "(sleep 2; touch " + marker + ") &\n" + answerJSON
	for _, pipeline := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "pipeline"}[pipeline], func(t *testing.T) {
			body := "```bash\n" + orphan + "\n```\n"
			if pipeline {
				body = "```bash\necho hi\n```\n```bash\ncat >/dev/null\n" + orphan + "\n```\n"
			}
			start := time.Now()
			answer, err := execWithin(t, 10*time.Second, CodeInput{
				Body: body, AllowedPrograms: []string{"bash"}, Timeout: 5 * time.Second, Sandbox: sandboxOff(),
			})
			require.NoError(t, err)
			require.Equal(t, "x", answer)
			require.Less(t, time.Since(start), 2*time.Second)

			time.Sleep(2500 * time.Millisecond)
			_, statErr := os.Stat(marker)
			require.True(t, os.IsNotExist(statErr), "backgrounded grandchild outlived the call")
		})
	}
}

func TestRunBlock_TimeoutKillsProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "orphan-alive")
	start := time.Now()
	_, _, timedOut, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash", Code: "(sleep 2; touch " + marker + ") &\nsleep 30",
		Timeout: 300 * time.Millisecond, Sandbox: sandboxOff(),
	})
	require.True(t, timedOut)
	require.Error(t, err)
	require.Less(t, time.Since(start), 3*time.Second)

	time.Sleep(2500 * time.Millisecond)
	_, statErr := os.Stat(marker)
	require.True(t, os.IsNotExist(statErr), "grandchild survived the timeout")
}
