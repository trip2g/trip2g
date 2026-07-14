package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet"
)

// authFakeSource is a static fleetgql.RoleSource for the auth integration test.
type authFakeSource struct{ roles []fleet.Role }

func (s authFakeSource) DiscoverParsed(context.Context) ([]fleet.Role, []error) {
	return s.roles, nil
}

// fakeMonolith stands in for the trip2g monolith's POST /_system/graphql,
// answering viewer{role} with the given role (mirrors #229's middleware tests).
func fakeMonolith(t *testing.T, role string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/_system/graphql", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"role":"` + role + `"}}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func graphqlPOST(t *testing.T, h http.Handler, path, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"query":"{ roles { name } }"}`
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestGraphQLServerRejectsUnauthenticated is the gate the #229 review demanded:
// an unauthenticated request to the fleet GraphQL port returns 401 (the
// delegated-admin middleware fails closed with no session cookie).
func TestGraphQLServerRejectsUnauthenticated(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN") // never reached: no cookie short-circuits
	h, err := newFleetGraphQLHandler(authFakeSource{}, mono.URL)
	require.NoError(t, err)

	rec := graphqlPOST(t, h, "/graphql", "")
	require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
}

// TestGraphQLServerAllowsAdmin: an admin-cookie request passes the gate and the
// GraphQL query executes (the monolith's viewer{role} answers ADMIN).
func TestGraphQLServerAllowsAdmin(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN")
	h, err := newFleetGraphQLHandler(
		authFakeSource{roles: []fleet.Role{{NotePath: "roles/indexer.md"}}}, mono.URL)
	require.NoError(t, err)

	rec := graphqlPOST(t, h, "/graphql", "trip2g_token=abc.def.ghi")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "indexer")
}

// TestGraphQLServerRejectsNonAdmin: a valid session cookie for a non-admin
// viewer is still rejected with 401.
func TestGraphQLServerRejectsNonAdmin(t *testing.T) {
	for _, role := range []string{"USER", "GUEST"} {
		mono := fakeMonolith(t, role)
		h, err := newFleetGraphQLHandler(authFakeSource{}, mono.URL)
		require.NoError(t, err)

		rec := graphqlPOST(t, h, "/graphql", "trip2g_token=abc")
		require.Equal(t, http.StatusUnauthorized, rec.Code, "role %s", role)
	}
}

// TestGraphQLServerGatesEntireMux proves the wrap covers the whole port, not
// only /graphql: any other browser path on this server is 401 without admin.
func TestGraphQLServerGatesEntireMux(t *testing.T) {
	mono := fakeMonolith(t, "ADMIN")
	h, err := newFleetGraphQLHandler(authFakeSource{}, mono.URL)
	require.NoError(t, err)

	rec := graphqlPOST(t, h, "/anything-else", "")
	require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
}

// TestWebhookDeliveryNotCookieGated documents the auth split. The webhook
// delivery path (monolith -> fleet, HMAC-authed) runs on a SEPARATE server
// (cli.cfg.ListenAddr, /deliver/) that main builds WITHOUT the delegated-admin
// wrapper — see run(). This mirrors that delivery mux with a stub handler and
// asserts a cookieless request reaches the handler (i.e. is NOT rejected by the
// admin-cookie gate), unlike the GraphQL mux above.
func TestWebhookDeliveryNotCookieGated(t *testing.T) {
	var reached bool
	deliveryMux := http.NewServeMux()
	deliveryMux.HandleFunc("/deliver/", func(w http.ResponseWriter, _ *http.Request) {
		reached = true // real fleet verifies HMAC here (f.ServeDelivery)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/deliver/somekey", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	deliveryMux.ServeHTTP(rec, req)

	require.True(t, reached, "delivery handler must run without an admin cookie")
	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
}
