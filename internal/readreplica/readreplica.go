// Package readreplica holds the pure decision + auth logic for trip2g's
// read-only replica mode. A replica serves safe (read) requests locally off a
// replicated SQLite file and forwards every mutating request to the leader
// (--leader-addr). Forwarded requests carry an X-Replica-Auth HMAC so the leader
// can authorize the forward and trust the real client IP it relays.
//
// This package is intentionally dependency-free and side-effect-free so the
// routing and signing rules are unit-testable without an HTTP stack.
package readreplica

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AuthHeader is the HTTP header carrying the replica→leader HMAC token.
const AuthHeader = "X-Replica-Auth"

// safeMethods are read-only per HTTP semantics and are served locally on a
// replica. Every other method (POST/PUT/PATCH/DELETE and any unknown verb) is
// treated as a write and forwarded to the leader.
var safeMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// IsWrite reports whether a request with the given HTTP method must be
// forwarded to the leader. The decision is made on the method alone, before any
// handler runs, so forwarding never replays handler side effects.
func IsWrite(method string) bool {
	_, safe := safeMethods[strings.ToUpper(method)]
	return !safe
}

// SignAuth returns an X-Replica-Auth value valid until now+ttl, signed with
// secret. Format: "<expiryUnix>.<base64url(hmacSHA256(expiryUnix))>".
func SignAuth(secret string, now time.Time, ttl time.Duration) string {
	expiry := strconv.FormatInt(now.Add(ttl).Unix(), 10)
	return expiry + "." + sign(secret, expiry)
}

// VerifyAuth checks header against secret at time now. It returns nil only when
// the signature is valid and the token has not expired.
func VerifyAuth(secret, header string, now time.Time) error {
	dot := strings.IndexByte(header, '.')
	if dot <= 0 {
		return errors.New("replica auth: malformed token")
	}

	expiryStr, sig := header[:dot], header[dot+1:]

	expected := sign(secret, expiryStr)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return errors.New("replica auth: bad signature")
	}

	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return errors.New("replica auth: bad expiry")
	}

	if now.Unix() > expiry {
		return errors.New("replica auth: token expired")
	}

	return nil
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
