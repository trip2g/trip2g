package agentruntime

import "context"

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// ToolCall is one tool invocation requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // raw JSON arguments.
}

// Message is one chat turn, provider-independent so tests can drive the loop
// with a stub LLM.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall // set on assistant turns that request tools.
	ToolCallID string     // set on tool-result turns; echoes the call ID.
	Name       string     // tool name on tool-result turns.
}

// ToolDef declares a tool exposed to the model. Parameters is a JSON-schema
// object.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// ChatResult is one model response plus its token usage.
type ChatResult struct {
	Content          string
	ToolCalls        []ToolCall
	PromptTokens     int
	CompletionTokens int

	// CachedPromptTokens is the share of PromptTokens the provider served from
	// its prompt cache — a subset of PromptTokens, never spend on top of it.
	// Zero when the provider reports no breakdown, which is not evidence of a
	// miss: most local runtimes cache and say nothing.
	CachedPromptTokens int
}

// LLM is the provider-independent chat surface the executor depends on. The
// real implementation (OpenAILLM) talks to any OpenAI-compatible endpoint; the
// tests pass a deterministic stub.
type LLM interface {
	Chat(ctx context.Context, model string, messages []Message, tools []ToolDef) (ChatResult, error)
}

// BudgetedLLM is an optional LLM extension: providers that can cap a single
// completion receive the run's remaining token budget per call, so the
// TokenCeiling binds at the provider, not just in the loop's post-hoc check.
type BudgetedLLM interface {
	ChatWithBudget(ctx context.Context, model string, messages []Message, tools []ToolDef, maxCompletionTokens int) (ChatResult, error)
}
