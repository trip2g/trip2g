// Package federationguest is the peer side of a federation handoff: a hub signs
// that one of its readers may look, and this instance opens a read-only guest
// session scoped to the edge the hub signed with.
//
// It takes two requests. The hub's page posts the handoff here cross-site, and a
// SameSite=Lax session cookie does not ride a cross-site POST, so that request
// cannot tell whether the browser already holds its own session. It answers with
// a short ticket and a 303; the GET that follows is a top-level navigation, the
// cookie rides it, and only that request decides whether to set one.
package federationguest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/appresp"
	"trip2g/internal/case/mcp"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	jwt "github.com/golang-jwt/jwt/v5"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg federationguest_test . Env

type Env interface {
	FederationGuestsEnabled() bool
	FederationMaxDepth() int
	VerifyGuestHandoff(ctx context.Context, token string) (mcp.GuestHandoffClaims, error)
	// SpendFederationGuestID marks a handoff or ticket id used and reports
	// whether it was still unused.
	SpendFederationGuestID(id string, expiresAt time.Time) bool
	// FederationGuestUser finds or creates the synthetic user of the pair
	// (edge, reader).
	FederationGuestUser(ctx context.Context, kid, reader string) (int64, error)
	GuestTicketSecret() string
	// Insecure drops Secure from the nonce cookie, as from the session cookie.
	Insecure() bool
}

// ErrRefused is every reason a handoff or ticket does not open a session. The
// visitor is told only that; the reason goes to the log.
var ErrRefused = errors.New("federation guest refused")

// TicketTTL is how long a ticket and its nonce cookie live.
const TicketTTL = 30 * time.Second

type ticketClaims struct {
	UserID int64  `json:"u"`
	Kid    string `json:"k"`
	Hop    int    `json:"h"`
	Reader string `json:"rd"`
	Path   string `json:"p"`
	// NonceHash binds the ticket to the browser that posted the handoff: the
	// hash of the nonce that browser got in a cookie. The nonce itself never
	// travels in the URL.
	NonceHash string `json:"n"`
	jwt.RegisteredClaims
}

// Admission is what an accepted handoff hands the browser: the ticket for the
// URL of the GET leg, and the nonce for the cookie that must ride with it.
type Admission struct {
	Ticket string
	Nonce  string
}

// Form is a posted handoff: the token, the target the form carried beside it,
// and the Origin header of the page that posted it.
type Form struct {
	Token  string
	Path   string
	Origin string
}

// Admit checks a handoff and returns the ticket its GET leg exchanges for a
// session. The form must carry the target the hub signed and come from a page
// at the origin the hub signed: a hub reader who copies the form onto another
// site cannot open a session in someone else's browser with it.
func Admit(ctx context.Context, env Env, form Form) (Admission, error) {
	if !env.FederationGuestsEnabled() {
		return Admission{}, fmt.Errorf("%w: guests are not accepted here", ErrRefused)
	}

	claims, err := env.VerifyGuestHandoff(ctx, form.Token)
	if err != nil {
		return Admission{}, fmt.Errorf("%w: %w", ErrRefused, err)
	}

	signedOrigin, signedOK := model.SiteOrigin(claims.Origin)
	postedOrigin, postedOK := model.SiteOrigin(form.Origin)

	switch {
	case !signedOK || !postedOK || signedOrigin != postedOrigin:
		return Admission{}, fmt.Errorf("%w: posted from %q, signed for %q", ErrRefused, form.Origin, claims.Origin)
	case claims.Path != form.Path:
		return Admission{}, fmt.Errorf("%w: token is for %q, not %q", ErrRefused, claims.Path, form.Path)
	case appresp.SamePath(claims.Path) != claims.Path:
		return Admission{}, fmt.Errorf("%w: target %q is not a path on this site", ErrRefused, claims.Path)
	case claims.Hop < 1 || claims.Hop > env.FederationMaxDepth():
		return Admission{}, fmt.Errorf("%w: hop %d outside 1..%d", ErrRefused, claims.Hop, env.FederationMaxDepth())
	case !isReaderID(claims.Reader):
		return Admission{}, fmt.Errorf("%w: reader id is not a sha256 hex", ErrRefused)
	}

	if !env.SpendFederationGuestID(claims.ID, claims.ExpiresAt) {
		return Admission{}, fmt.Errorf("%w: handoff %s already used", ErrRefused, claims.ID)
	}

	userID, err := env.FederationGuestUser(ctx, claims.Caller.KID, claims.Reader)
	if err != nil {
		return Admission{}, fmt.Errorf("federation guest user: %w", err)
	}

	id, err := randomID()
	if err != nil {
		return Admission{}, err
	}
	nonce, err := randomID()
	if err != nil {
		return Admission{}, err
	}
	ticket := ticketClaims{
		UserID:    userID,
		Kid:       claims.Caller.KID,
		Hop:       claims.Hop,
		Reader:    claims.Reader,
		Path:      claims.Path,
		NonceHash: nonceHash(nonce),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TicketTTL)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, ticket).SignedString(ticketKey(env))
	if err != nil {
		return Admission{}, fmt.Errorf("sign guest ticket: %w", err)
	}
	return Admission{Ticket: signed, Nonce: nonce}, nil
}

// Enter exchanges a ticket for the guest session it stands for and the path to
// open. nonce is the cookie the browser carries; only the browser that posted
// the handoff has it. Whether to store that session is the caller's: an
// existing own session wins.
//
// A ticket is spent only once the nonce matches. Spending it on a mismatch
// would let anyone who sees the URL burn a genuine reader's ticket.
func Enter(env Env, ticket, nonce string) (usertoken.Data, string, error) {
	if !env.FederationGuestsEnabled() {
		return usertoken.Data{}, "", fmt.Errorf("%w: guests are not accepted here", ErrRefused)
	}

	var claims ticketClaims
	_, err := jwt.ParseWithClaims(ticket, &claims, func(*jwt.Token) (any, error) {
		return ticketKey(env), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return usertoken.Data{}, "", fmt.Errorf("%w: %w", ErrRefused, err)
	}

	if nonce == "" || subtle.ConstantTimeCompare([]byte(nonceHash(nonce)), []byte(claims.NonceHash)) != 1 {
		return usertoken.Data{}, "", fmt.Errorf("%w: ticket opened outside the browser that posted it", ErrRefused)
	}

	if claims.ID == "" || !env.SpendFederationGuestID(claims.ID, claims.ExpiresAt.Time) {
		return usertoken.Data{}, "", fmt.Errorf("%w: ticket already used", ErrRefused)
	}

	data := usertoken.Data{
		ID:     int(claims.UserID),
		Role:   usertoken.RoleFederationGuest,
		Kid:    claims.Kid,
		Hop:    claims.Hop,
		Reader: claims.Reader,
	}
	return data, appresp.SamePath(claims.Path), nil
}

// ticketKey keeps the ticket from being a valid signature for anything else
// the same secret signs.
func ticketKey(env Env) []byte {
	return []byte("federation guest ticket:" + env.GuestTicketSecret())
}

func nonceHash(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(sum[:])
}

func isReaderID(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
