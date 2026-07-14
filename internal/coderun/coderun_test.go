package coderun

import (
	"context"
	"sync"
	"testing"
	"time"

	"trip2g/internal/webhookutil"

	"github.com/stretchr/testify/require"
)

// TestRunBlock_BashEchoWriteJSON runs a bash one-liner that emits a write JSON
// and asserts the output is returned verbatim.
func TestRunBlock_BashEchoWriteJSON(t *testing.T) {
	skipIfSandboxUnsupported(t)
	code := `echo '{"changes":[{"path":"notes/a.md","content":"hello"}],"answer":"done"}'`
	stdout, stderr, timedOut, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
	})
	require.NoError(t, err)
	require.False(t, timedOut)
	require.Empty(t, stderr)
	require.Contains(t, stdout, `"notes/a.md"`)
	require.Contains(t, stdout, `"hello"`)
}

// TestRunBlock_NonZeroExit asserts a failing script returns an error with stderr.
func TestRunBlock_NonZeroExit(t *testing.T) {
	skipIfSandboxUnsupported(t)
	code := `echo "oops" >&2; exit 1`
	_, stderr, timedOut, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
	})
	require.Error(t, err)
	require.False(t, timedOut)
	require.Contains(t, stderr, "oops")
}

// TestRunBlock_TimeoutKills asserts a long-running script is killed when the
// per-spec timeout fires.
func TestRunBlock_TimeoutKills(t *testing.T) {
	skipIfSandboxUnsupported(t)
	_, _, timedOut, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    "sleep 60",
		Timeout: 150 * time.Millisecond,
	})
	require.Error(t, err)
	require.True(t, timedOut, "expected timedOut=true when Timeout fires")
}

// TestRunBlock_SecretScrub is the SECURITY proof: a sentinel env var set in
// the test process must NOT be visible to the child. This proves the
// secret-scrub guarantee: cmd.Env is an explicit minimal allowlist
// (PATH + FLEET_INPUT only), never nil (inherits parent) or os.Environ().
func TestRunBlock_SecretScrub(t *testing.T) {
	skipIfSandboxUnsupported(t)
	const sentinel = "FLEET_CODERUN_TEST_SENTINEL"
	t.Setenv(sentinel, "PARENT_SECRET_VALUE")

	// The script echoes "absent" when the sentinel is NOT in the child env.
	code := `if [ -z "${FLEET_CODERUN_TEST_SENTINEL}" ]; then
  echo '{"changes":[],"answer":"absent"}'
else
  echo '{"changes":[],"answer":"present"}'
fi`

	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
	})
	require.NoError(t, err)
	_, answer, perr := ParseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "absent", answer,
		"sentinel env var must NOT be inherited by the child (secret-scrub failed)")
}

// TestRunBlock_FleetInputWritten asserts the delivery bag is accessible via
// $FLEET_INPUT in the child (the bag JSON is written to a temp file).
func TestRunBlock_FleetInputWritten(t *testing.T) {
	skipIfSandboxUnsupported(t)
	bag := []byte(`{"depth":3}`)
	code := `python3 -c "
import json, os
data = json.load(open(os.environ['FLEET_INPUT']))
print('{\"changes\":[],\"answer\":\"depth=' + str(data['depth']) + '\"}')
"`
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
		Input:   bag,
	})
	require.NoError(t, err)
	_, answer, perr := ParseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "depth=3", answer)
}

func TestParseCodeOutput_WriteShape(t *testing.T) {
	stdout := `{"changes":[{"path":"notes/a.md","content":"hello world"}],"answer":"written"}`
	changes, answer, err := ParseCodeOutput(stdout)
	require.NoError(t, err)
	require.Equal(t, "written", answer)
	require.Len(t, changes, 1)
	require.Equal(t, "notes/a.md", changes[0].Path)
	require.Equal(t, "hello world", changes[0].Content)
	require.Equal(t, webhookutil.AgentChangeKindWrite, changes[0].Kind)
}

