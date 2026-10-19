// Package federatedassetlink signs the short-lived link through which this
// instance relays one attachment of a federated peer's note.
//
// The link is the whole credential: an agent hands it to curl, which holds none
// of the MCP client's headers. So it names exactly one attachment and expires
// soon, and it is signed with a key derived from the instance secret rather than
// the secret itself — a session token and a link can never be mistaken for one
// another, whichever way round they are presented. Each link carries its own id,
// and Spent lets it be used once.
package federatedassetlink

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Link is the one attachment a link reaches.
type Link struct {
	KBID     string `json:"kb"`
	NotePath string `json:"np"`
	Name     string `json:"n"`
}

// Claim is a verified link: the attachment, the link's own id and its expiry.
type Claim struct {
	Link

	ID        string
	ExpiresAt time.Time
}

type claims struct {
	jwt.RegisteredClaims
	Link
}

func Sign(link Link, secret string, expiresAt time.Time) (string, error) {
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        rand.Text(),
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

func Parse(token string, secret string) (Claim, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) {
		return signingKey(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return Claim{}, fmt.Errorf("parse federated asset link: %w", err)
	}
	if c.ID == "" {
		return Claim{}, errors.New("parse federated asset link: no id")
	}

	return Claim{Link: c.Link, ID: c.ID, ExpiresAt: c.ExpiresAt.Time}, nil
}

// Spent holds the ids of links already used, each only until the link expires:
// after that Parse refuses the link anyway. The zero value is ready. It lives in
// one process, so each process that serves links keeps its own.
type Spent struct {
	mu       sync.Mutex
	expiries map[string]time.Time
}

// SpendFederatedAssetLink marks the link id as used and reports whether it was
// still unused.
func (s *Spent) SpendFederatedAssetLink(id string, expiresAt time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for spentID, exp := range s.expiries {
		if now.After(exp) {
			delete(s.expiries, spentID)
		}
	}

	if _, ok := s.expiries[id]; ok {
		return false
	}
	if s.expiries == nil {
		s.expiries = map[string]time.Time{}
	}
	s.expiries[id] = expiresAt

	return true
}

func signingKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("federated asset link"))
	return mac.Sum(nil)
}
