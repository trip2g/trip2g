package oidcauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
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
			require.Equal(t, testBoolClaim(true), claims.EmailVerified)
		})
	}
}

func TestVerifyIDTokenPreservesEmailVerifiedState(t *testing.T) {
	key := mustRSAKey(t)
	kid := "key-1"
	jwks := jwksJSON(t, kid, &key.PublicKey)
	params := VerifyParams{JWKSURI: testJWKSURI, Issuer: testIssuer, ClientID: testClientID}

	tests := []struct {
		name    string
		mutate  func(jwt.MapClaims)
		want    BoolClaim
		wantErr bool
	}{
		{name: "true", mutate: func(jwt.MapClaims) {}, want: testBoolClaim(true)},
		{name: "false", mutate: func(c jwt.MapClaims) { c["email_verified"] = false }, want: testBoolClaim(false)},
		{name: "omitted", mutate: func(c jwt.MapClaims) { delete(c, "email_verified") }, want: BoolClaim{}},
		{name: "null", mutate: func(c jwt.MapClaims) { c["email_verified"] = nil }, want: BoolClaim{Present: true}},
		{name: "invalid type", mutate: func(c jwt.MapClaims) { c["email_verified"] = "true" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := validClaims()
			tt.mutate(claims)
			raw := signIDToken(t, key, kid, jwt.SigningMethodRS256, claims)

			cache := NewKeyCache()
			cache.fetch = func(context.Context, string) ([]byte, error) { return jwks, nil }
			got, err := cache.VerifyIDToken(context.Background(), raw, params)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.EmailVerified)
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