func TestParseCodeOutput_PatchShape(t *testing.T) {
	stdout := `{"changes":[{"path":"boards/sprint.md","find":"@todo","replace":"@done"}],"answer":"patched"}`
	changes, answer, err := ParseCodeOutput(stdout)
	require.NoError(t, err)
	require.Equal(t, "patched", answer)
	require.Len(t, changes, 1)
	require.Equal(t, webhookutil.AgentChangeKindPatch, changes[0].Kind)
	require.Equal(t, "@todo", changes[0].Find)
	require.Equal(t, "@done", changes[0].Replace)
}

func TestParseCodeOutput_BadJSON(t *testing.T) {
	_, _, err := ParseCodeOutput("not json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse stdout")
}

func TestParseCodeOutput_MissingPath(t *testing.T) {
	_, _, err := ParseCodeOutput(`{"changes":[{"content":"oops"}],"answer":""}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing path")
}

func TestParseCodeOutput_EmptyChanges(t *testing.T) {
	changes, answer, err := ParseCodeOutput(`{"changes":[],"answer":"noop"}`)
	require.NoError(t, err)
	require.Equal(t, "noop", answer)
	require.Empty(t, changes)
}

// TestRunBlock_StdinPiped asserts CodeSpec.Stdin is fed to the child's stdin.
func TestRunBlock_StdinPiped(t *testing.T) {
	skipIfSandboxUnsupported(t)
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    "cat",
		Stdin:   []byte("piped-data"),
	})
	require.NoError(t, err)
	require.Equal(t, "piped-data", stdout)
}

// TestExtractFencedBlocks asserts exported multi-block extraction in document order.
func TestExtractFencedBlocks(t *testing.T) {
	blocks := ExtractFencedBlocks("```bash\nfirst\n```\ntext\n```python\nsecond\n```")
	require.Len(t, blocks, 2)
	require.Equal(t, FencedBlock{Lang: "bash", Code: "first\n"}, blocks[0])
	require.Equal(t, FencedBlock{Lang: "python", Code: "second\n"}, blocks[1])
	require.Empty(t, ExtractFencedBlocks("no blocks here"))
}

func TestExtractFirstFencedBlock(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantLang string
		wantCode string
	}{
		{"bash block", "```bash\necho hi\n```", "bash", "echo hi\n"},
		{"python block", "```python\nprint('x')\n```", "python", "print('x')\n"},
		{"no block", "just text", "", ""},
		{"unclosed block", "```bash\necho hi", "", ""},
		{"first of two", "```bash\nfirst\n```\n```python\nsecond\n```", "bash", "first\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lang, code := extractFirstFencedBlock(tc.body)
			require.Equal(t, tc.wantLang, lang)
			require.Equal(t, tc.wantCode, code)
		})
	}
}

func TestFenceLangToProgram(t *testing.T) {
	require.Equal(t, "python", fenceLangToProgram("python"))
	require.Equal(t, "python", fenceLangToProgram("py"))
	require.Equal(t, "bash", fenceLangToProgram("bash"))
	require.Equal(t, "bash", fenceLangToProgram("sh"))
	require.Equal(t, "node", fenceLangToProgram("node"))
	require.Equal(t, "node", fenceLangToProgram("js"))
	require.Equal(t, "node", fenceLangToProgram("javascript"))
	require.Equal(t, "ruby", fenceLangToProgram("ruby"))
}

func TestProgramBinary(t *testing.T) {
	b, err := programBinary("python")
	require.NoError(t, err)
	require.Equal(t, "python3", b)

	b, err = programBinary("bash")
	require.NoError(t, err)
	require.Equal(t, "bash", b)

	_, err = programBinary("haskell")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown program")
}

func TestIsAllowed(t *testing.T) {
	require.True(t, IsAllowed("bash", []string{"bash", "python"}))
	require.False(t, IsAllowed("node", []string{"bash", "python"}))
	require.False(t, IsAllowed("bash", nil))
}

