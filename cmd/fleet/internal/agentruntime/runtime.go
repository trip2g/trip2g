package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"trip2g/internal/webhookutil"
)

// Run status values.
const (
	StatusCompleted = "completed" // model finished (finish tool or no tool calls).
	StatusCapped    = "capped"    // token hard-cap stopped the loop.
	StatusMaxSteps  = "max_steps" // step guard stopped the loop.
)

// Tool names exposed to the model.
const (
	toolSearch    = "search"
	toolReadNote  = "read_note"
	toolWriteNote = "write_note"
	toolPatchNote = "patch_note"
	toolFinish    = "finish"
	toolExec      = "exec"
)

// fleetInputMessageName is the reserved message name that carries the delivery
// bag (Input.InputBag) as a system message. codellm reads it as $FLEET_INPUT and
// does NOT scan it for fenced code; it must match codellm's fleetInputMessageName.
const fleetInputMessageName = "fleet_input"

// toolInvoker is the extension-point function type for the tool registry seam.
// Built-in tools (search, read_note, write_note, patch_note, finish) are
// handled by execTool's switch. Extension tools (exec, future MCP plug-ins)
// register an invoker here via buildInvokers so the loop never needs a new
// switch case. future: populate from MCP server descriptors, config, etc.
type toolInvoker func(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) string

// Tool-call outcomes reported to Metrics. Only outcomeApplyFailed is a genuine
// write failure; the rest stay soft and self-correctable.
const (
	outcomeOK           = "ok"
	outcomeDenied       = "denied"
	outcomeInvalidArgs  = "invalid_args"
	outcomeError        = "error"
	outcomeApplyFailed  = "apply_failed"
	outcomeNotPermitted = "not_permitted"
)

// Denial kinds reported to Metrics.RecordDenial.
const (
	denialRead          = "read"
	denialWrite         = "write"
	denialNotPermitted  = "not_permitted"
	statusErrorForRun   = "error"
	tokenKindPrompt     = "prompt"
	tokenKindCompletion = "completion"
)

// Metrics is the optional run-metrics sink. It is declared here, minimally, so
// agentruntime never imports the fleet's metrics package: the fleet passes its
// own implementation, and --once passes none. Implementations must be nil-safe.
type Metrics interface {
	RecordRun(role, status string, steps int, seconds float64)
	RecordTokens(model, role, kind string, n int)
	RecordToolCall(tool, outcome string)
	RecordDenial(kind string)
	RecordApplyFailure(role, tool string)
}

// Input is one executor run, derived from a webhook delivery
// (instruction + read_patterns + write_patterns + model) plus the safety
// hard-cap. LLM and KB are injected so the loop is testable offline.
type Input struct {
	Instruction   string
	ReadPatterns  []string
	WritePatterns []string
	Model         string

	// Tools is the role-declared allowlist of tool names the model may use.
	// finish is always included regardless of this list.
	// An empty (nil) Tools means the full default offered set (backward-compat).
	Tools []string

	// ExecLLM, when non-nil, enables the exec(program, code) tool: the code is
	// sent as a one-fenced-block chat completion to this OpenAI-compatible
	// endpoint (codellm), which executes it and returns the writes as
	// write_note/patch_note tool calls plus finish(answer). nil disables exec
	// entirely. Program allowlisting and sandboxing are codellm's concern —
	// fleet executes no code in-process.
	ExecLLM LLM

	// MaxTokens is the NON-overridable per-run token hard-cap (safety floor).
	// The model has no tool to change it; the loop enforces it. Must be > 0.
	MaxTokens int
	// MaxSteps bounds tool-loop iterations as a secondary guard. Must be > 0.
	MaxSteps int

	// InputBag, when non-empty, is delivered as a system message named
	// "fleet_input" carrying the JSON delivery bag. codellm treats that message
	// as $FLEET_INPUT (it does not scan it for fenced code); a real LLM sees a
	// harmless labeled JSON context block. Used by the code→codellm path.
	InputBag []byte

	// HardFailApply makes a genuine write/patch APPLY failure (a bad or non-unique
	// patch find, a KB write error — NOT an out-of-scope denial) fail the whole
	// run instead of feeding the error back to the model as a soft, self-
	// correctable tool result. Set for executor:code/codellm runs to preserve
	// the code path's all-or-nothing semantics; left false for real-LLM roles
	// so they keep their self-correction loop.
	HardFailApply bool

	// Role labels this run's metrics (the role note path). Empty for --once and
	// tests, which report under the empty label.
	Role string

	// Metrics receives this run's outcome, spend and tool activity. nil records
	// nothing; the call sites stay unconditional either way.
	Metrics Metrics

	LLM LLM
	KB  KB
}

