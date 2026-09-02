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
// switch case. It returns the textual result for the model plus the outcome,
// the same contract as the built-in tools, so an extension write is metered
// and hard-failed exactly like write_note/patch_note.
// future: populate from MCP server descriptors, config, etc.
type toolInvoker func(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) (string, string)

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
	denialRead         = "read"
	denialWrite        = "write"
	denialNotPermitted = "not_permitted"
)

// Run status and token-kind label values reported to Metrics.
const (
	statusErrorForRun   = "error"
	tokenKindPrompt     = "prompt"
	tokenKindCompletion = "completion"
	// tokenKindCached is the share of tokenKindPrompt the provider served from
	// its prompt cache. It is a subset of the prompt tokens, so it is reported
	// but never added to the run's spend.
	tokenKindCached = "cached"
	// unknownTool is the label stand-in for a tool name the model invented. The
	// name is attacker-controllable, so it must never reach a metric label.
	unknownTool = "unknown"
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

	// AllowRoleAuthoring lets this run write notes whose frontmatter carries
	// fleet_id — that is, create or edit ROLE notes. Off by default: a role
	// declares its own write_patterns, so authoring one is privilege escalation
	// (see ErrRoleAuthoringDenied). Operators who genuinely want agents to
	// manage roles turn it on fleet-wide with --allow-role-authoring.
	AllowRoleAuthoring bool

	// Item is the 1-based fan-out index when one delivery runs the agent once per
	// item. It labels every run-log entry so an operator reading a fanned-out
	// delivery can tell whose call was whose. 0 for a single run.
	Item int

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
	// Logs is the run log: one entry per tool call, in the order they ran.
	// Denials duplicates the denied subset of it as flat strings, for the
	// callers that only ever wanted that.
	Logs []webhookutil.AgentLog `json:"logs,omitempty"`
}

// systemPromptTemplate is ordered most-stable-first so a provider with prefix
// caching can reuse it: the static preamble is identical for every run, the
// scope is fixed per role, and the per-delivery instruction comes last. Leading
// with the instruction — as this once did — leaves no shared prefix at all.
//
// The tools are deliberately NOT restated here: they already reach the model as
// the request's tool schemas, and a second prose copy both costs tokens on every
// step and drifts from the role's actual allowlist, inviting calls the runtime
// will only refuse.
const systemPromptTemplate = `You are a scoped trip2g micro-agent. The tools you were given are the only ones you have.

Rules:
- Reads or writes outside scope are rejected by the runtime; do not retry them.
- When done, call finish with a concise answer. Do not invent paths you have not seen.

Access scope (enforced by the runtime, not negotiable):
- You may read paths matching: %s
- You may write paths matching: %s

Instruction:
%s`

// Run executes the instruction through the LLM tool-call loop, enforcing scope
// and the token hard-cap, and returns the answer plus any in-scope changes. It
// wraps run so every exit path — including the error ones — is measured once.
func Run(ctx context.Context, in Input) (*Result, error) {
	started := time.Now()
	// res is allocated here, not in run, so a run that fails mid-loop still
	// reports the steps and spend it already consumed. The caller still gets the
	// nil-result-on-error contract.
	res := &Result{Status: StatusMaxSteps}
	err := run(ctx, in, res)
	recordRun(in, res, err, time.Since(started))
	if err != nil {
		return nil, err
	}
	return res, nil
}

// recordRun reports the run's terminal status, steps and duration.
func recordRun(in Input, res *Result, err error, elapsed time.Duration) {
	if in.Metrics == nil {
		return
	}
	status := res.Status
	if err != nil {
		status = statusErrorForRun
	}
	in.Metrics.RecordRun(in.Role, status, res.Steps, elapsed.Seconds())
}

