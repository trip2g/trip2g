package oidcauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// newMockIssuer stands up an httptest server that behaves like a minimal OIDC
// provider: discovery, token, and userinfo endpoints all point back at itself.
func newMockIssuer(t *testing.T) (*httptest.Server, *capturedRequests) {
	t.Helper()

	captured := &capturedRequests{}

	mux := http.NewServeMux()
	var serverURL string

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		endpoints := Endpoints{
			Issuer:                serverURL,
			AuthorizationEndpoint: serverURL + "/authorize",
			TokenEndpoint:         serverURL + "/token",
			UserInfoEndpoint:      serverURL + "/userinfo",
			JWKSURI:               serverURL + "/jwks",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(endpoints)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured.tokenBody = string(body)
		captured.tokenContentType = r.Header.Get("Content-Type")

		resp := TokenResponse{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			IDToken:     "test-id-token",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		captured.userInfoAuth = r.Header.Get("Authorization")

		resp := UserInfo{
			Sub:           "user-123",
			Email:         "alice@example.com",
			EmailVerified: true,
			Name:          "Alice Example",
			Groups:        []string{"admins", "editors"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	t.Cleanup(server.Close)

	return server, captured
}

type capturedRequests struct {
	tokenBody        string
	tokenContentType string
	userInfoAuth     string
}

func TestDiscover(t *testing.T) {
	server, _ := newMockIssuer(t)

	tests := []struct {
		name   string
		issuer string
	}{
		{name: "no trailing slash", issuer: server.URL},
		{name: "trailing slash", issuer: server.URL + "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints, err := Discover(tt.issuer)
			require.NoError(t, err)
			require.Equal(t, server.URL, endpoints.Issuer)
			require.Equal(t, server.URL+"/authorize", endpoints.AuthorizationEndpoint)
			require.Equal(t, server.URL+"/token", endpoints.TokenEndpoint)
			require.Equal(t, server.URL+"/userinfo", endpoints.UserInfoEndpoint)
			require.Equal(t, server.URL+"/jwks", endpoints.JWKSURI)
		})
	}
}

func TestDiscoverNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	_, err := Discover(server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 404")
}

func TestExchangeCode(t *testing.T) {
	server, captured := newMockIssuer(t)

	endpoints, err := Discover(server.URL)
	require.NoError(t, err)

	token, err := ExchangeCode("client-id", "client-secret", "auth-code", "https://app/callback", endpoints.TokenEndpoint)
	require.NoError(t, err)
	require.Equal(t, "test-access-token", token.AccessToken)
	require.Equal(t, "Bearer", token.TokenType)
	require.Equal(t, 3600, token.ExpiresIn)
	require.Equal(t, "test-id-token", token.IDToken)

	require.Equal(t, "application/x-www-form-urlencoded", captured.tokenContentType)

	form, err := url.ParseQuery(captured.tokenBody)
	require.NoError(t, err)
	require.Equal(t, "authorization_code", form.Get("grant_type"))
	require.Equal(t, "auth-code", form.Get("code"))
	require.Equal(t, "client-id", form.Get("client_id"))
	require.Equal(t, "client-secret", form.Get("client_secret"))
	require.Equal(t, "https://app/callback", form.Get("redirect_uri"))
}

func TestGetUserInfo(t *testing.T) {
	server, captured := newMockIssuer(t)

	endpoints, err := Discover(server.URL)
	require.NoError(t, err)

	info, err := GetUserInfo("test-access-token", endpoints.UserInfoEndpoint)
	require.NoError(t, err)
	require.Equal(t, "user-123", info.Sub)
	require.Equal(t, "alice@example.com", info.Email)
	require.True(t, info.EmailVerified)
	require.Equal(t, "Alice Example", info.Name)
	require.Equal(t, []string{"admins", "editors"}, info.Groups)

	require.Equal(t, "Bearer test-access-token", captured.userInfoAuth)
}

func TestBuildAuthURL(t *testing.T) {
	authzEndpoint := "https://issuer.example.com/authorize"
	raw := BuildAuthURL(authzEndpoint, "client-id", "https://app/callback", "state-xyz", "openid email")

	require.True(t, strings.HasPrefix(raw, authzEndpoint+"?"))

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	q := parsed.Query()
	require.Equal(t, "code", q.Get("response_type"))
	require.Equal(t, "client-id", q.Get("client_id"))
	require.Equal(t, "https://app/callback", q.Get("redirect_uri"))
	require.Equal(t, "openid email", q.Get("scope"))
	require.Equal(t, "state-xyz", q.Get("state"))
}

func TestBuildAuthURLDefaultScopes(t *testing.T) {
	raw := BuildAuthURL("https://issuer.example.com/authorize", "client-id", "https://app/callback", "state-xyz", "")

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "openid email profile", parsed.Query().Get("scope"))
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{name: "all set", cfg: Config{Issuer: "https://i", ClientID: "c", ClientSecret: "s"}, want: true},
		{name: "missing issuer", cfg: Config{ClientID: "c", ClientSecret: "s"}, want: false},
		{name: "missing client id", cfg: Config{Issuer: "https://i", ClientSecret: "s"}, want: false},
		{name: "missing client secret", cfg: Config{Issuer: "https://i", ClientID: "c"}, want: false},
		{name: "empty", cfg: Config{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.cfg.IsConfigured())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.Equal(t, "openid email profile", cfg.Scopes)
	require.False(t, cfg.IsConfigured())
}
