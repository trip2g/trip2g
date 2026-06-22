package oidcauth

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// Discover fetches the OIDC discovery document from the issuer's
// well-known endpoint and parses it into Endpoints.
//
// TODO: cache discovery per-issuer with a TTL on the app layer; login is infrequent so an uncached fetch is acceptable for now.
func Discover(issuer string) (Endpoints, error) {
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(discoveryURL)
	req.Header.SetMethod(fasthttp.MethodGet)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed to fetch discovery document: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return Endpoints{}, fmt.Errorf("oidc discovery failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var endpoints Endpoints
	err = json.Unmarshal(resp.Body(), &endpoints)
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed to unmarshal discovery document: %w", err)
	}

	if endpoints.AuthorizationEndpoint == "" || endpoints.TokenEndpoint == "" {
		return Endpoints{}, fmt.Errorf(
			"oidc discovery missing required endpoints: authorization=%q, token=%q",
			endpoints.AuthorizationEndpoint, endpoints.TokenEndpoint,
		)
	}

	return endpoints, nil
}