func run(ctx context.Context, in Input, res *Result) error {
	if in.LLM == nil {
		return errors.New("agentruntime: LLM is required")
	}
	if in.KB == nil {
		return errors.New("agentruntime: KB is required")
	}
	if in.MaxTokens <= 0 {
		return errors.New("agentruntime: MaxTokens must be > 0")
	}
	if in.MaxSteps <= 0 {
		return errors.New("agentruntime: MaxSteps must be > 0")
	}

	scoped := NewScopedKB(in.KB, in.ReadPatterns, in.WritePatterns)
	scoped.allowRoleAuthoring = in.AllowRoleAuthoring
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
				formatPatterns(in.ReadPatterns),
				formatPatterns(in.WritePatterns),
				in.Instruction,
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

	for step := range in.MaxSteps {
		// Hard-cap check happens BEFORE each model call: once spent, stop.
		if res.TokensUsed >= in.MaxTokens {
			res.Status = StatusCapped
			return nil
		}

		chat, err := chatWithBudget(ctx, in.LLM, in.Model, messages, tools, in.MaxTokens-res.TokensUsed)
		if err != nil {
			return fmt.Errorf("agentruntime: chat step %d: %w", step, err)
		}
		res.Steps++
		res.TokensUsed += chat.PromptTokens + chat.CompletionTokens
		if in.Metrics != nil {
			in.Metrics.RecordTokens(in.Model, in.Role, tokenKindPrompt, chat.PromptTokens)
			in.Metrics.RecordTokens(in.Model, in.Role, tokenKindCompletion, chat.CompletionTokens)
			in.Metrics.RecordTokens(in.Model, in.Role, tokenKindCached, chat.CachedPromptTokens)
		}

		// No tool calls means the model returned a final answer.
		if len(chat.ToolCalls) == 0 {
			res.Answer = chat.Content
			res.Status = StatusCompleted
			return nil
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
			step:      step + 1,
			item:      in.Item,
		}, chat.ToolCalls)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}

	return nil
}

