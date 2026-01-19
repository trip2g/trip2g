package githubauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	authURL      = "https://github.com/login/oauth/authorize"
	tokenURL     = "https://github.com/login/oauth/access_token" //nolint:gosec // API endpoint, not credential
	userURL      = "https://api.github.com/user"
	userEmailURL = "https://api.github.com/user/emails"
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

// GitHubAuthURL returns the URL to redirect the user to for GitHub OAuth.
func (c *Client) GitHubAuthURL(redirectURI, state string) string {
	return BuildAuthURL(c.config.ClientID, redirectURI, state)
}

// BuildAuthURL builds the GitHub OAuth authorization URL.
func BuildAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", "user:email")
	params.Set("state", state)

	return authURL + "?" + params.Encode()
}

// GitHubExchangeCode exchanges an authorization code for an access token.
func (c *Client) GitHubExchangeCode(code string) (*TokenResponse, error) {
	return ExchangeCode(c.config.ClientID, c.config.ClientSecret, code)
}

// ExchangeCode exchanges an authorization code for an access token.
func ExchangeCode(clientID, clientSecret, code string) (*TokenResponse, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("code", code)

	req.SetRequestURI(tokenURL)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBodyString(formData.Encode())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("github token exchange failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	return &tokenResp, nil
}

// GitHubGetPrimaryVerifiedEmail fetches the user's primary verified email.
func (c *Client) GitHubGetPrimaryVerifiedEmail(accessToken string) (string, error) {
	return GetPrimaryVerifiedEmail(accessToken)
}

// GetPrimaryVerifiedEmail fetches the user's primary verified email.
func GetPrimaryVerifiedEmail(accessToken string) (string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(userEmailURL)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "trip2g-oauth")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to get emails: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return "", fmt.Errorf("github emails failed: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	var emails []Email
	err = json.Unmarshal(resp.Body(), &emails)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal emails: %w", err)
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", errors.New("no primary verified email found")
}

// ErrorResponse represents a GitHub OAuth error response.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// ValidateCredentials validates OAuth credentials by making a test token exchange request.
// Returns nil if credentials are valid, error otherwise.
func ValidateCredentials(clientID, clientSecret string) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// Use a dummy code - we expect this to fail, but the error type tells us if credentials are valid
	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("code", "dummy_validation_code")

	req.SetRequestURI(tokenURL)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBodyString(formData.Encode())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	err := httpClient.DoTimeout(req, resp, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	// Parse response - GitHub returns 200 even for errors
	var errResp ErrorResponse
	err = json.Unmarshal(resp.Body(), &errResp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// "incorrect_client_credentials" means credentials are wrong
	// "bad_verification_code" means credentials are valid but code is wrong (expected)
	if errResp.Error == "incorrect_client_credentials" {
		return fmt.Errorf("invalid credentials: %s", errResp.ErrorDescription)
	}

	// Any other error (like bad_verification_code) means credentials are valid
	return nil
}
