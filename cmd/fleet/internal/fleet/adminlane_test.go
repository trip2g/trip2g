package fleet

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/fleet/internal/fleet/trip2ggql"
)

// TestAdminGQL_RawGraphQLWithBearer asserts the admin lane POSTs the raw
// GraphQL {query,variables} to /_system/graphql carrying the owner's personal
// token as a Bearer (NOT an MCP/JSON-RPC envelope, no X-Api-Key, no cookie),
// and decodes the typed response.
func TestAdminGQL_RawGraphQLWithBearer(t *testing.T) {
	var gotPath, gotAuth, gotKey, gotCookie, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("X-Api-Key")
		gotCookie = r.Header.Get("Cookie")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"data":{"admin":{"allChangeWebhooks":{"nodes":[]}}}}`))
	}))
	defer srv.Close()

	gql := NewAdminGraphQLClient(srv.URL, "t2g_owner", srv.Client())
	resp, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.NoError(t, err)
	require.Empty(t, resp.Admin.AllChangeWebhooks.Nodes)

	require.Equal(t, "/_system/graphql", gotPath)
	require.Equal(t, "Bearer t2g_owner", gotAuth, "admin lane must carry the personal token")
	require.Empty(t, gotKey, "admin lane must not send the legacy X-Api-Key")
	require.Empty(t, gotCookie, "the fleet holds no session; there is no cookie to send")
	require.Contains(t, gotBody, `"operationName":"ListChangeWebhooks"`)
	require.Contains(t, gotBody, "allChangeWebhooks")
	require.NotContains(t, gotBody, "jsonrpc")
}

// TestAdminGQL_GraphQLErrorSurfaces asserts a raw GraphQL error envelope from
// /_system/graphql surfaces as a Go error carrying the message.
func TestAdminGQL_GraphQLErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"Cannot query field allChangeWebhooks"}]}`))
	}))
	defer srv.Close()

	gql := NewAdminGraphQLClient(srv.URL, "t2g_owner", srv.Client())
	_, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Cannot query field allChangeWebhooks")
}

// TestAdminGQL_UnauthorizedIsNotRetried asserts a 401 fails the call outright.
// The fleet has one credential and no way to mint another, so a 401 means the
// token was revoked — retrying would only hammer trip2g and bury the reason.
func TestAdminGQL_UnauthorizedIsNotRetried(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	gql := NewAdminGraphQLClient(srv.URL, "t2g_revoked", srv.Client())
	_, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.Error(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "a revoked token must not be retried")
}

// TestScopedGQL_CarriesTheDeliveryToken asserts the scoped lane is the same
// doer under a different credential: the per-delivery token from the payload.
func TestScopedGQL_CarriesTheDeliveryToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":{"admin":{"allChangeWebhooks":{"nodes":[]}}}}`))
	}))
	defer srv.Close()

	gql := NewScopedGraphQLClient(srv.URL, "delivery-token", srv.Client())
	_, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.NoError(t, err)
	require.Equal(t, "Bearer delivery-token", gotAuth)
}
