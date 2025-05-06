package graphqlsse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/valyala/fasthttp"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"trip2g/internal/appreq"
)

// Transport is drop‑in compatible with gqlgen.Transport.
type Transport struct {
	KeepAlivePingInterval time.Duration
}

// -------------------- interface glue --------------------

// Supports keeps the same contract as the std‑lib version.
func (t Transport) Supports(r *http.Request) bool {
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		return false
	}
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && r.Method == http.MethodPost && mt == "application/json"
}

// Do selects the optimal path: real fasthttp when available,
// otherwise falls back to the original SSE transport.
func (t Transport) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	ctx := r.Context()

	if req, err := appreq.FromCtx(ctx); err == nil { // we are inside FastHTTP
		t.doFast(ctx, req.Req, w, exec)

		return
	}

	panic("SSEFastHTTP: not running inside fasthttp")
}

// -------------------- fastHTTP branch --------------------

func (t Transport) doFast(parent context.Context, fc *fasthttp.RequestCtx, ww http.ResponseWriter, exec graphql.GraphExecutor) {
	// 1. pre‑flight validation -------------------------------------------------
	if !fc.IsPost() ||
		!strings.Contains(string(fc.Request.Header.Peek("Accept")), "text/event-stream") {
		fc.Error("SSE requires POST and text/event-stream", fasthttp.StatusBadRequest)
		return
	}
	if mt, _, _ := mime.ParseMediaType(string(fc.Request.Header.ContentType())); mt != "application/json" {
		fc.Error("Content‑Type must be application/json", fasthttp.StatusUnsupportedMediaType)
		return
	}

	// 2. build parameters ------------------------------------------------------
	params := &graphql.RawParams{
		Headers: fastHeaderToHTTP(fc),
		ReadTime: graphql.TraceTiming{
			Start: graphql.Now(),
			End:   graphql.Now(),
		},
	}

	if err := json.Unmarshal(fc.PostBody(), params); err != nil {
		writeGraphQLError(fc, exec, parent, fmt.Errorf("decode body: %w", err))
		return
	}

	rc, errs := exec.CreateOperationContext(parent, params)
	ctx := graphql.WithOperationContext(parent, rc)

	// 3. response headers ------------------------------------------------------
	fc.SetStatusCode(fasthttp.StatusOK)
	ww.Header().Set("Content-Type", "text/event-stream")
	ww.Header().Set("Cache-Control", "no-cache")
	ww.Header().Set("Connection", "keep-alive")

	// 4. stream writer ---------------------------------------------------------
	fc.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send prelude so the browser starts the stream.
		fmt.Fprint(w, ":\n\n")
		_ = w.Flush()

		// Keep‑alive ticker
		var ticker *time.Ticker
		if t.KeepAlivePingInterval > 0 {
			ticker = time.NewTicker(t.KeepAlivePingInterval)
			defer ticker.Stop()
		}

		// helper to push SSE
		send := func(resp *graphql.Response) {
			if resp == nil {
				return
			}
			b, _ := json.Marshal(resp) // never fails for gqlgen
			w.WriteString(fmt.Sprintf("event: data\ndata: %s\n\n", b))
			_ = w.Flush()
		}

		// 5. main loop ---------------------------------------------------------
		if errs != nil {
			send(exec.DispatchError(ctx, errs))
		} else {
			responses, streamCtx := exec.DispatchOperation(ctx, rc)
			for {
				select {
				case <-streamCtx.Done():
					goto done
				default:
					send(responses(streamCtx))
					if ticker != nil {
						select {
						case <-ticker.C:
							fmt.Fprint(w, ": ping\n\n")
							_ = w.Flush()
						default:
						}
					}
				}
			}
		}
	done:
		fmt.Fprint(w, "event: complete\n\n")
		_ = w.Flush()
	})
}

// -------------------- helpers ------------------------------------------------

func fastHeaderToHTTP(h *fasthttp.RequestCtx) http.Header {
	out := make(http.Header, h.Request.Header.Len())
	h.Request.Header.VisitAll(func(k, v []byte) {
		out.Add(string(k), string(v))
	})
	return out
}

func writeGraphQLError(fc *fasthttp.RequestCtx, exec graphql.GraphExecutor, parent context.Context, err error) {
	gqlErr := gqlerror.Errorf(err.Error())
	resp := exec.DispatchError(parent, gqlerror.List{gqlErr})
	b, _ := json.Marshal(resp)
	fc.SetStatusCode(fasthttp.StatusBadRequest)
	fc.SetBody(b)
}