// toolLoop bundles the per-run state one assistant turn's tool calls act on.
type toolLoop struct {
	scoped    *ScopedKB
	res       *Result
	permitted map[string]bool
	invokers  map[string]toolInvoker
	messages  *[]Message
	// step is the 1-based model turn these calls came from; it labels every row
	// the turn adds to the run log. item is the run's fan-out index, or 0.
	step int
	item int
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
			if in.Metrics != nil {
				in.Metrics.RecordToolCall(toolFinish, outcomeOK)
			}
			logToolCall(tl.res, tl.item, tl.step, call, outcomeOK, "", 0)
			return true, nil
		}
		// Execution-time allowlist enforcement: reject any tool not in the
		// permitted set, even if the model hallucinated or was injected.
		if !tl.permitted[call.Name] {
			denial := "tool not permitted: " + call.Name
			tl.res.Denials = append(tl.res.Denials, denial)
			if in.Metrics != nil {
				// The rejected name is model-supplied: bucket it instead of
				// minting a metric series per hallucination.
				in.Metrics.RecordToolCall(unknownTool, outcomeNotPermitted)
				in.Metrics.RecordDenial(denialNotPermitted)
			}
			logToolCall(tl.res, tl.item, tl.step, call, outcomeNotPermitted, denial, 0)
			*tl.messages = append(*tl.messages, Message{
				Role:       RoleTool,
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    "error: " + denial,
			})
			continue
		}
		started := time.Now()
		denialsBefore := len(tl.res.Denials)
		output, outcome := execTool(ctx, tl.scoped, tl.res, call, tl.invokers)
		logToolCall(tl.res, tl.item, tl.step, call, outcome, output, time.Since(started))
		if in.Metrics != nil {
			in.Metrics.RecordToolCall(call.Name, outcome)
			// One denial per refused write, not per call: an exec batch can
			// refuse several, and still refuse some when its outcome is the
			// apply failure that followed.
			for range tl.res.Denials[denialsBefore:] {
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
		return fn(ctx, scoped, res, call)
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

// logToolCall appends one run-log entry for a finished tool call. The opaque
// Data bag carries this runtime's own vocabulary — tool, outcome, sizes — which
// nothing downstream parses; trip2g stores it and hands it back. Content never
// goes in: only how many bytes moved, so the log stays small and the written
// bytes are read back through the note version the write produced.
func logToolCall(res *Result, item, step int, call ToolCall, outcome, output string, elapsed time.Duration) {
	data := map[string]any{
		"tool":    call.Name,
		"step":    step,
		"outcome": outcome,
		"ms":      elapsed.Milliseconds(),
	}
	if item > 0 {
		data["item"] = item
	}

	// The arguments worth seeing, by the shape each tool takes. An unknown tool
	// (a role's MCP server, say) contributes none of these: its argument names
	// are not this runtime's to guess, and its values may hold anything.
	var args struct {
		Path    string `json:"path"`
		Query   string `json:"query"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal([]byte(call.Arguments), &args)
	if args.Path != "" {
		data["path"] = args.Path
	}
	if args.Query != "" {
		data["query"] = args.Query
	}
	if args.Content != "" {
		data["content_bytes"] = len(args.Content)
	}

	msg := call.Name
	if args.Path != "" {
		msg += " " + args.Path
	}
	if outcome == outcomeOK {
		data["result_bytes"] = len(output)
	} else {
		msg += ": " + outcome
		data["reason"] = strings.TrimPrefix(output, "error: ")
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		// A bag that will not serialize must not cost the operator the entry.
		encoded = nil
	}
	res.Logs = append(res.Logs, webhookutil.AgentLog{
		TS:    time.Now().UTC(),
		Level: levelForOutcome(outcome),
		Msg:   msg,
		Data:  encoded,
	})
}

// levelForOutcome grades an outcome for readers that only skim levels. A denial
// is a warning, not an error: the runtime feeds it back and the model is
// expected to correct itself. Only a failure the run cannot absorb is an error.
func levelForOutcome(outcome string) string {
	switch outcome {
	case outcomeOK:
		return "info"
	case outcomeError, outcomeApplyFailed:
		return "error"
	default:
		return "warn"
	}
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

// writeDenial classifies a ScopedKB write/patch error as a denial and builds the
// operator-facing run-log line. ErrRoleGuardUnverifiable is deliberately absent:
// it is not an authorization decision but a failure to check, most often a patch
// against a note that does not exist. Reporting it as a denial would fill the
// denial log with false role-authoring hits and hide trip2g's real reason, so it
// falls through to the apply-failure path it took before the guard existed. A denial is soft: the loop reports it and keeps
// going. It is deliberately NOT folded into the generic apply-failure path,
// because a denial the operator cannot see is the known failure mode here — the
// model paraphrases the tool error as its own refusal and the real cause never
// surfaces (see Role.WarnIfWriteScopeMisconfigured).
func writeDenial(err error, verb, path string) (string, bool) {
	switch {
	case errors.Is(err, ErrWriteDenied):
		return verb + " " + path, true
	case errors.Is(err, ErrRoleAuthoringDenied):
		return verb + " " + path + ": role note (fleet_id) — agents may not author roles", true
	default:
		return "", false
	}
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
	if denial, ok := writeDenial(err, "write", args.Path); ok {
		res.Denials = append(res.Denials, denial)
		return "error: " + err.Error(), outcomeDenied
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
	if denial, ok := writeDenial(err, "patch", args.Path); ok {
		res.Denials = append(res.Denials, denial)
		return "error: " + err.Error(), outcomeDenied
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
// as write_note. Every change is validated before any is applied: out-of-scope
// writes are trimmed, recorded and named in the result (a denial stays soft,
// as for write_note), while a change that cannot apply — a bad or non-unique
// patch find — refuses the whole batch as an apply failure, so a run cannot
// end with half its writes committed. A KB failing after validation is an
// apply failure too, with only the changes that landed reported. Program
// allowlisting and sandboxing are codellm-authoritative: a disallowed program
// or failing block comes back as a deterministic error (422), surfaced to the
// model as a soft tool error.
func makeExecInvoker(execLLM LLM) toolInvoker {
	return func(ctx context.Context, scoped *ScopedKB, res *Result, call ToolCall) (string, string) {
		var args struct {
			Program string `json:"program"`
			Code    string `json:"code"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "error: invalid arguments: " + err.Error(), outcomeInvalidArgs
		}
		body, err := fenceCodeBlock(args.Program, args.Code)
		if err != nil {
			return "error: " + err.Error(), outcomeInvalidArgs
		}
		// No fleet_input message: exec runs inline; the model already has context.
		// No explicit timeout: the call is bounded by the parent run context.
		chat, err := execLLM.Chat(ctx, execModel, []Message{{Role: RoleUser, Content: body}}, nil)
		if err != nil {
			return "error: " + err.Error(), outcomeError
		}
		changes, answer, perr := changesFromToolCalls(chat.ToolCalls)
		if perr != nil {
			return "error: " + perr.Error(), outcomeError
		}
		batch := execBatch{scoped: scoped, content: map[string]string{}}
		var prepared []preparedChange
		var denied []string
		for _, ch := range changes {
			pc, verr := batch.prepare(ctx, ch)
			if denial, ok := writeDenial(verr, "exec write", ch.Path); ok {
				res.Denials = append(res.Denials, denial)
				denied = append(denied, ch.Path+": "+verr.Error())
				continue
			}
			if verr != nil {
				return "error: apply " + ch.Path + ": " + verr.Error(), outcomeApplyFailed
			}
			prepared = append(prepared, pc)
		}
		for _, pc := range prepared {
			if aerr := applyExecChange(ctx, scoped, pc); aerr != nil {
				return "error: apply " + pc.change.Path + ": " + aerr.Error(), outcomeApplyFailed
			}
			res.Changes = append(res.Changes, pc.change)
		}
		summary := fmt.Sprintf("ok: ran %s, %d write(s)", args.Program, len(prepared))
		outcome := outcomeOK
		if len(denied) > 0 {
			summary += fmt.Sprintf(", %d denied (%s)", len(denied), strings.Join(denied, ", "))
			outcome = outcomeDenied
		}
		if answer != "" {
			summary += "; " + answer
		}
		return summary, outcome
	}
}

// preparedChange is an exec change that passed validation and is ready to
// apply: the normalized path the KB receives and, for a patch, the note bytes
// it was checked against, which the apply is conditioned on.
type preparedChange struct {
	change   webhookutil.AgentChange
	path     string
	verified *string
}

// execBatch validates the changes one exec call returned, in order, before any
// is applied. content carries what each change leaves behind to the checks of
// the later ones, so a patch after a write or patch of the same note is judged
// — by the role guard, by the uniqueness of its find, and by the bytes its
// conditional apply is pinned to — against what the KB will hold when it runs:
// the same view applying one at a time would have given.
//
// Unlike Patch, the exec lane conditions every patch on the bytes it checked,
// guard on or off: the uniqueness pre-check reads through remoteKB's overlay,
// seeded once per delivery, so a stale overlay must fail loudly as a hash
// mismatch rather than refuse (or pass) a patch the live note would not.
type execBatch struct {
	scoped  *ScopedKB
	content map[string]string
}

// prepare runs Write's or Patch's own pre-flight on one change without
// applying it. A patch must also find its match exactly once — what trip2g
// would refuse server-side, after earlier changes had already landed.
func (b *execBatch) prepare(ctx context.Context, ch webhookutil.AgentChange) (preparedChange, error) {
	if ch.Kind != webhookutil.AgentChangeKindPatch {
		norm, err := b.scoped.checkWrite(ch.Path, ch.Content)
		if err != nil {
			return preparedChange{}, err
		}
		b.content[norm] = ch.Content
		return preparedChange{change: ch, path: norm}, nil
	}
	norm, err := b.scoped.checkWriteScope(ch.Path)
	if err != nil {
		return preparedChange{}, err
	}
	current, seen := b.content[norm]
	if !seen {
		current, err = b.scoped.kb.Read(ctx, norm)
		if err != nil {
			return preparedChange{}, err
		}
	}
	if gerr := b.scoped.guardPatch(current, ch.Find, ch.Replace); gerr != nil {
		return preparedChange{}, gerr
	}
	patched, ok := applyPatchPreview(current, ch.Find, ch.Replace)
	if !ok {
		return preparedChange{}, fmt.Errorf("patch find not found in %s: %q", ch.Path, ch.Find)
	}
	b.content[norm] = patched
	return preparedChange{change: ch, path: norm, verified: &current}, nil
}

// applyExecChange is the second half of execBatch.prepare: the write or patch
// itself, past the checks already made.
func applyExecChange(ctx context.Context, scoped *ScopedKB, pc preparedChange) error {
	if pc.change.Kind == webhookutil.AgentChangeKindPatch {
		return scoped.applyPatch(ctx, pc.path, pc.change.Find, pc.change.Replace, pc.verified)
	}
	return scoped.kb.Write(ctx, pc.path, pc.change.Content)
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
			Description: "Apply a surgical find→replace edit to a document (write scope only). Preserves surrounding content. Fails if find is absent or occurs more than once, so include enough surrounding context to make the match unique.",
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
