// Package fastgql provides a fasthttp-to-net/http bridge for SSE streaming.
package fastgql

import (
	"bufio"
	"context"
	"net/http"
	"sync"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// NewSSEHandler wraps a net/http handler for SSE streaming over fasthttp.
//
// Unlike fasthttpadaptor.NewFastHTTPHandler, this function does not use sync.Pool
// for the response writer. This avoids a data race in fasthttpadaptor where a pooled
// writer is returned to the pool (via releaseNetHTTPResponseWriter) when the fasthttp
// streaming callback encounters a write error, while the SSE handler goroutine
// is still actively writing to the same writer.
//
// SSE response headers (Content-Type, Cache-Control, Connection) are pre-set on the
// fasthttp response before streaming starts. The net/http handler runs directly
// inside fasthttp's SetBodyStreamWriter callback, writing to the connection via
// a bufio.Writer. When a write or flush fails (client disconnect), the request
// context is cancelled, signaling the handler to stop.
func NewSSEHandler(h http.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var r http.Request

		err := fasthttpadaptor.ConvertRequest(ctx, &r, true)
		if err != nil {
			ctx.Logger().Printf("cannot parse requestURI %q: %v", r.RequestURI, err)
			ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
			return
		}

		ctx.Response.Header.Set("Content-Type", "text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ctx.SetBodyStreamWriter(func(bw *bufio.Writer) {
			reqCtx, cancel := context.WithCancel(r.Context())
			defer cancel()

			w := &sseResponseWriter{
				header: make(http.Header),
				bw:     bw,
				cancel: cancel,
			}

			h.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}

// sseResponseWriter implements http.ResponseWriter and http.Flusher for SSE
// streaming. It writes directly to a bufio.Writer connected to the fasthttp
// connection. On write or flush errors (client disconnect), it cancels the
// request context to signal the handler to stop.
//
// A mutex protects the bufio.Writer because gqlgen's SSE transport may write
// from multiple goroutines (main loop + keepalive pinger).
type sseResponseWriter struct {
	header http.Header
	mu     sync.Mutex
	bw     *bufio.Writer
	cancel context.CancelFunc
}

var _ http.ResponseWriter = (*sseResponseWriter)(nil)
var _ http.Flusher = (*sseResponseWriter)(nil)

func (w *sseResponseWriter) Header() http.Header {
	return w.header
}

// WriteHeader is a no-op: SSE headers are pre-set on the fasthttp response
// before the stream writer starts.
func (w *sseResponseWriter) WriteHeader(_ int) {}

func (w *sseResponseWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.bw.Write(p)
	if err != nil {
		w.cancel()
	}

	return n, err
}

func (w *sseResponseWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	err := w.bw.Flush()
	if err != nil {
		w.cancel()
	}
}
