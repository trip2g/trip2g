// Package federatedassetlink signs the short-lived link through which this
// instance relays one attachment of a federated peer's note.
//
// The link is the whole credential: an agent hands it to curl, which holds none
// of the MCP client's headers. So it names exactly one attachment and expires
// soon, and it is signed with a key derived from the instance secret rather than
// the secret itself — a session token and a link can never be mistaken for one
// another, whichever way round they are presented.
package federatedassetlink

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Link is the one attachment a link reaches.
type Link struct {
	KBID     string `json:"kb"`
	NotePath string `json:"np"`
	Name     string `json:"n"`
}

type claims struct {
	jwt.RegisteredClaims
	Link
}

func Sign(link Link, secret string, expiresAt time.Time) (string, error) {
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Link: link,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(signingKey(secret))
	if err != nil {
		return "", fmt.Errorf("sign federated asset link: %w", err)
	}

	return signed, nil
}

func Parse(token string, secret string) (Link, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(*jwt.Token) (any, error) {
		return signingKey(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return Link{}, fmt.Errorf("parse federated asset link: %w", err)
	}

	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return Link{}, errors.New("not a federated asset link")
	}

	return c.Link, nil
}

func signingKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("federated asset link"))
	return mac.Sum(nil)
}
