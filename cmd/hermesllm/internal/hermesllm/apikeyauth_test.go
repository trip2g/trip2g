package hermesllm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const testAPIKey = "an-api-key-that-is-long-enough!!"

// serveGated runs a request carrying the given Authorization header through the
// api-key middleware in front of a 200 handler.
func serveGated(configuredKey, authHeader string) *httptest.ResponseRecorder {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	APIKeyAuth(configuredKey)(next).ServeHTTP(rec, req)
	return rec
}

func TestAPIKeyAuth(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		header     string
		want       int
	}{
		{name: "correct key passes", configured: testAPIKey, header: "Bearer " + testAPIKey, want: http.StatusOK},
		{name: "bearer scheme is case-insensitive", configured: testAPIKey, header: "bearer " + testAPIKey, want: http.StatusOK},
		{name: "wrong key is rejected", configured: testAPIKey, header: "Bearer wrong", want: http.StatusUnauthorized},
		{name: "missing header is rejected", configured: testAPIKey, want: http.StatusUnauthorized},
		{name: "empty bearer is rejected", configured: testAPIKey, header: "Bearer ", want: http.StatusUnauthorized},
		{name: "non-bearer scheme is rejected", configured: testAPIKey, header: "Basic " + testAPIKey, want: http.StatusUnauthorized},
		{name: "raw key without scheme is rejected", configured: testAPIKey, header: testAPIKey, want: http.StatusUnauthorized},
		// No second auth lane here (unlike codellm): an empty configured key means
		// the operator turned key auth off, so every request is admitted.
		{name: "empty configured key disables the check", configured: "", want: http.StatusOK},
		{name: "empty configured key ignores any header", configured: "", header: "Bearer anything", want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, serveGated(tt.configured, tt.header).Code)
		})
	}
}

func TestChatCompletions_KeyGated(t *testing.T) {
	upstream := fakeHermes(t, http.StatusOK, hermesReply("done"))
	srv := newServer(t, upstream.URL, testAPIKey)

	require.Equal(t, http.StatusOK, chatRequest(srv, testAPIKey, agentRequest()).Code)
	require.Equal(t, http.StatusUnauthorized, chatRequest(srv, "wrong-key", agentRequest()).Code)
	require.Equal(t, http.StatusUnauthorized, chatRequest(srv, "", agentRequest()).Code)
}