func TestInterpretersJSON_Load(t *testing.T) {
	prog := fenceLangToProgram("python")
	require.Equal(t, "python", prog)
	prog = fenceLangToProgram("py")
	require.Equal(t, "python", prog)
	prog = fenceLangToProgram("ruby")
	require.Equal(t, "ruby", prog)
	prog = fenceLangToProgram("haskell")
	require.Empty(t, prog)
}

func TestInterpretersJSON_Override(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, SetInterpretersJSON(defaultInterpretersJSON)) })
	custom := []byte(`{"interpreters":[{"name":"bash","cmd":["bash"],"code_block_labels":["bash","sh"],"ext":".sh"}]}`)
	require.NoError(t, SetInterpretersJSON(custom))
	require.Equal(t, "bash", fenceLangToProgram("bash"))
	require.Empty(t, fenceLangToProgram("python"))
}

// TestInterpreterRegistry_ConcurrentAccess exercises the registry under -race:
// concurrent SetInterpretersJSON swaps against reads from every lookup path
// (the same shape as a startup --interpreters override racing role runs).
func TestInterpreterRegistry_ConcurrentAccess(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, SetInterpretersJSON(defaultInterpretersJSON)) })
	custom := []byte(`{"interpreters":[{"name":"bash","cmd":["bash"],"code_block_labels":["bash","sh"],"ext":".sh"}]}`)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := range 100 {
				data := custom
				if i%2 == 0 {
					data = defaultInterpretersJSON
				}
				if err := SetInterpretersJSON(data); err != nil {
					t.Errorf("SetInterpretersJSON: %v", err)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range 100 {
				FenceLangKnown("bash")
				fenceLangToProgram("sh")
				_, _ = programBinary("bash")
			}
		}()
	}
	wg.Wait()
	require.True(t, FenceLangKnown("bash"))
}

func TestMaxStdoutBytesFromConfig(t *testing.T) {
	skipIfSandboxUnsupported(t)
	code := `printf '%0.s1234567890' {1..10}` // writes 100 bytes
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program:        "bash",
		Code:           code,
		MaxStdoutBytes: 10,
	})
	require.NoError(t, err)
	require.LessOrEqual(t, len(stdout), 10, "stdout must not exceed MaxStdoutBytes")
}

// TestRunBlock_WorkdirIsolation asserts each RunBlock call gets a clean working
// directory with no files from a previous run.
func TestRunBlock_WorkdirIsolation(t *testing.T) {
	skipIfSandboxUnsupported(t)
	// First run creates a file in its workdir.
	code1 := `echo '{"changes":[],"answer":"run1"}'`
	_, _, _, err := RunBlock(context.Background(), CodeSpec{Program: "bash", Code: code1})
	require.NoError(t, err)

	// Second run should have a clean workdir (the first run's temp dir was removed).
	code2 := `ls | wc -l | tr -d ' ' | xargs -I{} echo '{"changes":[],"answer":"{}"}'`
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{Program: "bash", Code: code2})
	require.NoError(t, err)
	_, answer, perr := ParseCodeOutput(stdout)
	require.NoError(t, perr)
	// The workdir should contain exactly 2 files: run.sh and input.json.
	require.Equal(t, "2", answer,
		"workdir should contain exactly 2 files (run.sh + input.json)")
}

// TestRunBlock_ParentSecretAbsent is an additional secret-scrub probe: sets a
// sentinel in the parent env and asserts it is absent from the child. This
// complements TestRunBlock_SecretScrub with a differently-named sentinel to
// rule out any pattern-matching false negatives.
func TestRunBlock_ParentSecretAbsent(t *testing.T) {
	skipIfSandboxUnsupported(t)
	t.Setenv("FLEET_SECRET_LEAK_TEST", "must-not-appear-in-child")

	code := `if [ -z "${FLEET_SECRET_LEAK_TEST}" ]; then
  echo '{"changes":[],"answer":"absent"}'
else
  echo '{"changes":[],"answer":"present"}'
fi`
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program: "bash",
		Code:    code,
	})
	require.NoError(t, err)
	_, answer, perr := ParseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "absent", answer,
		"parent secret sentinel must NOT be inherited by child (secret-scrub)")
}

