package shortapitoken

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Data holds the claims embedded in a short API token.
type Data struct {
	Depth         int      `json:"d"`
	ReadPatterns  []string `json:"rp"`
	WritePatterns []string `json:"wp"`
}

type claims struct {
	jwt.RegisteredClaims
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
		return Data{}, fmt.Errorf("invalid short API token claims")
	}

	return c.Data, nil
}
