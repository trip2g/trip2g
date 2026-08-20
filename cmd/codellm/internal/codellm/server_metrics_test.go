package codellm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/codellmmetrics"
	"trip2g/cmd/codellm/internal/coderun"
)

// newMeteredTestServer is newTestServer with a live metrics sink attached.
func newMeteredTestServer() (*Server, *codellmmetrics.Metrics) {
	m := codellmmetrics.New()
	return New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         coderun.SandboxPolicy{Mode: coderun.SandboxOff},
		Metrics:         m,
	}), m
}

// scrape renders the registry the way Prometheus would read it.
func scrape(t *testing.T, m *codellmmetrics.Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

// TestMetrics_SuccessfulCompletion asserts a served completion lands in the
// request, block and change series.
func TestMetrics_SuccessfulCompletion(t *testing.T) {
	srv, m := newMeteredTestServer()
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{
		bashBody(`echo '{"changes":[{"path":"notes/a.md","content":"hi"}],"answer":"done"}'`),
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	out := scrape(t, m)
	require.Contains(t, out, `codellm_requests_total{endpoint="chat_completions",status="200"} 1`)
	require.Contains(t, out, `codellm_blocks_total{outcome="ok",program="bash"} 1`)
	require.Contains(t, out, `codellm_block_exit_codes_total{exit_code="0",program="bash"} 1`)
	require.Contains(t, out, `codellm_changes_total{kind="write"} 1`)
}

// TestMetrics_FailedBlockIsCountedByKind asserts a 422 is attributed to the
// coderun failure kind AND to the block's real exit code — the two signals the
// service's stability is read from.
func TestMetrics_FailedBlockIsCountedByKind(t *testing.T) {
	srv, m := newMeteredTestServer()
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(`echo boom >&2; exit 7`)})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	out := scrape(t, m)
	require.Contains(t, out, `codellm_requests_total{endpoint="chat_completions",status="422"} 1`)
	require.Contains(t, out, `codellm_exec_errors_total{kind="nonzero_exit"} 1`)
	require.Contains(t, out, `codellm_blocks_total{outcome="nonzero_exit",program="bash"} 1`)
	require.Contains(t, out, `codellm_block_exit_codes_total{exit_code="7",program="bash"} 1`)
}

// TestMetrics_AuthLanes asserts the api-key lane and a denied cookie request
// are counted separately: a rising denied rate on an endpoint that executes
// code is a probing signal.
func TestMetrics_AuthLanes(t *testing.T) {
	m := codellmmetrics.New()
	denyAll := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
	key := "codellm-test-key-0123456789abcdefghij"
	auth := BrowserAuthWithMetrics(denyAll, APIKeyCheck(key), m)
	served := auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	withKey := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	withKey.Header.Set("Authorization", "Bearer "+key)
	served.ServeHTTP(httptest.NewRecorder(), withKey)

	served.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))

	out := scrape(t, m)
	require.Contains(t, out, `codellm_auth_total{lane="apikey",result="allowed"} 1`)
	require.Contains(t, out, `codellm_auth_total{lane="cookie",result="denied"} 1`)
}