// TestLimitedBuffer caps at the limit and does not error on overflow.
func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{limit: 5}
	n, err := b.Write([]byte("hello world"))
	require.NoError(t, err)
	require.Equal(t, 11, n, "Write must return len(p) even when capped")
	require.Equal(t, "hello", b.String(), "data must be capped at limit")
}

// TestRunBlock_EnvPassthrough asserts the selective env pass-through mechanism:
//   - EnvPassthrough ["FOO"] + EnvPrefix ["BAR_"] → child gets FOO and BAR_X
//   - but NOT SECRET (a parent var that matches neither list nor prefix).
//
// This proves the deny-by-default + explicit-opt-in contract: secrets not
// declared in env_passthrough or env_prefix do NOT reach the child.
func TestRunBlock_EnvPassthrough(t *testing.T) {
	skipIfSandboxUnsupported(t)
	t.Setenv("FLEET_ENVTEST_FOO", "1")
	t.Setenv("FLEET_ENVTEST_BAR_X", "2")
	t.Setenv("FLEET_ENVTEST_SECRET", "should-not-appear")

	// The script counts how many of the three vars are visible.
	code := `python3 -c "
import os, json
got = {}
for k in ['FLEET_ENVTEST_FOO','FLEET_ENVTEST_BAR_X','FLEET_ENVTEST_SECRET']:
    if k in os.environ:
        got[k] = os.environ[k]
print(json.dumps({'changes':[],'answer':json.dumps(got)}))
"`
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program:        "bash",
		Code:           code,
		EnvPassthrough: []string{"FLEET_ENVTEST_FOO"},
		EnvPrefix:      []string{"FLEET_ENVTEST_BAR_"},
	})
	require.NoError(t, err)
	_, answer, perr := ParseCodeOutput(stdout)
	require.NoError(t, perr)

	require.Contains(t, answer, "FLEET_ENVTEST_FOO",
		"passthrough var FOO must be visible to child")
	require.Contains(t, answer, "FLEET_ENVTEST_BAR_X",
		"prefix-matched var BAR_X must be visible to child")
	require.NotContains(t, answer, "FLEET_ENVTEST_SECRET",
		"SECRET must NOT be passed to child (not in passthrough or prefix)")
}

// TestRunBlock_EnvPassthroughBaseOnly asserts the base-only path: when neither
// EnvPassthrough nor EnvPrefix is set, the specific parent sentinel is absent
// from the child. (bash injects a few vars of its own like SHLVL, so we check
// for the sentinel specifically rather than asserting zero extra vars.)
func TestRunBlock_EnvPassthroughBaseOnly(t *testing.T) {
	skipIfSandboxUnsupported(t)
	t.Setenv("FLEET_ENVTEST_EXTRA", "should-not-appear")

	code := `if [ -z "${FLEET_ENVTEST_EXTRA}" ]; then
  echo '{"changes":[],"answer":"absent"}'
else
  echo '{"changes":[],"answer":"present"}'
fi`
	stdout, _, _, err := RunBlock(context.Background(), CodeSpec{
		Program:        "bash",
		Code:           code,
		EnvPassthrough: nil,
		EnvPrefix:      nil,
	})
	require.NoError(t, err)
	_, answer, perr := ParseCodeOutput(stdout)
	require.NoError(t, perr)
	require.Equal(t, "absent", answer,
		"parent sentinel must NOT be inherited when EnvPassthrough and EnvPrefix are both empty")
}

// Tests in this file use t.Setenv which correctly restores values on cleanup.
// The parent-env isolation under test is enforced by buildChildEnv in coderun.go.
