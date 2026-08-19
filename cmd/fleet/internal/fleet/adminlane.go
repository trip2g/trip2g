package fleet

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

// NewAdminGraphQLClient builds the genqlient client for the trip2g admin lane:
// the typed operations in cmd/fleet/internal/fleet/trip2ggql POSTed as raw
// GraphQL to <baseURL>/_system/graphql, authenticated by the owner's personal
// token. That token resolves to an admin user, which is what trip2g's admin
// namespace gates on. A 401 here means it was revoked, and the fleet holds no
// second credential to fall back to.
func NewAdminGraphQLClient(baseURL, token string, hc *http.Client) graphql.Client {
	return newBearerClient(baseURL, token, hc)
}

// NewScopedGraphQLClient builds the genqlient client for the fleet's scoped
// lane: the per-delivery Bearer token, so trip2g enforces the delivery's
// read/write scope.
func NewScopedGraphQLClient(baseURL, token string, hc *http.Client) graphql.Client {
	return newBearerClient(baseURL, token, hc)
}

func newBearerClient(baseURL, token string, hc *http.Client) graphql.Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return graphql.NewClient(baseURL+"/_system/graphql", &bearerDoer{token: token, hc: hc})
}

// bearerDoer is the genqlient graphql.Doer both lanes use; they differ only in
// which token they carry.
type bearerDoer struct {
	token string
	hc    *http.Client
}

// Do clones the request so the caller's headers stay untouched, injects the
// Bearer Authorization header, and sends it.
func (d *bearerDoer) Do(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+d.token)
	return d.hc.Do(r)
}