// Result is the outcome of a run. Changes carries the writes in the same shape
// the existing webhook change-apply path consumes (webhookutil.AgentChange).
type Result struct {
	Answer     string                    `json:"answer"`
	Changes    []webhookutil.AgentChange `json:"changes"`
	Status     string                    `json:"status"`
	TokensUsed int                       `json:"tokens_used"`
	Steps      int                       `json:"steps"`
	Denials    []string                  `json:"denials,omitempty"`
}

const systemPromptTemplate = `You are a scoped trip2g micro-agent. Follow the instruction below.

Instruction:
%s

Access scope (enforced by the runtime, not negotiable):
- You may read paths matching: %s
- You may write paths matching: %s

Tools:
- search(query): find documents within your read scope.
- read_note(path): read a document's content (read scope only).
- write_note(path, content): create or replace a document (write scope only).
- patch_note(path, find, replace): surgically edit a document (write scope only). Fails if find is absent or occurs more than once; include enough surrounding context to make the match unique.
- finish(answer): end the run with your final answer/summary.

Rules:
- Reads or writes outside scope are rejected by the runtime; do not retry them.
- When done, call finish with a concise answer. Do not invent paths you have not seen.`

// Run executes the instruction through the LLM tool-call loop, enforcing scope
// and the token hard-cap, and returns the answer plus any in-scope changes. It
// wraps run so every exit path — including the error ones — is measured once.
func Run(ctx context.Context, in Input) (*Result, error) {
	started := time.Now()
	res, err := run(ctx, in)
	recordRun(in, res, err, time.Since(started))
	return res, err
}

// recordRun reports the run's terminal status, steps and token spend. A run
// that failed outright has no Result, so it is counted with zero steps under
// the error status.
func recordRun(in Input, res *Result, err error, elapsed time.Duration) {
	if in.Metrics == nil {
		return
	}
	if res == nil {
		in.Metrics.RecordRun(in.Role, statusErrorForRun, 0, elapsed.Seconds())
		return
	}
	status := res.Status
	if err != nil {
		status = statusErrorForRun
	}
	in.Metrics.RecordRun(in.Role, status, res.Steps, elapsed.Seconds())
}

