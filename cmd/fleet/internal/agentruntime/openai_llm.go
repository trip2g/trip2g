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

// LLMMetrics is the optional provider-call sink. Declared here, minimally, so
// agentruntime never imports the fleet's metrics package. Implementations must
// be nil-safe.
type LLMMetrics interface {
	RecordLLMRequest(lane, model, status string, seconds float64)
	RecordLLMRetry(lane, reason string)
}

// Retry/failure reasons reported to LLMMetrics.
const (
	reasonRateLimited = "429"
	reasonServer      = "5xx"
	reasonClient      = "4xx"
	reasonCanceled    = "canceled"
	reasonTimeout     = "timeout"
	reasonNetwork     = "network"
	reasonEmpty       = "empty_completion"
	statusOK          = "ok"
)

// OpenAILLM is the production LLM backed by any OpenAI-compatible chat
// completions endpoint. BaseURL is the configurable knob: leave empty for the
// default OpenAI endpoint, or point it at a local/vendor model
// (e.g. "http://localhost:11434/v1" for Ollama).
//
// Transient failures (HTTP 429/5xx, network errors) are retried with
// exponential backoff, bounded by llmMaxAttempts and the caller's context.
type OpenAILLM struct {
	client  *goopenai.Client
	metrics LLMMetrics
	lane    string
}

// SetMetrics attaches the metrics sink and names the lane this client serves
// (fleetmetrics.LaneLLM for the role's model, LaneExec for codellm). Kept a
// setter so the constructor signature — and every existing call site — stays
// as it was.
func (l *OpenAILLM) SetMetrics(m LLMMetrics, lane string) {
	l.metrics = m
	l.lane = lane
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

	started := time.Now()
	resp, err := l.createWithRetry(ctx, req)
	if err != nil {
		l.recordRequest(model, err, time.Since(started))
		return ChatResult{}, err
	}

	out := ChatResult{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
	}
	if len(resp.Choices) == 0 {
		// A 200 with no choices is a provider-side failure, not a success: record
		// it as one before handing ErrEmptyCompletion back.
		l.recordRequest(model, ErrEmptyCompletion, time.Since(started))
		return out, ErrEmptyCompletion
	}
	l.recordRequest(model, nil, time.Since(started))
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
		// Only count it as a retry when another attempt actually follows; the
		// last failure is the request's outcome, not a retry.
		if l.metrics != nil && attempt < llmMaxAttempts-1 {
			l.metrics.RecordLLMRetry(l.lane, llmErrorReason(err))
		}
		lastErr = err
	}
	return goopenai.ChatCompletionResponse{}, fmt.Errorf("llm: giving up after %d attempts: %w", llmMaxAttempts, lastErr)
}

// recordRequest reports one completed chat call (all its attempts included).
func (l *OpenAILLM) recordRequest(model string, err error, elapsed time.Duration) {
	if l.metrics == nil {
		return
	}
	status := statusOK
	if err != nil {
		status = llmErrorReason(err)
	}
	l.metrics.RecordLLMRequest(l.lane, model, status, elapsed.Seconds())
}

// llmErrorReason maps a provider error onto a bounded label: the HTTP class it
// came back as, a cancellation, or a transport failure.
func llmErrorReason(err error) string {
	if errors.Is(err, ErrEmptyCompletion) {
		return reasonEmpty
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return reasonTimeout
	}
	if errors.Is(err, context.Canceled) {
		return reasonCanceled
	}
	var apiErr *goopenai.APIError
	if errors.As(err, &apiErr) {
		return httpStatusReason(apiErr.HTTPStatusCode)
	}
	var reqErr *goopenai.RequestError
	if errors.As(err, &reqErr) {
		return httpStatusReason(reqErr.HTTPStatusCode)
	}
	return reasonNetwork
}

// httpStatusReason buckets an HTTP status into the reason labels.
func httpStatusReason(code int) string {
	switch {
	case code == http.StatusTooManyRequests:
		return reasonRateLimited
	case code >= http.StatusInternalServerError:
		return reasonServer
	case code >= http.StatusBadRequest:
		return reasonClient
	default:
		return reasonNetwork
	}
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
