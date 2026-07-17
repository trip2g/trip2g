package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	goopenai "github.com/sashabaranov/go-openai"
)

// ErrEmptyCompletion is returned when the provider answers 200 with no choices.
// Treating it as success would feed an empty assistant turn back into the loop
// and mask provider-side failures (filtered content, truncated batches).
var ErrEmptyCompletion = errors.New("llm: empty completion (no choices)")

const (
	llmMaxAttempts  = 3
	llmRetryBaseGap = 500 * time.Millisecond
)

// OpenAILLM is the production LLM backed by any OpenAI-compatible chat
// completions endpoint. BaseURL is the configurable knob: leave empty for the
// default OpenAI endpoint, or point it at a local/vendor model
// (e.g. "http://localhost:11434/v1" for Ollama).
//
// Transient failures (HTTP 429/5xx, network errors) are retried with
// exponential backoff, bounded by llmMaxAttempts and the caller's context.
type OpenAILLM struct {
	client *goopenai.Client
}

// NewOpenAILLM builds an OpenAILLM. apiKey is the bearer credential; baseURL,
// when non-empty, overrides the default endpoint.
func NewOpenAILLM(apiKey, baseURL string) *OpenAILLM {
	cfg := goopenai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &OpenAILLM{client: goopenai.NewClientWithConfig(cfg)}
}

// Chat issues one chat-completion request with the given tools.
func (l *OpenAILLM) Chat(ctx context.Context, model string, messages []Message, tools []ToolDef) (ChatResult, error) {
	return l.chat(ctx, model, messages, tools, 0)
}

// ChatWithBudget is the BudgetedLLM hook: maxCompletionTokens caps this single
// completion so one runaway call cannot blow far past the run's token ceiling.
func (l *OpenAILLM) ChatWithBudget(ctx context.Context, model string, messages []Message, tools []ToolDef, maxCompletionTokens int) (ChatResult, error) {
	return l.chat(ctx, model, messages, tools, maxCompletionTokens)
}

func (l *OpenAILLM) chat(ctx context.Context, model string, messages []Message, tools []ToolDef, maxCompletionTokens int) (ChatResult, error) {
	req := goopenai.ChatCompletionRequest{
		Model:    model,
		Messages: toOpenAIMessages(messages),
		Tools:    toOpenAITools(tools),
	}
	if maxCompletionTokens > 0 {
		req.MaxCompletionTokens = maxCompletionTokens
	}

	resp, err := l.createWithRetry(ctx, req)
	if err != nil {
		return ChatResult{}, err
	}

	out := ChatResult{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
	}
	if len(resp.Choices) == 0 {
		return out, ErrEmptyCompletion
	}
	msg := resp.Choices[0].Message
	out.Content = msg.Content
	for _, tc := range msg.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
}

// createWithRetry retries transient failures with exponential backoff
// (500ms, 1s). Non-retryable errors (4xx other than 429, context
// cancellation) surface immediately.
func (l *OpenAILLM) createWithRetry(ctx context.Context, req goopenai.ChatCompletionRequest) (goopenai.ChatCompletionResponse, error) {
	var lastErr error
	for attempt := range llmMaxAttempts {
		if attempt > 0 {
			gap := llmRetryBaseGap << (attempt - 1)
			select {
			case <-ctx.Done():
				return goopenai.ChatCompletionResponse{}, ctx.Err()
			case <-time.After(gap):
			}
		}
		resp, err := l.client.CreateChatCompletion(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !isRetryableLLMError(ctx, err) {
			return goopenai.ChatCompletionResponse{}, err
		}
		lastErr = err
	}
	return goopenai.ChatCompletionResponse{}, fmt.Errorf("llm: giving up after %d attempts: %w", llmMaxAttempts, lastErr)
}

// isRetryableLLMError reports whether err is worth another attempt: rate
// limiting (429), server-side failures (5xx), or transport-level errors.
// Context cancellation and other 4xx (bad request, auth) are terminal.
func isRetryableLLMError(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *goopenai.APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode == http.StatusTooManyRequests || apiErr.HTTPStatusCode >= http.StatusInternalServerError
	}
	var reqErr *goopenai.RequestError
	if errors.As(err, &reqErr) {
		return reqErr.HTTPStatusCode == http.StatusTooManyRequests || reqErr.HTTPStatusCode >= http.StatusInternalServerError
	}
	// Non-HTTP failure (connection refused, reset, DNS) — transient by nature.
	return true
}

func toOpenAIMessages(messages []Message) []goopenai.ChatCompletionMessage {
	out := make([]goopenai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		om := goopenai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, goopenai.ToolCall{
				ID:   tc.ID,
				Type: goopenai.ToolTypeFunction,
				Function: goopenai.FunctionCall{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		out = append(out, om)
	}
	return out
}

func toOpenAITools(tools []ToolDef) []goopenai.Tool {
	out := make([]goopenai.Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, goopenai.Tool{
			Type: goopenai.ToolTypeFunction,
			Function: &goopenai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return out
}
