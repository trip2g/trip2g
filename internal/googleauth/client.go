package googleauth

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	authURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL    = "https://oauth2.googleapis.com/token" //nolint:gosec // API endpoint, not credential
	userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
)

type Client struct {
	config     Config
	http       *fasthttp.Client
	reqTimeout time.Duration
}

func NewClient(config Config) *Client {
	return &Client{
		config:     config,
		reqTimeout: 10 * time.Second,
		http: &fasthttp.Client{
			NoDefaultUserAgentHeader: true,
		},
	}
}

// GoogleAuthURL returns the URL to redirect the user to for Google OAuth.
func (c *Client) GoogleAuthURL(redirectURI, state string) string {
	return BuildAuthURL(c.config.ClientID, redirectURI, state)
}

// BuildAuthURL builds the Google OAuth authorization URL.
func BuildAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "email profile")
	params.Set("state", state)
	params.Set("access_type", "online")

	return authURL + "?" + params.Encode()
}

// GoogleExchangeCode exchanges an authorization code for an access token.
func (c *Client) GoogleExchangeCode(code, redirectURI string) (*TokenResponse, error) {
	return ExchangeCode(c.config.ClientID, c.config.ClientSecret, code, redirectURI)
}

// ExchangeCode exchanges an authorization code for an access token.
func ExchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("code", code)
	formData.Set("grant_type", "authorization_code")
	formData.Set("redirect_uri", redirectURI)

	req.SetRequestURI(tokenURL)
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
		return nil, fmt.Errorf("google token exchange failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	return &tokenResp, nil
}

// GoogleGetUserInfo fetches user information using the access token.
func (c *Client) GoogleGetUserInfo(accessToken string) (*UserInfo, error) {
	return GetUserInfo(accessToken)
}

// GetUserInfo fetches user information using the access token.
func GetUserInfo(accessToken string) (*UserInfo, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(userInfoURL)
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
		return nil, fmt.Errorf("google user info failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var userInfo UserInfo
	err = json.Unmarshal(resp.Body(), &userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &userInfo, nil
}

// ErrorResponse represents an OAuth error response.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// ValidateCredentials validates OAuth credentials by making a test token exchange request.
// Returns nil if credentials are valid, error otherwise.
func ValidateCredentials(clientID, clientSecret, redirectURI string) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// Use a dummy code - we expect this to fail, but the error type tells us if credentials are valid
	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("code", "dummy_validation_code")
	formData.Set("grant_type", "authorization_code")
	formData.Set("redirect_uri", redirectURI)

	req.SetRequestURI(tokenURL)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBodyString(formData.Encode())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	// Parse error response
	var errResp ErrorResponse
	err = json.Unmarshal(resp.Body(), &errResp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// "invalid_client" means credentials are wrong
	// "invalid_grant" means credentials are valid but code is wrong (expected)
	if errResp.Error == "invalid_client" {
		return fmt.Errorf("invalid credentials: %s", errResp.ErrorDescription)
	}

	// Any other error (like invalid_grant) means credentials are valid
	return nil
}
