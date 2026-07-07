package oidcauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// ErrVerification wraps every id_token verification failure so callers can
// treat an invalid/forged token as a validation (auth-rejected) error rather
// than a system error. Verification never panics.
var ErrVerification = errors.New("id_token verification failed")

// IDTokenClaims are the verified claims trip2g reads from an OIDC id_token.
type IDTokenClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Nonce         string
	Groups        []string
}

// VerifyParams configures id_token verification. Issuer/ClientID come from the
// stored credential; JWKSURI from discovery. Nonce is verified only when set.
type VerifyParams struct {
	JWKSURI  string
	Issuer   string
	ClientID string
	Nonce    string
}

type idTokenClaims struct {
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Nonce         string   `json:"nonce"`
	Groups        []string `json:"groups"`
	jwt.RegisteredClaims
}

// VerifyIDToken verifies an OIDC id_token's signature against the provider's
// JWKS and its standard claims (iss, aud, exp, iat), plus nonce when supplied.
// Only RSA signatures are accepted; alg=none and symmetric algorithms are
// rejected. Any failure returns an ErrVerification-wrapped error and nil claims.
func (c *KeyCache) VerifyIDToken(ctx context.Context, rawIDToken string, p VerifyParams) (*IDTokenClaims, error) {
	if rawIDToken == "" {
		return nil, fmt.Errorf("%w: empty id_token", ErrVerification)
	}
	if p.JWKSURI == "" || p.Issuer == "" || p.ClientID == "" {
		return nil, fmt.Errorf("%w: missing verification parameters", ErrVerification)
	}

	claims := &idTokenClaims{}
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)
		return c.keyForKID(ctx, p.JWKSURI, kid)
	}

	parsed, err := jwt.ParseWithClaims(rawIDToken, claims, keyFunc,
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		jwt.WithIssuer(p.Issuer),
		jwt.WithAudience(p.ClientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrVerification, err)
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("%w: token invalid", ErrVerification)
	}

	if p.Nonce != "" && claims.Nonce != p.Nonce {
		return nil, fmt.Errorf("%w: nonce mismatch", ErrVerification)
	}

	return &IDTokenClaims{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Nonce:         claims.Nonce,
		Groups:        claims.Groups,
	}, nil
}
