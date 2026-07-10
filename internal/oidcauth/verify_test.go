package oidcauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer   = "https://issuer.example.com/"
	testClientID = "test-client-id"
	testJWKSURI  = "https://issuer.example.com/jwks"
)

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// jwksJSON renders a single-key JWKS document for the given kid + public key.
func jwksJSON(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()
	doc := jwksDocument{Keys: []jwk{{
		Kty: "RSA",
		Kid: kid,
		Use: "sig",
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big2bytes(pub.E)),
	}}}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

func big2bytes(e int) []byte {
	var b []byte
	for e > 0 {
		b = append([]byte{byte(e & 0xff)}, b...)
		e >>= 8
	}
	return b
}

// signIDToken signs a JWT with the given method, kid, and claims.
func signIDToken(t *testing.T, key *rsa.PrivateKey, kid string, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	if kid != "" {
		token.Header["kid"] = kid
	}
	var (
		signed string
		err    error
	)
	if method == jwt.SigningMethodNone {
		signed, err = token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	} else {
		signed, err = token.SignedString(key)
	}
	require.NoError(t, err)
	return signed
}

func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":            testIssuer,
		"aud":            testClientID,
		"sub":            "user-123",
		"email":          "alice@example.com",
		"email_verified": true,
		"groups":         []string{"admins"},
		"iat":            now.Unix(),
		"exp":            now.Add(time.Hour).Unix(),
	}
}

func TestVerifyIDToken(t *testing.T) {
	key := mustRSAKey(t)
	kid := "key-1"
	jwks := jwksJSON(t, kid, &key.PublicKey)

	otherKey := mustRSAKey(t)

	baseParams := VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID}

	tests := []struct {
		name    string
		token   string
		params  VerifyParams
		wantErr bool
	}{
		{
			name:   "valid token",
			token:  signIDToken(t, key, kid, jwt.SigningMethodRS256, validClaims()),
			params: baseParams,
		},
		{
			name:    "wrong signature",
			token:   signIDToken(t, otherKey, kid, jwt.SigningMethodRS256, validClaims()),
			params:  baseParams,
			wantErr: true,
		},
		{
			name: "wrong issuer",
			token: signIDToken(t, key, kid, jwt.SigningMethodRS256, func() jwt.MapClaims {
				c := validClaims()
				c["iss"] = "https://evil.example.com/"
				return c
			}()),
			params:  baseParams,
			wantErr: true,
		},
		{
			name: "wrong audience",
			token: signIDToken(t, key, kid, jwt.SigningMethodRS256, func() jwt.MapClaims {
				c := validClaims()
				c["aud"] = "some-other-client"
				return c
			}()),
			params:  baseParams,
			wantErr: true,
		},
		{
			name: "expired",
			token: signIDToken(t, key, kid, jwt.SigningMethodRS256, func() jwt.MapClaims {
				c := validClaims()
				c["exp"] = time.Now().Add(-time.Hour).Unix()
				return c
			}()),
			params:  baseParams,
			wantErr: true,
		},
		{
			name: "missing exp",
			token: signIDToken(t, key, kid, jwt.SigningMethodRS256, func() jwt.MapClaims {
				c := validClaims()
				delete(c, "exp")
				return c
			}()),
			params:  baseParams,
			wantErr: true,
		},
		{
			name:    "alg none rejected",
			token:   signIDToken(t, key, kid, jwt.SigningMethodNone, validClaims()),
			params:  baseParams,
			wantErr: true,
		},
		{
			name: "nonce match",
			token: signIDToken(t, key, kid, jwt.SigningMethodRS256, func() jwt.MapClaims {
				c := validClaims()
				c["nonce"] = "n-123"
				return c
			}()),
			params: VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID, Nonce: "n-123"},
		},
		{
			name: "nonce mismatch",
			token: signIDToken(t, key, kid, jwt.SigningMethodRS256, func() jwt.MapClaims {
				c := validClaims()
				c["nonce"] = "n-123"
				return c
			}()),
			params:  VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID, Nonce: "expected-other"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewKeyCache()
			cache.fetch = func(_ context.Context, _ string) ([]byte, error) { return jwks, nil }

			claims, err := cache.VerifyIDToken(context.Background(), tt.token, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, claims)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, claims)
			require.Equal(t, "user-123", claims.Subject)
			require.Equal(t, "alice@example.com", claims.Email)
		})
	}
}

