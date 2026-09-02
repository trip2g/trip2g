package agentruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// execStub is a fake exec endpoint (codellm) recording the request it received
// and returning a canned tool_calls completion or an error.
type execStub struct {
	res ChatResult
	err error

	calls    int
	model    string
	messages []Message
}

func (e *execStub) Chat(_ context.Context, model string, messages []Message, _ []ToolDef) (ChatResult, error) {
	e.calls++
	e.model = model
	e.messages = messages
	return e.res, e.err
}

// execInput builds a Run input whose scripted main LLM first calls
// exec(program, code) and then finishes, against the given exec endpoint.
func execInput(kb KB, writePatterns []string, program, code string, exec LLM) (Input, *stubLLM) {
	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolExec, map[string]any{
					"program": program,
					"code":    code,
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}
	return Input{
		Instruction:   "run some code",
		ReadPatterns:  []string{"**"},
		WritePatterns: writePatterns,
		Model:         "m",
		MaxTokens:     10000,
		MaxSteps:      5,
		ExecLLM:       exec,
		LLM:           llm,
		KB:            kb,
	}, llm
}

// runWithExec drives execInput through Run and requires it to succeed.
func runWithExec(t *testing.T, kb *memKB, writePatterns []string, program, code string, exec LLM) (*Result, *stubLLM) {
	t.Helper()
	in, llm := execInput(kb, writePatterns, program, code, exec)
	res, err := Run(context.Background(), in)
	require.NoError(t, err)
	return res, llm
}

// execResult returns the tool result the main model received for its exec call.
func execResult(llm *stubLLM) string {
	var out string
	for _, m := range llm.seen {
		if m.Role == RoleTool && m.Name == toolExec {
			out = m.Content
		}
	}
	return out
}

// failingWriteKB is a memKB whose Write of failPath fails after validation —
// the remote KB refusing a write mid-batch.
type failingWriteKB struct {
	*memKB
	failPath string
}

func (k *failingWriteKB) Write(ctx context.Context, path, content string) error {
	if path == k.failPath {
		return errors.New("boom")
	}
	return k.memKB.Write(ctx, path, content)
}

// TestRun_ExecToolGatedByExecLLM asserts:
//  1. exec is NOT in the offered tool set when ExecLLM is nil.
//  2. exec IS in the offered set when ExecLLM is non-nil.
func TestRun_ExecToolGatedByExecLLM(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "content"})
	finishOnly := func() *stubLLM {
		return &stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
					PromptTokens: 5, CompletionTokens: 5},
			},
		}
	}

	t.Run("absent when ExecLLM nil", func(t *testing.T) {
		llm := finishOnly()
		_, err := Run(context.Background(), Input{
			Instruction:   "x",
			ReadPatterns:  []string{"**"},
			WritePatterns: []string{"**"},
			Model:         "m",
			MaxTokens:     10000,
			MaxSteps:      5,
			ExecLLM:       nil, // nil → exec disabled
			LLM:           llm,
			KB:            kb,
		})
		require.NoError(t, err)
		for _, td := range llm.seenTools {
			require.NotEqual(t, toolExec, td.Name,
				"exec must NOT be offered when ExecLLM is nil")
		}
	})

	t.Run("present when ExecLLM non-nil", func(t *testing.T) {
		llm := finishOnly()
		_, err := Run(context.Background(), Input{
			Instruction:   "x",
			ReadPatterns:  []string{"**"},
			WritePatterns: []string{"**"},
			Model:         "m",
			MaxTokens:     10000,
			MaxSteps:      5,
			ExecLLM:       &execStub{},
			LLM:           llm,
			KB:            kb,
		})
		require.NoError(t, err)
		found := false
		for _, td := range llm.seenTools {
			if td.Name == toolExec {
				found = true
			}
		}
		require.True(t, found, "exec must be offered when ExecLLM is non-nil")
	})
}

// TestRun_ExecToolRoutesThroughExecLLM asserts the exec tool sends the code as
// a one-fenced-block chat completion (fence label = program name) to ExecLLM
// and applies the returned write_note tool call through the scoped KB.
func TestRun_ExecToolRoutesThroughExecLLM(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "notes/exec.md", "content": "from exec"}),
			toolCall("c1", toolFinish, map[string]any{"answer": "wrote"}),
		}},
	}

	res, llm := runWithExec(t, kb, []string{"notes/**"}, "bash", "echo hi", exec)

	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, "from exec", kb.docs["notes/exec.md"], "exec tool write must land in KB")
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/exec.md", res.Changes[0].Path)

	// Wire shape: exactly one call, one user message, code fenced under the
	// program-name label.
	require.Equal(t, 1, exec.calls)
	require.Len(t, exec.messages, 1)
	require.Equal(t, RoleUser, exec.messages[0].Role)
	require.Equal(t, "```bash\necho hi\n```", exec.messages[0].Content)

	// The model gets the same summary shape as the old in-process exec.
	var summary string
	for _, m := range llm.seen {
		if m.Role == RoleTool && m.Name == toolExec {
			summary = m.Content
		}
	}
	require.Equal(t, "ok: ran bash, 1 write(s); wrote", summary)
}

