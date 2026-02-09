package fastgql_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"trip2g/internal/fastgql"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// mockSSEHandler simulates gqlgen's SSE transport: sets headers, writes
// an initial comment, flushes, then streams events until done or context cancelled.
func mockSSEHandler(eventCount int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		fmt.Fprint(w, ":\n\n")
		flusher.Flush()

		for i := range eventCount {
			select {
			case <-r.Context().Done():
				return
			default:
				fmt.Fprintf(w, "event: next\ndata: {\"n\":%d}\n\n", i)
				flusher.Flush()
			}
		}

		fmt.Fprint(w, "event: complete\n\n")
		flusher.Flush()
	})
}

func startTestServer(t *testing.T, handler fasthttp.RequestHandler) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &fasthttp.Server{Handler: handler}

	go func() {
		_ = server.Serve(ln)
	}()

	t.Cleanup(func() {
		_ = server.Shutdown()
		ln.Close()
	})

	return ln
}

func TestSSEHandler_StreamsEvents(t *testing.T) {
	handler := fastgql.NewSSEHandler(mockSSEHandler(3))
	ln := startTestServer(t, handler)

	req, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/graphql",
		strings.NewReader(`{"query":"subscription { test }"}`))
	require.NoError(t, err)

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	require.Contains(t, bodyStr, ":\n\n", "should contain initial SSE comment")
	require.Contains(t, bodyStr, "event: next\ndata: {\"n\":0}", "should contain first event")
	require.Contains(t, bodyStr, "event: next\ndata: {\"n\":2}", "should contain last event")
	require.Contains(t, bodyStr, "event: complete", "should contain complete event")
}

func TestSSEHandler_ResponseHeaders(t *testing.T) {
	handler := fastgql.NewSSEHandler(mockSSEHandler(1))
	ln := startTestServer(t, handler)

	req, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/graphql",
		strings.NewReader(`{"query":"subscription { test }"}`))
	require.NoError(t, err)

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	require.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
}

// TestSSEHandler_CancelsContextOnDisconnect verifies that when the client
// disconnects, the handler's context is cancelled (via a failed write/flush),
// causing the handler to stop.
//
// The handler must be actively writing for disconnect detection to work:
// fasthttp detects a closed connection only when it tries to write data.
// This matches real gqlgen SSE behavior where events or keepalive pings
// are sent continuously.
func TestSSEHandler_CancelsContextOnDisconnect(t *testing.T) {
	ctxCancelled := make(chan struct{})

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)

		fmt.Fprint(w, ":\n\n")
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				close(ctxCancelled)
				return
			case <-time.After(5 * time.Millisecond):
				fmt.Fprint(w, "event: next\ndata: {}\n\n")
				flusher.Flush()
			}
		}
	})

	handler := fastgql.NewSSEHandler(h)
	ln := startTestServer(t, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+ln.Addr().String()+"/graphql",
		strings.NewReader(`{"query":"subscription { test }"}`))
	require.NoError(t, err)

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)

	select {
	case <-ctxCancelled:
		// Handler context was cancelled — success.
	case <-time.After(5 * time.Second):
		t.Fatal("handler context was not cancelled after client disconnect")
	}
}

func TestSSEHandler_ImplementsFlusher(t *testing.T) {
	flusherOK := make(chan bool, 1)

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, ok := w.(http.Flusher)
		flusherOK <- ok
	})

	handler := fastgql.NewSSEHandler(h)
	ln := startTestServer(t, handler)

	req, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String(),
		strings.NewReader(`{}`))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)

	select {
	case ok := <-flusherOK:
		require.True(t, ok, "sseResponseWriter must implement http.Flusher")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handler")
	}
}

// TestSSEHandler_NoRaceWithConcurrentRequests verifies that concurrent SSE
// connections and regular requests do not race on shared state.
//
// fasthttpadaptor.NewFastHTTPHandler uses sync.Pool for response writers.
// For SSE (long-lived connections), when the connection closes, the streaming
// callback releases the writer to the pool while the SSE goroutine is still
// writing — causing a data race. NewSSEHandler avoids this by not using
// sync.Pool. This test ensures no race is detected under -race.
func TestSSEHandler_NoRaceWithConcurrentRequests(t *testing.T) {
	regularHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	})

	boundedSSE := mockSSEHandler(20)

	sseH := fastgql.NewSSEHandler(boundedSSE)
	regularH := fasthttpadaptor.NewFastHTTPHandler(regularHandler)

	serverHandler := func(ctx *fasthttp.RequestCtx) {
		if strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream") {
			sseH(ctx)
		} else {
			regularH(ctx)
		}
	}

	ln := startTestServer(t, serverHandler)
	addr := "http://" + ln.Addr().String()

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr,
				strings.NewReader(`{"query":"subscription { test }"}`))
			if err != nil {
				return
			}

			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return
			}

			defer resp.Body.Close()

			_, _ = io.ReadAll(resp.Body)
		}()
	}

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr,
				strings.NewReader(`{"query":"{ __typename }"}`))
			if err != nil {
				return
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return
			}

			defer resp.Body.Close()

			_, _ = io.ReadAll(resp.Body)
		}()
	}

	wg.Wait()
}