func run(ctx context.Context, in Input) (*Result, error) {
	if in.LLM == nil {
		return nil, errors.New("agentruntime: LLM is required")
	}
	if in.KB == nil {
		return nil, errors.New("agentruntime: KB is required")
	}
	if in.MaxTokens <= 0 {
		return nil, errors.New("agentruntime: MaxTokens must be > 0")
	}
	if in.MaxSteps <= 0 {
		return nil, errors.New("agentruntime: MaxSteps must be > 0")
	}

	scoped := NewScopedKB(in.KB, in.ReadPatterns, in.WritePatterns)
	tools := allowedToolDefs(in.Tools, in.ExecLLM != nil)
	// permitted is the execution-time enforcement set: same as the advertised set.
	// finish is always present (already guaranteed by allowedToolDefs).
	permitted := make(map[string]bool, len(tools))
	for _, td := range tools {
		permitted[td.Name] = true
	}

	// Tool registry: extension invokers (exec + future MCP plug-ins) are called
	// before the built-in switch in execTool. Built-in tools leave this nil-safe.
	invokers := buildInvokers(in.ExecLLM)

	messages := []Message{
		{
			Role: RoleSystem,
			Content: fmt.Sprintf(
				systemPromptTemplate,
				in.Instruction,
				formatPatterns(in.ReadPatterns),
				formatPatterns(in.WritePatterns),
			),
		},
	}
	// Delivery bag rides as a labeled system message (fleet_input). codellm
	// consumes it as $FLEET_INPUT; a real LLM sees a harmless JSON context block.
	if len(in.InputBag) > 0 {
		messages = append(messages, Message{
			Role:    RoleSystem,
			Name:    fleetInputMessageName,
			Content: string(in.InputBag),
		})
	}
	messages = append(messages, Message{Role: RoleUser, Content: "Begin."})

	res := &Result{Status: StatusMaxSteps}

	for step := range in.MaxSteps {
		// Hard-cap check happens BEFORE each model call: once spent, stop.
		if res.TokensUsed >= in.MaxTokens {
			res.Status = StatusCapped
			return res, nil
		}

		chat, err := chatWithBudget(ctx, in.LLM, in.Model, messages, tools, in.MaxTokens-res.TokensUsed)
		if err != nil {
			return nil, fmt.Errorf("agentruntime: chat step %d: %w", step, err)
		}
		res.Steps++
		res.TokensUsed += chat.PromptTokens + chat.CompletionTokens
		if in.Metrics != nil {
			in.Metrics.RecordTokens(in.Model, in.Role, tokenKindPrompt, chat.PromptTokens)
			in.Metrics.RecordTokens(in.Model, in.Role, tokenKindCompletion, chat.CompletionTokens)
		}

		// No tool calls means the model returned a final answer.
		if len(chat.ToolCalls) == 0 {
			res.Answer = chat.Content
			res.Status = StatusCompleted
			return res, nil
		}

		messages = append(messages, Message{
			Role:      RoleAssistant,
			Content:   chat.Content,
			ToolCalls: chat.ToolCalls,
		})

		done, err := runToolCalls(ctx, in, toolLoop{
			scoped:    scoped,
			res:       res,
			permitted: permitted,
			invokers:  invokers,
			messages:  &messages,
		}, chat.ToolCalls)
		if err != nil {
			return nil, err
		}
		if done {
			return res, nil
		}
	}

	return res, nil
}

// toolLoop bundles the per-run state one assistant turn's tool calls act on.
type toolLoop struct {
	scoped    *ScopedKB
	res       *Result
	permitted map[string]bool
	invokers  map[string]toolInvoker
	messages  *[]Message
}

// runToolCalls executes one assistant turn's tool calls in order, appending
// each result as a tool message. It returns done=true when the model called
// finish (the run's terminal state is already set on res), and an error only
// for a HardFailApply apply failure.
func runToolCalls(ctx context.Context, in Input, tl toolLoop, calls []ToolCall) (bool, error) {
	for _, call := range calls {
		if call.Name == toolFinish {
			tl.res.Answer = finishAnswer(call.Arguments)
			tl.res.Status = StatusCompleted
			return true, nil
		}
		// Execution-time allowlist enforcement: reject any tool not in the
		// permitted set, even if the model hallucinated or was injected.
		if !tl.permitted[call.Name] {
			denial := "tool not permitted: " + call.Name
			tl.res.Denials = append(tl.res.Denials, denial)
			if in.Metrics != nil {
				in.Metrics.RecordToolCall(call.Name, outcomeNotPermitted)
				in.Metrics.RecordDenial(denialNotPermitted)
			}
			*tl.messages = append(*tl.messages, Message{
				Role:       RoleTool,
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    "error: " + denial,
			})
			continue
		}
		output, outcome := execTool(ctx, tl.scoped, tl.res, call, tl.invokers)
		if in.Metrics != nil {
			in.Metrics.RecordToolCall(call.Name, outcome)
			if outcome == outcomeDenied {
				in.Metrics.RecordDenial(denialForTool(call.Name))
			}
		}
		// Apply-error hard-fail (executor:code/codellm path): a genuine write/
		// patch apply failure (bad/non-unique find, KB write error) fails the
		// whole run, preserving the code path's all-or-nothing semantics.
		// Real-LLM roles (HardFailApply=false) keep the soft, self-correctable
		// tool result.
		if outcome == outcomeApplyFailed {
			if in.Metrics != nil {
				in.Metrics.RecordApplyFailure(in.Role, call.Name)
			}
			if in.HardFailApply {
				return false, fmt.Errorf("agentruntime: apply %s: %s", call.Name, output)
			}
		}
		*tl.messages = append(*tl.messages, Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    output,
		})
	}
	return false, nil
}

