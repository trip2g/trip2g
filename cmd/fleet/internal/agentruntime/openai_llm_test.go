package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// llmStubServer is an httptest server speaking the OpenAI chat-completions
// wire shape. respond is called per request with the attempt number (1-based)
// and the decoded request body; it returns the HTTP status and response JSON.
func llmStubServer(t *testing.T, respond func(attempt int, body map[string]any) (int, string)) (*OpenAILLM, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		status, resp := respond(int(n), body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, resp)
	}))
	t.Cleanup(srv.Close)
	return NewOpenAILLM("test-key", srv.URL+"/v1"), &calls
}

const okCompletion = `{"id":"1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`

// TestOpenAILLM_RetriesTransientThenSucceeds: a 429 then a 500 must be retried;
// the third attempt succeeds and its result is returned.
func TestOpenAILLM_RetriesTransientThenSucceeds(t *testing.T) {
	llm, calls := llmStubServer(t, func(attempt int, _ map[string]any) (int, string) {
		switch attempt {
		case 1:
			return http.StatusTooManyRequests, `{"error":{"message":"rate limited","type":"rate_limit_exceeded"}}`
		case 2:
			return http.StatusInternalServerError, `{"error":{"message":"boom","type":"server_error"}}`
		default:
			return http.StatusOK, okCompletion
		}
	})

	res, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.NoError(t, err)
	require.Equal(t, "hi", res.Content)
	require.Equal(t, 7, res.PromptTokens)
	require.Equal(t, 3, res.CompletionTokens)
	require.Equal(t, int32(3), calls.Load())
}

// TestOpenAILLM_GivesUpAfterMaxAttempts: persistent 5xx exhausts the bounded
// retry budget and surfaces the last error.
func TestOpenAILLM_GivesUpAfterMaxAttempts(t *testing.T) {
	llm, calls := llmStubServer(t, func(_ int, _ map[string]any) (int, string) {
		return http.StatusBadGateway, `{"error":{"message":"upstream down","type":"server_error"}}`
	})

	_, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "giving up after 3 attempts")
	require.Equal(t, int32(llmMaxAttempts), calls.Load())
}

// TestOpenAILLM_DoesNotRetryClientError: a 400 is terminal — exactly one request.
func TestOpenAILLM_DoesNotRetryClientError(t *testing.T) {
	llm, calls := llmStubServer(t, func(_ int, _ map[string]any) (int, string) {
		return http.StatusBadRequest, `{"error":{"message":"bad request","type":"invalid_request_error"}}`
	})

	_, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
}

// TestOpenAILLM_EmptyChoicesIsError: a 200 with zero choices must not be
// treated as a silent empty completion. Usage still comes back so the run's
// token accounting stays correct.
func TestOpenAILLM_EmptyChoicesIsError(t *testing.T) {
	llm, _ := llmStubServer(t, func(_ int, _ map[string]any) (int, string) {
		return http.StatusOK, `{"id":"1","object":"chat.completion","choices":[],"usage":{"prompt_tokens":7,"completion_tokens":0,"total_tokens":7}}`
	})

	res, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.ErrorIs(t, err, ErrEmptyCompletion)
	require.Equal(t, 7, res.PromptTokens)
}

// TestOpenAILLM_ChatWithBudgetCapsCompletion: the remaining run budget is
// forwarded as max_completion_tokens so the provider itself binds the ceiling.
func TestOpenAILLM_ChatWithBudgetCapsCompletion(t *testing.T) {
	var gotBudget float64
	llm, _ := llmStubServer(t, func(_ int, body map[string]any) (int, string) {
		gotBudget, _ = body["max_completion_tokens"].(float64)
		return http.StatusOK, okCompletion
	})

	_, err := llm.ChatWithBudget(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil, 1234)
	require.NoError(t, err)
	require.Equal(t, 1234, int(gotBudget))
}

// TestOpenAILLM_PlainChatOmitsBudget: Chat (no budget) must not send a cap.
func TestOpenAILLM_PlainChatOmitsBudget(t *testing.T) {
	var hasBudget bool
	llm, _ := llmStubServer(t, func(_ int, body map[string]any) (int, string) {
		_, hasBudget = body["max_completion_tokens"]
		return http.StatusOK, okCompletion
	})

	_, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.NoError(t, err)
	require.False(t, hasBudget)
}

// cachedCompletion carries the usage breakdown every OpenAI-compatible
// provider with prefix caching reports: cached_tokens is the share of
// prompt_tokens that was served from cache, not spend on top of it.
const cachedCompletion = `{"id":"1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2048,"completion_tokens":3,"total_tokens":2051,"prompt_tokens_details":{"cached_tokens":1920}}}`

// TestOpenAILLM_ReportsCachedPromptTokens: the cache-hit share is read off the
// usage breakdown so cache effectiveness is measurable on any provider that
// reports it, with no provider-specific request field.
func TestOpenAILLM_ReportsCachedPromptTokens(t *testing.T) {
	llm, _ := llmStubServer(t, func(int, map[string]any) (int, string) {
		return http.StatusOK, cachedCompletion
	})

	res, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.NoError(t, err)
	require.Equal(t, 2048, res.PromptTokens)
	require.Equal(t, 1920, res.CachedPromptTokens)
}

// TestOpenAILLM_CachedPromptTokensAbsentIsZero: a provider that omits the
// breakdown (most local runtimes) must not crash the call.
func TestOpenAILLM_CachedPromptTokensAbsentIsZero(t *testing.T) {
	llm, _ := llmStubServer(t, func(int, map[string]any) (int, string) {
		return http.StatusOK, okCompletion
	})

	res, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.NoError(t, err)
	require.Zero(t, res.CachedPromptTokens)
}

// idempotencyStubServer records the Idempotency-Key header of every request and
// fails the first two attempts of each call with a 500. keys returns a snapshot.
func idempotencyStubServer(t *testing.T) (*OpenAILLM, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		n := len(keys)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if n%llmMaxAttempts != 0 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":{"message":"boom","type":"server_error"}}`)
			return
		}
		fmt.Fprint(w, okCompletion)
	}))
	t.Cleanup(srv.Close)
	snapshot := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), keys...)
	}
	return NewOpenAILLM("test-key", srv.URL+"/v1"), snapshot
}

// TestOpenAILLM_IdempotencyKeyStableAcrossRetries: every attempt of one Chat
// call carries the same non-empty Idempotency-Key, so a code-executing endpoint
// can recognise a replay; a second call carries a different key.
func TestOpenAILLM_IdempotencyKeyStableAcrossRetries(t *testing.T) {
	llm, keys := idempotencyStubServer(t)
	msgs := []Message{{Role: RoleUser, Content: "x"}}

	_, err := llm.Chat(context.Background(), "test-model", msgs, nil)
	require.NoError(t, err)
	got := keys()
	require.Len(t, got, llmMaxAttempts)
	first := got[0]
	require.NotEmpty(t, first)
	for _, k := range got {
		require.Equal(t, first, k, "all attempts of one call must share the key")
	}

	_, err = llm.ChatWithBudget(context.Background(), "test-model", msgs, nil, 10)
	require.NoError(t, err)
	got = keys()
	require.Len(t, got, 2*llmMaxAttempts)
	second := got[llmMaxAttempts]
	require.NotEmpty(t, second)
	require.NotEqual(t, first, second, "different calls must carry different keys")
	for _, k := range got[llmMaxAttempts:] {
		require.Equal(t, second, k)
	}
}
