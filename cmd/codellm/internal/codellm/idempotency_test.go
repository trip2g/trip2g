package codellm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

// keyedRequest builds a chat-completions request carrying an Idempotency-Key.
func keyedRequest(t *testing.T, key string, messages []goopenai.ChatCompletionMessage) *http.Request {
	t.Helper()
	body, err := json.Marshal(goopenai.ChatCompletionRequest{Model: "codellm", Messages: messages})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set(idempotencyKeyHeader, key)
	return req
}

// doChatKeyed is doChat with an Idempotency-Key header.
func doChatKeyed(t *testing.T, srv *Server, key string, messages []goopenai.ChatCompletionMessage) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, keyedRequest(t, key, messages))
	return rec
}

// countingBlock returns a bash block that appends one line to the counter file
// on every execution, then emits the given stdout contract.
func countingBlock(t *testing.T, contract string) (goopenai.ChatCompletionMessage, func() int) {
	t.Helper()
	counter := filepath.Join(t.TempDir(), "runs")
	return bashBody(`echo run >> "` + counter + `"; ` + contract), func() int {
		data, err := os.ReadFile(counter)
		if os.IsNotExist(err) {
			return 0
		}
		require.NoError(t, err)
		return bytes.Count(data, []byte("\n"))
	}
}

const okContract = `echo '{"changes":[{"path":"notes/a.md","content":"x"}],"answer":"done"}'`

// TestIdempotency_ReplayServesRecordedResponse: the same key returns the
// byte-identical response and the block runs once.
func TestIdempotency_ReplayServesRecordedResponse(t *testing.T) {
	srv := newTestServer()
	block, runs := countingBlock(t, okContract)
	msgs := []goopenai.ChatCompletionMessage{block}

	first := doChatKeyed(t, srv, "key-1", msgs)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	replay := doChatKeyed(t, srv, "key-1", msgs)

	require.Equal(t, first.Code, replay.Code)
	require.Equal(t, first.Header().Get("Content-Type"), replay.Header().Get("Content-Type"))
	require.Equal(t, first.Body.Bytes(), replay.Body.Bytes(), "replay must be byte-identical")
	require.Equal(t, 1, runs(), "the block must execute exactly once")
}

// TestIdempotency_DifferentKeyExecutesAgain: a new key is a new call.
func TestIdempotency_DifferentKeyExecutesAgain(t *testing.T) {
	srv := newTestServer()
	block, runs := countingBlock(t, okContract)
	msgs := []goopenai.ChatCompletionMessage{block}

	require.Equal(t, http.StatusOK, doChatKeyed(t, srv, "key-a", msgs).Code)
	require.Equal(t, http.StatusOK, doChatKeyed(t, srv, "key-b", msgs).Code)
	require.Equal(t, 2, runs())
}

// TestIdempotency_NoKeyExecutesEveryTime: without the header the endpoint
// behaves as before — every request executes.
func TestIdempotency_NoKeyExecutesEveryTime(t *testing.T) {
	srv := newTestServer()
	block, runs := countingBlock(t, okContract)
	msgs := []goopenai.ChatCompletionMessage{block}

	require.Equal(t, http.StatusOK, doChat(t, srv, msgs).Code)
	require.Equal(t, http.StatusOK, doChat(t, srv, msgs).Code)
	require.Equal(t, 2, runs())
}

// TestIdempotency_ReplayOfFailureDoesNotReexecute: a failed execution is
// recorded too; the replay gets the same error without running the block.
func TestIdempotency_ReplayOfFailureDoesNotReexecute(t *testing.T) {
	srv := newTestServer()
	block, runs := countingBlock(t, `echo boom >&2; exit 7`)
	msgs := []goopenai.ChatCompletionMessage{block}

	first := doChatKeyed(t, srv, "key-fail", msgs)
	require.Equal(t, http.StatusUnprocessableEntity, first.Code)
	replay := doChatKeyed(t, srv, "key-fail", msgs)

	require.Equal(t, http.StatusUnprocessableEntity, replay.Code)
	require.Equal(t, first.Body.Bytes(), replay.Body.Bytes())
	require.Equal(t, 1, runs())
}

// TestIdempotency_ConcurrentReplayWaitsForInFlight: a replay that arrives
// while the first request is still executing waits for it and receives the
// same response instead of starting a second execution.
func TestIdempotency_ConcurrentReplayWaitsForInFlight(t *testing.T) {
	srv := newTestServer()
	gate := filepath.Join(t.TempDir(), "go")
	block, runs := countingBlock(t, `while [ ! -f "`+gate+`" ]; do sleep 0.02; done; `+okContract)
	msgs := []goopenai.ChatCompletionMessage{block}

	handler := srv.Handler()
	serve := func(req *http.Request) chan *httptest.ResponseRecorder {
		ch := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			ch <- rec
		}()
		return ch
	}

	firstCh := serve(keyedRequest(t, "key-c", msgs))
	require.Eventually(t, func() bool {
		srv.replays.mu.Lock()
		defer srv.replays.mu.Unlock()
		return len(srv.replays.entries) == 1
	}, 2*time.Second, 5*time.Millisecond, "first request must be registered as in flight")

	replayCh := serve(keyedRequest(t, "key-c", msgs))
	select {
	case <-replayCh:
		t.Fatal("replay must wait for the in-flight execution")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, os.WriteFile(gate, nil, 0o600))
	first := <-firstCh
	replay := <-replayCh
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	require.Equal(t, first.Body.Bytes(), replay.Body.Bytes())
	require.Equal(t, 1, runs())
}

// TestIdempotency_ExpiredEntryExecutesAgain: past the TTL the key is
// forgotten and the request executes.
func TestIdempotency_ExpiredEntryExecutesAgain(t *testing.T) {
	srv := newTestServer()
	now := time.Now()
	srv.replays.now = func() time.Time { return now }
	block, runs := countingBlock(t, okContract)
	msgs := []goopenai.ChatCompletionMessage{block}

	require.Equal(t, http.StatusOK, doChatKeyed(t, srv, "key-ttl", msgs).Code)
	now = now.Add(srv.replays.ttl + time.Second)
	require.Equal(t, http.StatusOK, doChatKeyed(t, srv, "key-ttl", msgs).Code)
	require.Equal(t, 2, runs())
	require.Len(t, srv.replays.entries, 1, "the expired entry must have been swept")
}

// TestMetrics_ReplayIsCounted asserts a served replay lands in the replay
// counter and the executed-request series does not grow.
func TestMetrics_ReplayIsCounted(t *testing.T) {
	srv, m := newMeteredTestServer()
	msgs := []goopenai.ChatCompletionMessage{bashBody(okContract)}
	require.Equal(t, http.StatusOK, doChatKeyed(t, srv, "key-m", msgs).Code)
	require.Equal(t, http.StatusOK, doChatKeyed(t, srv, "key-m", msgs).Code)

	out := scrape(t, m)
	require.Contains(t, out, `codellm_exec_replays_total 1`)
	require.Contains(t, out, `codellm_blocks_total{outcome="ok",program="bash"} 1`)
}