// chatWithBudget passes the run's remaining token budget to LLMs that support
// per-call completion caps (BudgetedLLM); others get the plain Chat call.
func chatWithBudget(ctx context.Context, llm LLM, model string, messages []Message, tools []ToolDef, remaining int) (ChatResult, error) {
	if b, ok := llm.(BudgetedLLM); ok {
		return b.ChatWithBudget(ctx, model, messages, tools, remaining)
	}
	return llm.Chat(ctx, model, messages, tools)
}

// execTool runs one tool call against the scoped KB and returns the textual
// result to feed back to the model, plus the outcome for metrics. Only
// outcomeApplyFailed marks a write/patch that could not be APPLIED (bad or
// non-unique find, KB write error) — that is what the HardFailApply path
// (executor:code/codellm) fails on. Scope denials, invalid arguments and
// read/search errors get their own outcomes and stay soft, so they remain
// self-correctable in every path. Scope denials are recorded and surfaced to
// the model (so it learns), never silently swallowed.
// Extension tools registered in invokers are dispatched before the built-in
// switch, forming the tool registry seam for exec and future MCP plug-ins.
func execTool(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall, invokers map[string]toolInvoker) (string, string) {
	if fn, ok := invokers[call.Name]; ok {
		out := fn(ctx, scoped, res, call)
		return out, outcomeFor(out)
	}
	switch call.Name {
	case toolSearch:
		return execSearch(ctx, scoped, call)
	case toolReadNote:
		return execReadNote(ctx, scoped, res, call)
	case toolWriteNote:
		return execWriteNote(ctx, scoped, res, call)
	case toolPatchNote:
		return execPatchNote(ctx, scoped, res, call)
	default:
		return "error: unknown tool " + call.Name, outcomeError
	}
}

// outcomeFor classifies an extension tool's textual result: the registry seam
// returns a string, and "error: ..." is its failure convention.
func outcomeFor(output string) string {
	if strings.HasPrefix(output, "error:") {
		return outcomeError
	}
	return outcomeOK
}

// denialForTool maps a denied tool call to the denial kind it represents.
func denialForTool(tool string) string {
	if tool == toolReadNote {
		return denialRead
	}
	return denialWrite
}

func execSearch(ctx context.Context, scoped *ScopedKB, call ToolCall) (string, string) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "error: invalid arguments: " + err.Error(), outcomeInvalidArgs
	}
	docs, err := scoped.Search(ctx, args.Query)
	if err != nil {
		return "error: " + err.Error(), outcomeError
	}
	if len(docs) == 0 {
		return "no in-scope documents matched.", outcomeOK
	}
	var b strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&b, "- %s\n", d.Path)
	}
	return b.String(), outcomeOK
}

func execReadNote(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) (string, string) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "error: invalid arguments: " + err.Error(), outcomeInvalidArgs
	}
	content, err := scoped.Read(ctx, args.Path)
	if errors.Is(err, ErrReadDenied) {
		res.Denials = append(res.Denials, "read "+args.Path)
		return "error: " + ErrReadDenied.Error(), outcomeDenied
	}
	if err != nil {
		return "error: " + err.Error(), outcomeError
	}
	return content, outcomeOK
}

func execWriteNote(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) (string, string) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "error: invalid arguments: " + err.Error(), outcomeInvalidArgs
	}
	err := scoped.Write(ctx, args.Path, args.Content)
	if errors.Is(err, ErrWriteDenied) {
		res.Denials = append(res.Denials, "write "+args.Path)
		return "error: " + ErrWriteDenied.Error(), outcomeDenied
	}
	if err != nil {
		return "error: " + err.Error(), outcomeApplyFailed
	}
	res.Changes = append(res.Changes, webhookutil.AgentChange{
		Path:    args.Path,
		Content: args.Content,
		Kind:    webhookutil.AgentChangeKindWrite,
	})
	return "ok: wrote " + args.Path, outcomeOK
}

