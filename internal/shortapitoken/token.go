package shortapitoken

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tokenType is the positive discriminator that distinguishes a short API token
// from other HS256 JWTs signed with the same secret (notably session-login
// tokens). Parse requires it, so a JWT lacking it is rejected as not-a-short-
// API-token rather than silently accepted with empty scope.
const tokenType = "sat" // short API token

// Data holds the claims embedded in a short API token.
type Data struct {
	Depth         int      `json:"d"`
	ReadPatterns  []string `json:"rp"`
	WritePatterns []string `json:"wp"`
	DeliveryKind  string   `json:"dk,omitempty"` // "change" | "cron"
	DeliveryID    int64    `json:"di,omitempty"`
}

type claims struct {
	jwt.RegisteredClaims
	// Typ is the token-type discriminator; always tokenType for short API
	// tokens. Session JWTs never carry it.
	Typ string `json:"typ"`
	Data
}

// Sign creates a signed JWT token with the given data and TTL.
func Sign(d Data, secret string, ttl time.Duration) (string, error) {
	now := time.Now()

	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Typ:  tokenType,
		Data: d,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign short API token: %w", err)
	}

	return signed, nil
}

// Parse validates and parses a short API token.
func Parse(tokenStr string, secret string) (Data, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse short API token: %w", err)
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return Data{}, errors.New("invalid short API token claims")
	}

	// Require the positive discriminator: a validly-signed JWT that lacks it
	// (e.g. a session-login token sharing the signing secret) is NOT a short
	// API token and must be rejected to avoid token-type confusion.
	if c.Typ != tokenType {
		return Data{}, errors.New("not a short API token")
	}

	return c.Data, nil
}
