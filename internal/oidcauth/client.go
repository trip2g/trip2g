// ID-token verification lives in verify.go / jwks.go. The callback reads
// identity from UserInfo and uses verified ID-token email claims only as the
// presence-aware fallback documented in handleoidccallback.
package oidcauth

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"
)

const defaultScopes = "openid email profile"

// BuildAuthURL builds the OIDC authorization URL. The authorization endpoint
// comes from discovery, unlike Google's hardcoded constant.
func BuildAuthURL(authzEndpoint, clientID, redirectURI, state, scopes, nonce string) string {
	if scopes == "" {
		scopes = defaultScopes
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", state)
	if nonce != "" {
		params.Set("nonce", nonce)
	}

	return authzEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token.
func ExchangeCode(clientID, clientSecret, code, redirectURI, tokenEndpoint string) (*TokenResponse, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("code", code)
	formData.Set("grant_type", "authorization_code")
	formData.Set("redirect_uri", redirectURI)

	req.SetRequestURI(tokenEndpoint)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBodyString(formData.Encode())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("oidc token exchange failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	return &tokenResp, nil
}

// GetUserInfo fetches user information using the access token.
func GetUserInfo(accessToken, userInfoEndpoint string) (*UserInfo, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(userInfoEndpoint)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("oidc user info failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var userInfo UserInfo
	err = json.Unmarshal(resp.Body(), &userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &userInfo, nil
}