// TestRun_ExecToolAppliesPatch asserts a patch_note tool call from the exec
// endpoint is applied as a patch through the scoped KB.
func TestRun_ExecToolAppliesPatch(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/p.md": "status: todo"})
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolPatchNote, map[string]any{"path": "notes/p.md", "find": "todo", "replace": "done"}),
			toolCall("c1", toolFinish, map[string]any{"answer": ""}),
		}},
	}

	res, _ := runWithExec(t, kb, []string{"notes/**"}, "python", "patch()", exec)

	require.Equal(t, "status: done", kb.docs["notes/p.md"])
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/p.md", res.Changes[0].Path)
}

// TestRun_ExecToolDeniedOutOfScope asserts a write returned by the exec
// endpoint targeting a path outside write_patterns is denied (recorded in
// res.Denials) and never reaches the KB.
func TestRun_ExecToolDeniedOutOfScope(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "secret/x.md", "content": "leak"}),
			toolCall("c1", toolFinish, map[string]any{"answer": "tried"}),
		}},
	}

	res, _ := runWithExec(t, kb, []string{"notes/**"}, "bash", "echo leak", exec)

	require.Empty(t, res.Changes, "out-of-scope exec write must not appear in Changes")
	require.Len(t, res.Denials, 1)
	require.Contains(t, res.Denials[0], "secret/x.md")
	_, leaked := kb.docs["secret/x.md"]
	require.False(t, leaked, "out-of-scope path must not appear in KB")
}

// TestRun_ExecToolDenialReachesModelAndMetrics asserts an out-of-scope write in
// an exec batch is reported the way write_note reports its own denial: named in
// the tool result the model receives, recorded on res.Denials, and counted as
// a write denial — while the in-scope write in the same batch still lands.
func TestRun_ExecToolDenialReachesModelAndMetrics(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "notes/ok.md", "content": "fine"}),
			toolCall("c1", toolWriteNote, map[string]any{"path": "secret/x.md", "content": "leak"}),
			toolCall("c2", toolFinish, map[string]any{"answer": "wrote"}),
		}},
	}
	m := newFakeMetrics()
	in, llm := execInput(kb, []string{"notes/**"}, "bash", "echo hi", exec)
	in.Metrics = m

	res, err := Run(context.Background(), in)
	require.NoError(t, err)

	require.Equal(t, "ok: ran bash, 1 write(s), 1 denied (secret/x.md: write denied: path outside write scope); wrote", execResult(llm))
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/ok.md", res.Changes[0].Path)
	require.Equal(t, "fine", kb.docs["notes/ok.md"])
	require.Equal(t, []string{"exec write secret/x.md"}, res.Denials)
	require.Equal(t, []string{denialWrite}, m.denials)
	require.Equal(t, outcomeDenied, m.tools[toolExec])
}

// TestRun_ExecToolApplyFailureHardFailsRun asserts a patch the exec endpoint
// returns that cannot be applied is an apply failure like patch_note's: counted
// as one, and fatal to the run under HardFailApply.
func TestRun_ExecToolApplyFailureHardFailsRun(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/p.md": "todo todo"})
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolPatchNote, map[string]any{"path": "notes/p.md", "find": "todo", "replace": "done"}),
			toolCall("c1", toolFinish, map[string]any{"answer": ""}),
		}},
	}
	m := newFakeMetrics()
	in, _ := execInput(kb, []string{"notes/**"}, "python", "patch()", exec)
	in.Metrics = m
	in.HardFailApply = true

	res, err := Run(context.Background(), in)
	require.Error(t, err)
	require.Nil(t, res)
	require.Contains(t, err.Error(), "apply exec")
	require.Contains(t, err.Error(), "notes/p.md")

	require.Equal(t, "todo todo", kb.docs["notes/p.md"], "a refused patch must not touch the note")
	require.Equal(t, []string{toolExec}, m.applies)
	require.Equal(t, outcomeApplyFailed, m.tools[toolExec])
	require.Len(t, m.runs, 1)
	require.Equal(t, statusErrorForRun, m.runs[0].status)
}

