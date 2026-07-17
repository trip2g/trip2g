package fleet

import (
	"bytes"
	"io"
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

// NewAdminGraphQLClient builds the genqlient client for the trip2g admin lane:
// the typed operations in cmd/fleet/internal/fleet/trip2ggql POSTed as raw GraphQL to
// <baseURL>/_system/graphql, authenticated by a HAT-minted admin session
// cookie. The cookie is minted on first use and refreshed on a 401 (re-auth +
// one retry), matching the old hand-written GraphQLAdmin lane.
func NewAdminGraphQLClient(baseURL, jwtSecret, adminEmail string, hc *http.Client) graphql.Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	doer := &adminDoer{
		hat: newHATAuthenticator(baseURL, jwtSecret, adminEmail, hc),
		hc:  hc,
	}
	return graphql.NewClient(baseURL+"/_system/graphql", doer)
}

// adminDoer is the genqlient graphql.Doer for the admin lane. It attaches the
// HAT admin session cookie to each request, minting it on first use and
// re-authenticating once on a 401.
type adminDoer struct {
	hat *hatAuthenticator
	hc  *http.Client
}

// Do mints the admin cookie on first use, sends the request with the cached
// cookie(s), and on a 401 re-runs the HAT exchange and retries once.
func (d *adminDoer) Do(req *http.Request) (*http.Response, error) {
	body, err := drainBody(req)
	if err != nil {
		return nil, err
	}
	if len(d.hat.cached()) == 0 {
		if aerr := d.hat.authenticate(req.Context()); aerr != nil {
			return nil, aerr
		}
	}
	resp, err := d.send(req, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		if aerr := d.hat.authenticate(req.Context()); aerr != nil {
			return nil, aerr
		}
		return d.send(req, body)
	}
	return resp, nil
}

// send clones the original request with the buffered body and the current admin
// cookie(s), so a 401 retry replays cleanly with a fresh cookie.
func (d *adminDoer) send(orig *http.Request, body []byte) (*http.Response, error) {
	req := orig.Clone(orig.Context())
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}
	for _, ck := range d.hat.cached() {
		req.AddCookie(ck)
	}
	return d.hc.Do(req)
}

// drainBody buffers req.Body so the request can be replayed on a 401 retry.
func drainBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	return body, nil
}

// NewScopedGraphQLClient builds the genqlient client for the fleet's scoped
// lane: the per-delivery Bearer token is injected into every request by
// scopedDoer, so trip2g enforces the delivery's read/write scope.
func NewScopedGraphQLClient(baseURL, token string, hc *http.Client) graphql.Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	doer := &scopedDoer{token: token, hc: hc}
	return graphql.NewClient(baseURL+"/_system/graphql", doer)
}

// scopedDoer is the genqlient graphql.Doer for the scoped lane. It attaches
// the per-delivery Bearer token to each request, mirroring adminDoer exactly
// except authentication is a static token rather than a HAT-minted cookie.
type scopedDoer struct {
	token string
	hc    *http.Client
}

// Do clones the request, injects the Bearer Authorization header, and sends it.
func (d *scopedDoer) Do(req *http.Request) (*http.Response, error) {
	body, err := drainBody(req)
	if err != nil {
		return nil, err
	}
	r := req.Clone(req.Context())
	if body != nil {
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	}
	r.Header.Set("Authorization", "Bearer "+d.token)
	return d.hc.Do(r)
}
