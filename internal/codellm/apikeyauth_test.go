package codellm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"

	"trip2g/internal/coderun"
	"trip2g/internal/delegatedadmin"
)

const testAPIKey = "s3cr3t-api-key"

// twoLaneServer builds a codellm server with BOTH lanes wired: the OpenAI-standard
// api_key check (APIKeyCheck over the given configured key) and the browser
// delegated-admin cookie gate pointed at monolithURL. This mirrors production
// main.go.
func twoLaneServer(t *testing.T, configuredKey, monolithURL string) *Server {
	t.Helper()
	mw, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: monolithURL})
	require.NoError(t, err)
	tokenCheck := APIKeyCheck(configuredKey)
	return New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         coderun.SandboxPolicy{Mode: coderun.SandboxOff},
		Auth:            BrowserAuth(mw.Wrap, tokenCheck),
		TokenCheck:      tokenCheck,
	})
}

// chatReq builds a /v1/chat/completions request that echoes an empty result,
// optionally carrying a bearer api_key and/or a session cookie.
func chatReq(bearer, cookie string) *http.Request {
	payload := goopenai.ChatCompletionRequest{
		Model:    "codellm",
		Messages: []goopenai.ChatCompletionMessage{bashBody(`echo '{"changes":[],"answer":"ok"}'`)},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	return req
}

// TestAPIKeyAuth_CorrectKey_NoCookie_200 — a caller presenting the correct
// api_key is served WITHOUT any admin cookie (any OpenAI-standard client).
func TestAPIKeyAuth_CorrectKey_NoCookie_200(t *testing.T) {
	srv := twoLaneServer(t, testAPIKey, "http://127.0.0.1:1") // monolith never contacted
	rec := serve(srv, chatReq(testAPIKey, ""))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// TestAPIKeyAuth_WrongKey_NoCookie_401 — a wrong api_key falls through to the
// cookie gate; with no cookie the request is rejected (no bypass).
func TestAPIKeyAuth_WrongKey_NoCookie_401(t *testing.T) {
	srv := twoLaneServer(t, testAPIKey, "http://127.0.0.1:1")
	rec := serve(srv, chatReq("wrong-key", ""))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestBrowserLane_NoKey_AdminCookie_200 — with an api_key configured, a browser
// request carrying no key but a valid admin cookie still works: the key check
// fails, the request falls through to the delegated-admin gate.
func TestBrowserLane_NoKey_AdminCookie_200(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN")
	srv := twoLaneServer(t, testAPIKey, mono.URL)
	rec := serve(srv, chatReq("", sessionCookie))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// TestAPIKeyUnset_KeyHeader_Ignored_401 — with NO api_key configured, any bearer
// value is ignored (key auth is disabled) and only the cookie path applies; with
// no cookie the request is rejected. This is the fail-safe: an empty configured
// key must never authenticate a caller.
func TestAPIKeyUnset_KeyHeader_Ignored_401(t *testing.T) {
	srv := twoLaneServer(t, "", "http://127.0.0.1:1")
	rec := serve(srv, chatReq("anything", ""))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Unit-level APIKeyCheck matrix ---

func TestAPIKeyCheck_EmptyConfigured_NeverMatches(t *testing.T) {
	check := APIKeyCheck("")
	// Empty configured key: no header, empty header, and any non-empty header
	// must ALL fail — an empty key can never authenticate (auth-bypass guard).
	cases := []string{"", "Bearer ", "Bearer x", "Bearer "}
	for _, auth := range cases {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		require.Error(t, check(req), "empty configured key must never match (auth=%q)", auth)
	}
}

func TestAPIKeyCheck_CorrectKey_Passes(t *testing.T) {
	check := APIKeyCheck(testAPIKey)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	require.NoError(t, check(req))
}

func TestAPIKeyCheck_WrongOrMissing_Fails(t *testing.T) {
	check := APIKeyCheck(testAPIKey)
	for _, auth := range []string{"", "Bearer ", "Bearer wrong", "Basic " + testAPIKey, testAPIKey} {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		require.Error(t, check(req), "auth=%q must fail", auth)
	}
}

func TestAPIKeyCheck_BearerCaseInsensitive(t *testing.T) {
	check := APIKeyCheck(testAPIKey)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "bearer "+testAPIKey)
	require.NoError(t, check(req), "the Bearer scheme is case-insensitive per RFC 7235")
}
