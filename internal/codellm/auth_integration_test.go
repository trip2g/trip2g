package codellm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"

	"trip2g/internal/agentruntime"
	"trip2g/internal/delegatedadmin"
)

// fakeMonolith spins up a stand-in /_system/graphql that answers viewer{role}
// with the given role, so the delegated-admin gate has an authority to ask.
func fakeMonolith(t *testing.T, role string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/_system/graphql", r.URL.Path)
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"role":"` + role + `"}}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// gatedServer builds a codellm server whose browser-facing endpoints are gated
// by the delegated-admin middleware pointed at monolithURL (fleet TokenCheck nil,
// i.e. the secure default where every browser request needs an admin cookie).
func gatedServer(t *testing.T, monolithURL string) *Server {
	t.Helper()
	mw, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: monolithURL})
	require.NoError(t, err)
	return New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         agentruntime.SandboxPolicy{Mode: agentruntime.SandboxOff},
		Auth:            BrowserAuth(mw.Wrap, nil),
	})
}

const sessionCookie = "trip2g_token=abc.def.ghi"

// graphqlReq builds a parseMarkdown GraphQL request, optionally with the session
// cookie.
func graphqlReq(cookie string) *http.Request {
	body, _ := json.Marshal(map[string]string{"query": `{ parseMarkdown(md: "hello") { index kind } }`})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	return req
}

// debugChatReq builds an x_fleet_debug chat request, optionally with the cookie.
func debugChatReq(cookie string) *http.Request {
	payload := map[string]any{
		"model":         "codellm",
		"messages":      []goopenai.ChatCompletionMessage{bashBody(`echo '{"changes":[],"answer":""}'`)},
		"x_fleet_debug": true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	return req
}

func serve(srv *Server, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// TestBrowserGraphQL_NoCookie_401 — a browser request to the GraphQL endpoint
// without an admin cookie is rejected before it reaches the resolver.
func TestBrowserGraphQL_NoCookie_401(t *testing.T) {
	srv := gatedServer(t, "http://127.0.0.1:1") // monolith never contacted (no cookie)
	rec := serve(srv, graphqlReq(""))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestBrowserGraphQL_AdminCookie_Passes — with a mocked admin viewer{role} the
// GraphQL request is served.
func TestBrowserGraphQL_AdminCookie_Passes(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN")
	srv := gatedServer(t, mono.URL)
	rec := serve(srv, graphqlReq(sessionCookie))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var env struct {
		Data struct {
			ParseMarkdown []struct {
				Index int    `json:"index"`
				Kind  string `json:"kind"`
			} `json:"parseMarkdown"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Nil(t, env.Errors, "graphql errors: %s", env.Errors)
	require.NotEmpty(t, env.Data.ParseMarkdown)
}

// TestBrowserGraphQL_NonAdmin_401 — a logged-in non-admin is rejected.
func TestBrowserGraphQL_NonAdmin_401(t *testing.T) {
	mono := fakeMonolith(t, "USER")
	srv := gatedServer(t, mono.URL)
	rec := serve(srv, graphqlReq(sessionCookie))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestBrowserDebug_NoCookie_401 — the x_fleet_debug execution path is gated too.
func TestBrowserDebug_NoCookie_401(t *testing.T) {
	srv := gatedServer(t, "http://127.0.0.1:1")
	rec := serve(srv, debugChatReq(""))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestBrowserDebug_AdminCookie_Passes — with a mocked admin cookie the
// x_fleet_debug request runs and returns the debug field.
func TestBrowserDebug_AdminCookie_Passes(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN")
	srv := gatedServer(t, mono.URL)
	rec := serve(srv, debugChatReq(sessionCookie))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	require.Contains(t, rec.Body.String(), "x_fleet_debug")
}

// TestHealthz_Open — liveness is never gated (probes hit codellm directly).
func TestHealthz_Open(t *testing.T) {
	srv := gatedServer(t, "http://127.0.0.1:1")
	rec := serve(srv, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}
