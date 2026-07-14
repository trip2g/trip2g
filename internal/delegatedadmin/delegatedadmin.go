// Package delegatedadmin provides an HTTP middleware that gates a request on the
// caller being a trip2g admin, by delegating the decision to the monolith.
//
// It is the shared auth piece for browser-facing services that live behind a
// same-origin Caddy proxy (fleet's GraphQL, codellm). Each of those services
// binds to a private interface; Caddy forwards the browser request — including
// the trip2g session cookie — verbatim. This middleware then, per request,
// re-sends THAT cookie to the monolith's `POST /_system/graphql` with
// `query { viewer { role } }` and allows only when the answer is `ADMIN`.
//
// Design guarantees:
//   - Full parity with the monolith admin check (the ban validator and every
//     other validator run in the monolith, not a copy here).
//   - The service holds no secret and verifies nothing itself — it asks the
//     authority.
//   - It forwards the END-USER's cookie, never a service token/HAT. Using a
//     service credential would answer "is the service admin", not "is the
//     caller admin".
//   - Fail-closed: monolith unreachable / timeout / non-200 / GraphQL error →
//     reject. Never allow on uncertainty.
package delegatedadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// DefaultCookieName is the trip2g session cookie name (see usertoken.DefaultConfig).
const DefaultCookieName = "trip2g_token"

// systemGraphQLPath is the monolith GraphQL endpoint (/graphql is deprecated).
const systemGraphQLPath = "/_system/graphql"

// viewerRoleQuery asks the monolith who the caller is.
const viewerRoleQuery = `query { viewer { role } }`

// adminRole is the value the GraphQL Role enum serializes to for admins.
// The enum is uppercase (GUEST/USER/ADMIN), not the lowercase JWT claim.
const adminRole = "ADMIN"

// defaultTimeout bounds the delegated round-trip so a slow/hung monolith
// fails closed instead of holding the caller open.
const defaultTimeout = 5 * time.Second

// Config configures the middleware.
type Config struct {
	// MonolithBaseURL is where viewer{role} is asked, e.g. "http://localhost:8081".
	// Required. No trailing slash needed.
	MonolithBaseURL string

	// HTTPClient performs the delegated request. Optional; a client with a sane
	// timeout is used when nil.
	HTTPClient *http.Client

	// CookieName is the session cookie whose presence is required before the
	// delegated call is made. Optional; defaults to DefaultCookieName.
	CookieName string
}

// Middleware gates requests on delegated admin authentication.
type Middleware struct {
	baseURL    string
	client     *http.Client
	cookieName string
}

// New builds a Middleware. MonolithBaseURL must be non-empty.
func New(cfg Config) (*Middleware, error) {
	if cfg.MonolithBaseURL == "" {
		return nil, errors.New("delegatedadmin: MonolithBaseURL is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	cookieName := cfg.CookieName
	if cookieName == "" {
		cookieName = DefaultCookieName
	}
	return &Middleware{
		baseURL:    cfg.MonolithBaseURL,
		client:     client,
		cookieName: cookieName,
	}, nil
}

// Wrap returns a handler that only calls next when the caller is a verified
// admin; otherwise it writes 401 and does not call next.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.isAdmin(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type viewerRoleResponse struct {
	Data struct {
		Viewer struct {
			Role string `json:"role"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// isAdmin performs the delegated check. Any uncertainty returns false
// (fail-closed).
func (m *Middleware) isAdmin(r *http.Request) bool {
	// Fast-fail with no round-trip when the session cookie is absent.
	if _, err := r.Cookie(m.cookieName); err != nil {
		return false
	}

	role, err := m.fetchViewerRole(r.Context(), r.Header.Get("Cookie"))
	if err != nil {
		return false
	}
	return role == adminRole
}

// fetchViewerRole forwards the caller's raw Cookie header to the monolith and
// returns the viewer role. Any transport/status/decode/GraphQL error is
// returned so the caller can fail closed.
func (m *Middleware) fetchViewerRole(ctx context.Context, cookieHeader string) (string, error) {
	body, err := json.Marshal(map[string]string{"query": viewerRoleQuery})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+systemGraphQLPath, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// Forward the end-user's cookie verbatim — this is the whole point.
	req.Header.Set("Cookie", cookieHeader)

	// net/http (not fasthttp) deliberately: this propagates the inbound
	// request's ctx, so a slow/hung monolith fails closed via context
	// cancellation within the caller's own request lifetime. fasthttp's client
	// has no context param (only DoTimeout/DoDeadline), which would need a
	// manually threaded deadline to get the same guarantee. Low-QPS auth gate:
	// cancellation-correctness beats throughput here.
	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("delegatedadmin: monolith returned status %d", resp.StatusCode)
	}

	var parsed viewerRoleResponse
	if err = json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Errors) > 0 {
		return "", fmt.Errorf("delegatedadmin: graphql error: %s", parsed.Errors[0].Message)
	}
	return parsed.Data.Viewer.Role, nil
}