func execPatchNote(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) (string, string) {
	var args struct {
		Path    string `json:"path"`
		Find    string `json:"find"`
		Replace string `json:"replace"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "error: invalid arguments: " + err.Error(), outcomeInvalidArgs
	}
	err := scoped.Patch(ctx, args.Path, args.Find, args.Replace)
	if errors.Is(err, ErrWriteDenied) {
		res.Denials = append(res.Denials, "patch "+args.Path)
		return "error: " + ErrWriteDenied.Error(), outcomeDenied
	}
	if err != nil {
		return "error: " + err.Error(), outcomeApplyFailed
	}
	res.Changes = append(res.Changes, webhookutil.AgentChange{
		Path:    args.Path,
		Find:    args.Find,
		Replace: args.Replace,
		Kind:    webhookutil.AgentChangeKindPatch,
	})
	return "ok: patched " + args.Path, outcomeOK
}

func finishAnswer(arguments string) string {
	var args struct {
		Answer string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return ""
	}
	return args.Answer
}

func formatPatterns(patterns []string) string {
	if len(patterns) == 0 {
		return "(none)"
	}
	return strings.Join(patterns, ", ")
}

// allowedToolDefs returns the ToolDef slice the model will see. When allowlist
// is non-empty, only tools named in it are included; finish is always injected
// regardless. An empty allowlist returns the full default offered set.
// execEnabled gates whether exec is included in the base set.
func allowedToolDefs(allowlist []string, execEnabled bool) []ToolDef {
	all := toolDefs(execEnabled)
	if len(allowlist) == 0 {
		return all
	}
	permitted := make(map[string]bool, len(allowlist))
	for _, name := range allowlist {
		permitted[name] = true
	}
	// finish is non-negotiable.
	permitted[toolFinish] = true

	var out []ToolDef
	for _, td := range all {
		if permitted[td.Name] {
			out = append(out, td)
		}
	}
	return out
}

// execModel is the model id sent on exec-tool chat calls. codellm echoes it;
// the exec endpoint does not route by model.
const execModel = "codellm"

// buildInvokers constructs the extension tool registry. When execLLM is
// non-nil, exec is registered.
// Future MCP tools: add invokers here, gated by their own enablement knobs.
// Never add tools unconditionally.
func buildInvokers(execLLM LLM) map[string]toolInvoker {
	if execLLM == nil {
		return nil
	}
	return map[string]toolInvoker{
		toolExec: makeExecInvoker(execLLM),
	}
}

// makeExecInvoker returns an invoker for the exec(program, code) tool. It wraps
// the code in a single fenced block labeled with the program name and sends it
// as a one-shot chat completion to execLLM (codellm), which executes the block
// and returns the writes as write_note/patch_note tool calls plus finish(answer).
// Those changes are applied via the scoped KB — same write_patterns enforcement
// as write_note. Out-of-scope writes are denied and recorded, not silently
// dropped. Program allowlisting and sandboxing are codellm-authoritative: a
// disallowed program or failing block comes back as a deterministic error (422),
// surfaced to the model as a soft tool error.
func makeExecInvoker(execLLM LLM) toolInvoker {
	return func(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) string {
		var args struct {
			Program string `json:"program"`
			Code    string `json:"code"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "error: invalid arguments: " + err.Error()
		}
		body, err := fenceCodeBlock(args.Program, args.Code)
		if err != nil {
			return "error: " + err.Error()
		}
		// No fleet_input message: exec runs inline; the model already has context.
		// No explicit timeout: the call is bounded by the parent run context.
		chat, err := execLLM.Chat(ctx, execModel, []Message{{Role: RoleUser, Content: body}}, nil)
		if err != nil {
			return "error: " + err.Error()
		}
		changes, answer, perr := changesFromToolCalls(chat.ToolCalls)
		if perr != nil {
			return "error: " + perr.Error()
		}
		nWritten := 0
		for _, ch := range changes {
			var applyErr error
			switch ch.Kind {
			case webhookutil.AgentChangeKindPatch:
				applyErr = scoped.Patch(ctx, ch.Path, ch.Find, ch.Replace)
			default: // AgentChangeKindWrite or empty
				applyErr = scoped.Write(ctx, ch.Path, ch.Content)
			}
			if errors.Is(applyErr, ErrWriteDenied) {
				res.Denials = append(res.Denials, "exec write "+ch.Path)
				continue
			}
			if applyErr != nil {
				return "error: apply " + ch.Path + ": " + applyErr.Error()
			}
			res.Changes = append(res.Changes, ch)
			nWritten++
		}
		summary := fmt.Sprintf("ok: ran %s, %d write(s)", args.Program, nWritten)
		if answer != "" {
			summary += "; " + answer
		}
		return summary
	}
}