// TestRun_ExecToolValidatesBatchBeforeApplying pins the mixed batch: three
// writes, the second out of scope, the third a patch whose find is not unique.
// Every change is validated before any is applied, so the bad patch refuses the
// whole batch — nothing lands, the denial is still recorded, and the model sees
// the apply error instead of a success line over a half-written vault.
func TestRun_ExecToolValidatesBatchBeforeApplying(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/c.md": "x x"})
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "notes/a.md", "content": "one"}),
			toolCall("c1", toolWriteNote, map[string]any{"path": "secret/b.md", "content": "two"}),
			toolCall("c2", toolPatchNote, map[string]any{"path": "notes/c.md", "find": "x", "replace": "y"}),
			toolCall("c3", toolFinish, map[string]any{"answer": "three"}),
		}},
	}
	m := newFakeMetrics()
	in, llm := execInput(kb, []string{"notes/**"}, "bash", "run", exec)
	in.Metrics = m

	res, err := Run(context.Background(), in)
	require.NoError(t, err, "without HardFailApply the apply failure stays a soft tool result")

	require.Empty(t, res.Changes, "nothing may be applied when the batch fails validation")
	_, wroteA := kb.docs["notes/a.md"]
	require.False(t, wroteA, "the valid first write must not be committed ahead of the bad patch")
	require.Equal(t, "x x", kb.docs["notes/c.md"])
	require.Equal(t, []string{"exec write secret/b.md"}, res.Denials)
	require.Equal(t, []string{denialWrite}, m.denials)
	require.Equal(t, []string{toolExec}, m.applies)
	require.Equal(t, outcomeApplyFailed, m.tools[toolExec])
	got := execResult(llm)
	require.True(t, strings.HasPrefix(got, "error: apply notes/c.md:"), "model must see the apply error, got %q", got)
}

// TestRun_ExecToolMidBatchWriteFailureIsApplyFailure asserts a KB write that
// fails after validation (the remote KB refusing mid-batch) is still an apply
// failure, and that only the changes that actually landed are reported.
func TestRun_ExecToolMidBatchWriteFailureIsApplyFailure(t *testing.T) {
	kb := &failingWriteKB{memKB: newMemKB(nil), failPath: "notes/b.md"}
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "notes/a.md", "content": "one"}),
			toolCall("c1", toolWriteNote, map[string]any{"path": "notes/b.md", "content": "two"}),
			toolCall("c2", toolFinish, map[string]any{"answer": ""}),
		}},
	}
	m := newFakeMetrics()
	in, llm := execInput(kb, []string{"notes/**"}, "bash", "run", exec)
	in.Metrics = m

	res, err := Run(context.Background(), in)
	require.NoError(t, err)

	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/a.md", res.Changes[0].Path)
	require.Equal(t, "one", kb.docs["notes/a.md"])
	require.Equal(t, []string{toolExec}, m.applies)
	require.Equal(t, outcomeApplyFailed, m.tools[toolExec])
	require.Equal(t, "error: apply notes/b.md: boom", execResult(llm))
}

// TestRun_ExecToolBatchSeesItsOwnEarlierChanges asserts a later change in an
// exec batch is validated against what the earlier ones leave behind, not the
// note as it was before the batch: the second patch's find only exists once the
// first has run, exactly as applying them one at a time would have found.
func TestRun_ExecToolBatchSeesItsOwnEarlierChanges(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/n.md": "a a"})
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolPatchNote, map[string]any{"path": "notes/n.md", "find": "a a", "replace": "b c"}),
			toolCall("c1", toolPatchNote, map[string]any{"path": "notes/n.md", "find": "c", "replace": "d"}),
			toolCall("c2", toolFinish, map[string]any{"answer": ""}),
		}},
	}

	res, llm := runWithExec(t, kb, []string{"notes/**"}, "bash", "run", exec)

	require.Equal(t, "ok: ran bash, 2 write(s)", execResult(llm))
	require.Len(t, res.Changes, 2)
	require.Equal(t, "b d", kb.docs["notes/n.md"])
}

// TestRun_ExecToolBatchCannotMintRoleNoteAcrossPatches asserts the role guard
// judges each patch against the note as the batch has already changed it. Two
// patches that are each harmless on the original note — one fixing the fence,
// one fixing the key — would together turn it into a role note; the second
// must be denied.
func TestRun_ExecToolBatchCannotMintRoleNoteAcrossPatches(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/r.md": "-x-\nfleet_yd: c\n---\nbody"})
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolPatchNote, map[string]any{"path": "notes/r.md", "find": "-x-", "replace": "---"}),
			toolCall("c1", toolPatchNote, map[string]any{"path": "notes/r.md", "find": "fleet_yd", "replace": "fleet_id"}),
			toolCall("c2", toolFinish, map[string]any{"answer": ""}),
		}},
	}

	res, _ := runWithExec(t, kb, []string{"notes/**"}, "bash", "run", exec)

	require.Len(t, res.Changes, 1)
	require.Equal(t, []string{"exec write notes/r.md: role note (fleet_id) — agents may not author roles"}, res.Denials)
	require.Equal(t, "---\nfleet_yd: c\n---\nbody", kb.docs["notes/r.md"])
	require.False(t, declaresRole(kb.docs["notes/r.md"]), "the batch must not leave a role note behind")
}

