package codellm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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

// TestIdempotency_CancelledFirstRequestReplaysItsFailure is the headline
// scenario: the client connection drops mid-block. The request ctx is
// cancelled, the child is killed, the 422 is recorded, and the retry under
// the same key gets that 422 — the block never runs a second time.
func TestIdempotency_CancelledFirstRequestReplaysItsFailure(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	firstCh := serve(keyedRequest(t, "key-x", msgs).WithContext(ctx))
	require.Eventually(t, func() bool {
		srv.replays.mu.Lock()
		defer srv.replays.mu.Unlock()
		return len(srv.replays.entries) == 1 && runs() == 1
	}, 2*time.Second, 5*time.Millisecond, "first request must be executing")

	replayCh := serve(keyedRequest(t, "key-x", msgs))
	select {
	case <-replayCh:
		t.Fatal("replay must wait for the in-flight execution")
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	first := <-firstCh
	replay := <-replayCh
	require.Equal(t, http.StatusUnprocessableEntity, first.Code, first.Body.String())
	require.Equal(t, http.StatusUnprocessableEntity, replay.Code)
	require.Equal(t, first.Body.Bytes(), replay.Body.Bytes())
	require.Equal(t, 1, runs(), "the retry must not re-execute the block")
}

// TestIdempotency_WaiterCancelledGetsAnError: a replay that gives up while
// the first request is still running gets an explicit 503, not an empty 200.
func TestIdempotency_WaiterCancelledGetsAnError(t *testing.T) {
	srv := newTestServer()
	gate := filepath.Join(t.TempDir(), "go")
	block, _ := countingBlock(t, `while [ ! -f "`+gate+`" ]; do sleep 0.02; done; `+okContract)
	msgs := []goopenai.ChatCompletionMessage{block}

	handler := srv.Handler()
	firstReq := keyedRequest(t, "key-w", msgs)
	firstCh := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, firstReq)
		firstCh <- rec
	}()
	require.Eventually(t, func() bool {
		srv.replays.mu.Lock()
		defer srv.replays.mu.Unlock()
		return len(srv.replays.entries) == 1
	}, 2*time.Second, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, keyedRequest(t, "key-w", msgs).WithContext(ctx))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "server_error")

	require.NoError(t, os.WriteFile(gate, nil, 0o600))
	require.Equal(t, http.StatusOK, (<-firstCh).Code)
}

// TestIdempotency_PanicCompletesTheEntry: a panic inside the completion must
// not leave the key in flight forever. The replay gets a 500 promptly, and the
// completion ran only once.
func TestIdempotency_PanicCompletesTheEntry(t *testing.T) {
	srv := newTestServer()
	var calls atomic.Int32
	serve := srv.chat
	srv.chat = func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			panic("boom")
		}
		serve(w, r)
	}
	// A real server so net/http recovers the panic the way production does.
	ts := httptest.NewServer(srv.Handler())
	ts.Config.ErrorLog = slog.NewLogLogger(slog.NewTextHandler(io.Discard, nil), slog.LevelError)
	t.Cleanup(ts.Close)
	msgs := []goopenai.ChatCompletionMessage{bashBody(okContract)}
	client := &http.Client{Timeout: 2 * time.Second}

	req := keyedRequest(t, "key-p", msgs)
	post := func() (*http.Response, error) {
		body, err := json.Marshal(goopenai.ChatCompletionRequest{Model: "codellm", Messages: msgs})
		require.NoError(t, err)
		r, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+req.URL.Path, bytes.NewReader(body))
		require.NoError(t, err)
		r.Header.Set(idempotencyKeyHeader, "key-p")
		return client.Do(r)
	}

	resp, err := post()
	if err == nil {
		resp.Body.Close()
	}
	replay, err := post()
	require.NoError(t, err, "replay must not hang on the never-completed entry")
	defer replay.Body.Close()
	require.Equal(t, http.StatusInternalServerError, replay.StatusCode)
	require.Equal(t, int32(1), calls.Load(), "replay must not run the completion again")
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
