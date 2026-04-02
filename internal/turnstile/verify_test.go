package turnstile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_EmptySecretSkips(t *testing.T) {
	c := New(Config{SecretKey: ""})
	err := c.VerifyCaptcha(context.Background(), "some-token", "1.2.3.4")
	require.NoError(t, err)
}

func TestClient_EmptyTokenFails(t *testing.T) {
	c := New(Config{SecretKey: "secret"})
	err := c.VerifyCaptcha(context.Background(), "", "1.2.3.4")
	require.Error(t, err)
	require.Contains(t, err.Error(), "captcha token is required")
}

func TestClient_VerifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	c := New(Config{SecretKey: "secret-key"})
	c.verifyURL = server.URL
	err := c.VerifyCaptcha(context.Background(), "test-token", "1.2.3.4")
	require.NoError(t, err)
}

func TestClient_VerifyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success": false}`))
	}))
	defer server.Close()

	c := New(Config{SecretKey: "secret-key"})
	c.verifyURL = server.URL
	err := c.VerifyCaptcha(context.Background(), "test-token", "1.2.3.4")
	require.Error(t, err)
	require.Contains(t, err.Error(), "turnstile verification failed")
}

func TestClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := New(Config{SecretKey: "secret-key"})
	c.verifyURL = server.URL
	err := c.VerifyCaptcha(context.Background(), "test-token", "1.2.3.4")
	require.Error(t, err)
}
