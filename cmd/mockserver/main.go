// Command mockserver is a generic jsonnet-driven HTTP mock server.
// It evaluates a jsonnet config file per request, passing the request as an
// ext-var and using the returned object to write the HTTP response.
//
// Flags / env:
//
//	--config <path>           : jsonnet config file (required)
//	--listen / MOCKSERVER_LISTEN : listen address (default ":9090")
//	--log-level               : log verbosity debug|info|warn|error (default "info")
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"trip2g/internal/jsonneteval"
	"trip2g/internal/logger"
	"trip2g/internal/zerologger"
)

const (
	defaultListen     = ":9090"
	defaultLogLevel   = "info"
	defaultCT         = "application/json"
	defaultStatus     = 200
	bodyLimit         = 2 * 1024 * 1024 // 2 MiB
	readHeaderTimeout = 10 * time.Second
)

// responseShape is the object the jsonnet config must return.
type responseShape struct {
	// Status is the HTTP status code; defaults to 200 when zero.
	Status int `json:"status"`
	// Headers are written verbatim; defaults to {"Content-Type":"application/json"} when nil.
	Headers map[string]string `json:"headers"`
	// Body is any JSON value serialised and written as the response body.
	Body json.RawMessage `json:"body"`
	// BodyText, when non-nil, is written verbatim and takes precedence over Body.
	BodyText *string `json:"bodyText"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mockserver:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configPath string
		listenAddr string
		logLevel   string
	)
	flag.StringVar(&configPath, "config", "", "path to jsonnet config file (required)")
	flag.StringVar(&listenAddr, "listen", envOr("MOCKSERVER_LISTEN", defaultListen), "listen address")
	flag.StringVar(&logLevel, "log-level", defaultLogLevel, "log level (debug|info|warn|error)")
	flag.Parse()

	if configPath == "" {
		return errors.New("--config is required")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	lg := zerologger.New(logLevel, false)

	mux := http.NewServeMux()
	mux.HandleFunc("/", newHandler(string(configBytes), lg))

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	lg.Info("mockserver: listening", "addr", listenAddr, "config", configPath)

	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// newHandler returns an http.HandlerFunc that evaluates configSrc per request.
// It is factored out of run so tests can drive it via httptest without a listener.
func newHandler(configSrc string, lg logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, bodyLimit))
		if err != nil {
			lg.Error("mockserver: read body", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		reqJSON, err := json.Marshal(buildReqObj(r, body))
		if err != nil {
			lg.Error("mockserver: marshal request", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		out, err := jsonneteval.EvalJSON(configSrc, map[string]string{"request": string(reqJSON)})
		if err != nil {
			lg.Error("mockserver: eval jsonnet", "error", err, "path", r.URL.Path)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		var resp responseShape
		if err = json.Unmarshal(out, &resp); err != nil {
			lg.Error("mockserver: decode response shape", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeResp(w, lg, resp)
	}
}

// buildReqObj constructs the request object passed to the jsonnet config as the
// "request" ext-var. Headers and query params are flattened to the first value.
func buildReqObj(r *http.Request, body []byte) map[string]any {
	headers := make(map[string]string, len(r.Header))
	for k, vals := range r.Header {
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}

	query := make(map[string]string)
	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			query[k] = vals[0]
		}
	}

	return map[string]any{
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   query,
		"headers": headers,
		"body":    string(body),
	}
}

// writeResp applies defaults and writes status, headers, and body.
func writeResp(w http.ResponseWriter, lg logger.Logger, resp responseShape) {
	status := defaultStatus
	if resp.Status != 0 {
		status = resp.Status
	}

	if len(resp.Headers) > 0 {
		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
	} else {
		w.Header().Set("Content-Type", defaultCT)
	}

	w.WriteHeader(status)

	if resp.BodyText != nil {
		if _, err := io.WriteString(w, *resp.BodyText); err != nil {
			lg.Error("mockserver: write body text", "error", err)
		}
		return
	}
	if len(resp.Body) > 0 {
		if _, err := w.Write(resp.Body); err != nil {
			lg.Error("mockserver: write body", "error", err)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