// TestRun_ExecToolBatchPinsPatchToContentItLeaves asserts a patch following a
// write of the same note in one batch is conditioned on the bytes that write
// leaves — what trip2g will hold when the patch runs — not on a pre-batch read
// that would fail for a note the batch itself creates.
func TestRun_ExecToolBatchPinsPatchToContentItLeaves(t *testing.T) {
	kb := &conditionalKB{KB: newMemKB(nil)}
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "notes/a.md", "content": "v1\n"}),
			toolCall("c1", toolPatchNote, map[string]any{"path": "notes/a.md", "find": "v1", "replace": "v2"}),
			toolCall("c2", toolFinish, map[string]any{"answer": ""}),
		}},
	}
	in, llm := execInput(kb, []string{"notes/**"}, "bash", "run", exec)

	res, err := Run(context.Background(), in)
	require.NoError(t, err)

	require.Equal(t, "ok: ran bash, 2 write(s)", execResult(llm))
	require.Len(t, res.Changes, 2)
	require.True(t, kb.gotCalled, "a hash-capable KB must get the conditional patch")
	require.Equal(t, contentHash("v1\n"), kb.gotHash)
}

// TestRun_ExecToolEndpointError asserts an error from the exec endpoint (e.g.
// codellm's deterministic 422 for a disallowed program or a failing block) is
// surfaced to the model as a soft tool error, producing no changes.
func TestRun_ExecToolEndpointError(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{err: errors.New(`program "node" not in --allowed-programs [bash]`)}

	res, llm := runWithExec(t, kb, []string{"**"}, "node", "console.log(1)", exec)

	require.Empty(t, res.Changes, "failed exec must not produce changes")
	var gotError bool
	for _, msg := range llm.seen {
		if strings.Contains(msg.Content, "not in --allowed-programs") {
			gotError = true
		}
	}
	require.True(t, gotError, "model must receive the exec endpoint's error")
}

// TestRun_ExecToolRejectsFenceInCode asserts code containing a ``` fence
// marker is rejected up front (it cannot ride the one-block wire protocol)
// without calling the exec endpoint.
func TestRun_ExecToolRejectsFenceInCode(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{}

	res, llm := runWithExec(t, kb, []string{"**"}, "bash", "cat <<EOF\n```md\nEOF\n", exec)

	require.Empty(t, res.Changes)
	require.Equal(t, 0, exec.calls, "exec endpoint must not be called")
	var gotError bool
	for _, msg := range llm.seen {
		if msg.Role == RoleTool && strings.HasPrefix(msg.Content, "error:") {
			gotError = true
		}
	}
	require.True(t, gotError, "model must receive a soft error")
}

// TestRun_ExecToolRejectsBadProgram asserts an empty or non-token program name
// is rejected before any endpoint call.
func TestRun_ExecToolRejectsBadProgram(t *testing.T) {
	for _, program := range []string{"", "ba sh", "bash\nrm"} {
		kb := newMemKB(nil)
		exec := &execStub{}

		res, _ := runWithExec(t, kb, []string{"**"}, program, "echo hi", exec)

		require.Empty(t, res.Changes, "program %q must not produce changes", program)
		require.Equal(t, 0, exec.calls, "program %q must not reach the endpoint", program)
	}
}

// TestRun_ExecToolUnexpectedToolCall asserts a tool call other than
// write_note/patch_note/finish coming back from the exec endpoint is a soft
// error, not a silent skip.
func TestRun_ExecToolUnexpectedToolCall(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", "read_note", map[string]any{"path": "notes/a.md"}),
		}},
	}

	res, llm := runWithExec(t, kb, []string{"**"}, "bash", "echo hi", exec)

	require.Empty(t, res.Changes)
	var gotError bool
	for _, msg := range llm.seen {
		if strings.Contains(msg.Content, "unexpected tool call") {
			gotError = true
		}
	}
	require.True(t, gotError)
}

// TestRun_DefaultToolCountUnchangedWithoutExecLLM asserts that with ExecLLM
// nil, the default tool set is still exactly 5 (backward-compat).
func TestRun_DefaultToolCountUnchangedWithoutExecLLM(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "x"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}
	_, err := Run(context.Background(), Input{
		Instruction:  "x",
		ReadPatterns: []string{"**"},
		Model:        "m",
		MaxTokens:    10000,
		MaxSteps:     5,
		ExecLLM:      nil,
		LLM:          llm,
		KB:           kb,
	})
	require.NoError(t, err)
	require.Len(t, llm.seenTools, 5,
		"ExecLLM=nil must not add exec to the offered set")
}