func TestVerifyIDTokenCachesKeys(t *testing.T) {
	key := mustRSAKey(t)
	kid := "key-1"
	jwks := jwksJSON(t, kid, &key.PublicKey)

	var fetches int32
	cache := NewKeyCache()
	cache.fetch = func(_ context.Context, _ string) ([]byte, error) {
		atomic.AddInt32(&fetches, 1)
		return jwks, nil
	}

	params := VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID}
	for range 3 {
		_, err := cache.VerifyIDToken(context.Background(), signIDToken(t, key, kid, jwt.SigningMethodRS256, validClaims()), params)
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), atomic.LoadInt32(&fetches), "JWKS should be fetched once and cached")
}

// ecJWKSJSON renders a single-key JWKS document for an EC P-256 public key.
func ecJWKSJSON(t *testing.T, kid string, pub *ecdsa.PublicKey) []byte {
	t.Helper()
	doc := map[string]any{"keys": []map[string]any{{
		"kty": "EC",
		"kid": kid,
		"use": "sig",
		"alg": "ES256",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(pub.X.FillBytes(make([]byte, 32))),
		"y":   base64.RawURLEncoding.EncodeToString(pub.Y.FillBytes(make([]byte, 32))),
	}}}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

// IdPs like Auth0, Keycloak, and Apple commonly sign id_tokens with EC keys
// (ES256). Verification must accept them, not just RSA.
func TestVerifyIDTokenAcceptsES256(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	kid := "ec-key-1"
	jwks := ecJWKSJSON(t, kid, &ecKey.PublicKey)

	token := jwt.NewWithClaims(jwt.SigningMethodES256, validClaims())
	token.Header["kid"] = kid
	signed, err := token.SignedString(ecKey)
	require.NoError(t, err)

	cache := NewKeyCache()
	cache.fetch = func(_ context.Context, _ string) ([]byte, error) { return jwks, nil }

	claims, err := cache.VerifyIDToken(context.Background(),
		signed, VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID})
	require.NoError(t, err, "ES256-signed id_token with an EC JWKS key must verify")
	require.NotNil(t, claims)
	require.Equal(t, "user-123", claims.Subject)
}

// A hanging JWKS fetch for one URI must not block verification with keys
// already cached for another URI.
func TestKeyCacheCachedReadsNotBlockedByInflightFetch(t *testing.T) {
	key := mustRSAKey(t)
	kid := "cached-kid"
	jwks := jwksJSON(t, kid, &key.PublicKey)

	const slowURI = "https://slow-idp.example.com/jwks"
	release := make(chan struct{})
	fetchStarted := make(chan struct{})

	cache := NewKeyCache()
	cache.fetch = func(_ context.Context, uri string) ([]byte, error) {
		if uri == slowURI {
			close(fetchStarted)
			<-release
			return nil, errors.New("slow jwks endpoint gave up")
		}
		return jwks, nil
	}

	params := VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID}

	// Warm the cache for the fast URI.
	_, err := cache.VerifyIDToken(context.Background(), signIDToken(t, key, kid, jwt.SigningMethodRS256, validClaims()), params)
	require.NoError(t, err)

	// Goroutine A hangs inside the JWKS fetch for the slow URI.
	slowDone := make(chan struct{})
	t.Cleanup(func() { <-slowDone })
	t.Cleanup(func() { close(release) })
	go func() {
		defer close(slowDone)
		_, _ = cache.keyForKID(context.Background(), slowURI, "unknown-kid")
	}()
	<-fetchStarted

	// Goroutine B verifies with the already-cached key while A is blocked.
	cachedToken := signIDToken(t, key, kid, jwt.SigningMethodRS256, validClaims())
	verified := make(chan error, 1)
	go func() {
		_, verr := cache.VerifyIDToken(context.Background(), cachedToken, params)
		verified <- verr
	}()

	select {
	case verr := <-verified:
		require.NoError(t, verr)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cached-key verification blocked behind an in-flight JWKS fetch for another URI")
	}
}

func TestVerifyIDTokenRefetchesOnUnknownKID(t *testing.T) {
	keyA := mustRSAKey(t)
	keyB := mustRSAKey(t)

	current := jwksJSON(t, "key-a", &keyA.PublicKey)
	var fetches int32

	cache := NewKeyCache()
	cache.fetch = func(_ context.Context, _ string) ([]byte, error) {
		atomic.AddInt32(&fetches, 1)
		return current, nil
	}

	params := VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID}

	// Warm the cache with key-a.
	_, err := cache.VerifyIDToken(context.Background(), signIDToken(t, keyA, "key-a", jwt.SigningMethodRS256, validClaims()), params)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&fetches))

	// Rotate: issuer now serves key-b, token signed with the new kid.
	current = jwksJSON(t, "key-b", &keyB.PublicKey)
	_, err = cache.VerifyIDToken(context.Background(), signIDToken(t, keyB, "key-b", jwt.SigningMethodRS256, validClaims()), params)
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&fetches), "unknown kid should trigger a refetch")
}
