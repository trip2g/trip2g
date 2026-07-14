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

const testChannelToken = "s3cr3t-channel-token"

// twoLaneServer builds a codellm server with BOTH lanes wired: the fleet lane
// (ChannelTokenCheck over the given configured token) and the browser lane
// (delegated-admin gate pointed at monolithURL). This mirrors production main.go.
func twoLaneServer(t *testing.T, configuredToken, monolithURL string) *Server {
	t.Helper()
	mw, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: monolithURL})
	require.NoError(t, err)
	tokenCheck := ChannelTokenCheck(configuredToken)
	return New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         coderun.SandboxPolicy{Mode: coderun.SandboxOff},
		Auth:            BrowserAuth(mw.Wrap, tokenCheck),
		TokenCheck:      tokenCheck,
	})
}

// chatReq builds a /v1/chat/completions request that echoes an empty result,
// optionally carrying a bearer channel token and/or a session cookie.
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

// TestFleetLane_CorrectToken_NoCookie_200 — a server-to-server caller presenting
// the correct channel token is served WITHOUT any admin cookie (the fleet lane).
func TestFleetLane_CorrectToken_NoCookie_200(t *testing.T) {
	srv := twoLaneServer(t, testChannelToken, "http://127.0.0.1:1") // monolith never contacted
	rec := serve(srv, chatReq(testChannelToken, ""))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// TestFleetLane_WrongToken_NoCookie_401 — a wrong channel token falls through to
// the cookie gate; with no cookie the request is rejected (no bypass).
func TestFleetLane_WrongToken_NoCookie_401(t *testing.T) {
	srv := twoLaneServer(t, testChannelToken, "http://127.0.0.1:1")
	rec := serve(srv, chatReq("wrong-token", ""))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestBrowserLane_NoToken_AdminCookie_200 — with a channel token configured, a
// browser request carrying no token but a valid admin cookie still works: the
// fleet check fails, the request falls through to the delegated-admin gate.
func TestBrowserLane_NoToken_AdminCookie_200(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN")
	srv := twoLaneServer(t, testChannelToken, mono.URL)
	rec := serve(srv, chatReq("", sessionCookie))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// TestChannelUnset_TokenHeader_Ignored_401 — with NO channel token configured,
// any bearer token is ignored (the lane is disabled) and only the cookie path
// applies; with no cookie the request is rejected. This is the fail-safe: an
// empty configured token must never authenticate a caller.
func TestChannelUnset_TokenHeader_Ignored_401(t *testing.T) {
	srv := twoLaneServer(t, "", "http://127.0.0.1:1")
	rec := serve(srv, chatReq("anything", ""))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Unit-level ChannelTokenCheck matrix ---

func TestChannelTokenCheck_EmptyConfigured_NeverMatches(t *testing.T) {
	check := ChannelTokenCheck("")
	// Empty configured token: no header, empty header, and any non-empty header
	// must ALL fail — an empty token can never authenticate (auth-bypass guard).
	cases := []string{"", "Bearer ", "Bearer x", "Bearer "}
	for _, auth := range cases {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		require.Error(t, check(req), "empty configured token must never match (auth=%q)", auth)
	}
}

func TestChannelTokenCheck_CorrectToken_Passes(t *testing.T) {
	check := ChannelTokenCheck(testChannelToken)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+testChannelToken)
	require.NoError(t, check(req))
}

func TestChannelTokenCheck_WrongOrMissing_Fails(t *testing.T) {
	check := ChannelTokenCheck(testChannelToken)
	for _, auth := range []string{"", "Bearer ", "Bearer wrong", "Basic " + testChannelToken, testChannelToken} {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		require.Error(t, check(req), "auth=%q must fail", auth)
	}
}

func TestChannelTokenCheck_BearerCaseInsensitive(t *testing.T) {
	check := ChannelTokenCheck(testChannelToken)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "bearer "+testChannelToken)
	require.NoError(t, check(req), "the Bearer scheme is case-insensitive per RFC 7235")
}