// fenceCodeBlock wraps code in a ```program fenced block for the exec wire
// protocol. The program name doubles as the fence label — valid because every
// interpreter name is also a registered fence label (pinned by
// TestInterpreterNamesAreFenceLabels in the coderun package). Code that itself
// contains a ``` marker cannot ride the one-block protocol and is rejected up
// front (a deterministic error beats silent block corruption).
func fenceCodeBlock(program, code string) (string, error) {
	if program == "" || strings.ContainsAny(program, " \t\n`") {
		return "", fmt.Errorf("exec: invalid program name %q", program)
	}
	if strings.Contains(code, "```") {
		return "", errors.New("exec: code containing ``` cannot be sent as a fenced block")
	}
	if !strings.HasSuffix(code, "\n") {
		code += "\n"
	}
	return "```" + program + "\n" + code + "```", nil
}

// changesFromToolCalls maps the exec endpoint's write_note/patch_note/finish
// tool calls back to AgentChanges plus the finish answer (the inverse of
// codellm's {changes}→tool_calls mapping). Any other tool name is an error —
// the exec endpoint's contract is exactly these three.
func changesFromToolCalls(calls []ToolCall) ([]webhookutil.AgentChange, string, error) {
	var changes []webhookutil.AgentChange
	var answer string
	for _, tc := range calls {
		switch tc.Name {
		case toolWriteNote:
			var a struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(tc.Arguments), &a); err != nil {
				return nil, "", fmt.Errorf("exec: %s arguments: %w", tc.Name, err)
			}
			changes = append(changes, webhookutil.AgentChange{
				Path:    a.Path,
				Content: a.Content,
				Kind:    webhookutil.AgentChangeKindWrite,
			})
		case toolPatchNote:
			var a struct {
				Path    string `json:"path"`
				Find    string `json:"find"`
				Replace string `json:"replace"`
			}
			if err := json.Unmarshal([]byte(tc.Arguments), &a); err != nil {
				return nil, "", fmt.Errorf("exec: %s arguments: %w", tc.Name, err)
			}
			changes = append(changes, webhookutil.AgentChange{
				Path:    a.Path,
				Find:    a.Find,
				Replace: a.Replace,
				Kind:    webhookutil.AgentChangeKindPatch,
			})
		case toolFinish:
			answer = finishAnswer(tc.Arguments)
		default:
			return nil, "", fmt.Errorf("exec: unexpected tool call %q from exec endpoint", tc.Name)
		}
	}
	return changes, answer, nil
}

// execToolDef returns the ToolDef for the exec tool advertised to the model
// when ExecLLM is set.
func execToolDef() ToolDef {
	return ToolDef{
		Name:        toolExec,
		Description: "Run a code snippet. stdout MUST be JSON: {\"changes\":[{\"path\",\"content\"} or {\"path\",\"find\",\"replace\"}],\"answer\":\"...\"}. Writes go through write_patterns scope — out-of-scope paths are denied.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"program": map[string]any{
					"type":        "string",
					"description": "Program to run: python, bash, or node.",
				},
				"code": map[string]any{
					"type":        "string",
					"description": "Source code to execute.",
				},
			},
			"required": []string{"program", "code"},
		},
	}
}

// toolDefs returns the full set of ToolDefs offered to the model.
// The exec tool is included when execEnabled is true.
func toolDefs(execEnabled bool) []ToolDef {
	out := []ToolDef{
		{
			Name:        toolSearch,
			Description: "Find documents within your read scope.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        toolReadNote,
			Description: "Read a document's full content (read scope only).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        toolWriteNote,
			Description: "Create or replace a document (write scope only).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        toolPatchNote,
			Description: "Apply a surgical find→replace edit to a document (write scope only). Preserves surrounding content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"find":    map[string]any{"type": "string"},
					"replace": map[string]any{"type": "string"},
				},
				"required": []string{"path", "find", "replace"},
			},
		},
		{
			Name:        toolFinish,
			Description: "End the run with a final answer/summary.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"answer": map[string]any{"type": "string"},
				},
				"required": []string{"answer"},
			},
		},
	}
	if execEnabled {
		out = append(out, execToolDef())
	}
	return out
}
