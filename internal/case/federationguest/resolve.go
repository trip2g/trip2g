// Package federationguest is the peer side of a federation handoff: a hub signs
// that one of its readers may look, and this instance opens a read-only guest
// session scoped to the edge the hub signed with.
//
// It takes two requests. The hub's page posts the handoff here cross-site, and a
// SameSite=Lax session cookie does not ride a cross-site POST, so that request
// cannot tell whether the browser already holds its own session. It answers with
// an opaque ticket and a 303; the GET that follows is a top-level navigation, the
// cookie rides it, and only that request decides whether to set one. What the
// ticket stands for stays on the server.
package federationguest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"trip2g/internal/appresp"
	"trip2g/internal/case/mcp"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg federationguest_test . Env

type Env interface {
	FederationMaxDepth() int
	VerifyGuestHandoff(ctx context.Context, token string) (mcp.GuestHandoffClaims, error)
	// SpendFederationGuestID marks a handoff or ticket id used and reports
	// whether it was still unused.
	SpendFederationGuestID(id string, expiresAt time.Time) bool
	// FederationGuestUser finds or creates the synthetic user of the pair
	// (edge, reader).
	FederationGuestUser(ctx context.Context, kid, reader string) (int64, error)
	// GuestTickets holds what an issued ticket stands for until it is opened.
	GuestTickets() *Tickets
	// Insecure drops Secure from the nonce cookie, as from the session cookie.
	Insecure() bool
}

// ErrRefused is every reason a handoff or ticket does not open a session. The
// visitor is told only that; the reason goes to the log.
var ErrRefused = errors.New("federation guest refused")

// Refusal is a refusal of a handoff whose signature was genuine: it knows the
// hub the reader came from, so the refusal page can lead back there. A handoff
// that failed its signature names no hub.
type Refusal struct {
	Back string
	Err  error
}

func (r *Refusal) Error() string { return r.Err.Error() }
func (r *Refusal) Unwrap() error { return r.Err }

// TicketTTL is how long a ticket and its nonce cookie live.
const TicketTTL = 30 * time.Second

// ticket is what an issued ticket id stands for.
type ticket struct {
	session usertoken.Data
	path    string
	// nonceHash binds the ticket to the browser that posted the handoff: the
	// hash of the nonce that browser got in a cookie. The nonce itself never
	// travels in the URL.
	nonceHash string
	expiresAt time.Time
}

// Tickets are the issued, unopened tickets; the zero value is ready to use. A ticket id is random and means
// nothing outside this store, so a URL in a log or a browser history carries
// neither the reader nor the edge.
type Tickets struct {
	mu    sync.Mutex
	items map[string]ticket
}

func (t *Tickets) put(id string, item ticket) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.items == nil {
		t.items = map[string]ticket{}
	}

	now := time.Now()
	for key, other := range t.items {
		if now.After(other.expiresAt) {
			delete(t.items, key)
		}
	}
	t.items[id] = item
}

// take returns the ticket and forgets it, but only for the nonce that was bound
// to it: a mismatch leaves the ticket alone, so that someone who sees the URL
// cannot burn a genuine reader's ticket.
func (t *Tickets) take(id, nonce string) (ticket, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	item, ok := t.items[id]
	if !ok || time.Now().After(item.expiresAt) {
		delete(t.items, id)
		return ticket{}, false
	}
	if nonce == "" || subtle.ConstantTimeCompare([]byte(nonceHash(nonce)), []byte(item.nonceHash)) != 1 {
		return ticket{}, false
	}

	delete(t.items, id)
	return item, true
}

// Hop is the only hop a base admits: a reader arrives from their own base and a
// guest goes no further. The number stays in the token, so that a later version
// can open chains without changing its format.
const Hop = 1

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
	claims, err := env.VerifyGuestHandoff(ctx, form.Token)
	if err != nil {
		return Admission{}, fmt.Errorf("%w: %w", ErrRefused, err)
	}
	admission, err := admit(ctx, env, form, claims)
	if err != nil {
		back, _ := model.SiteOrigin(claims.Origin)
		return Admission{}, &Refusal{Back: back, Err: err}
	}
	return admission, nil
}

func admit(ctx context.Context, env Env, form Form, claims mcp.GuestHandoffClaims) (Admission, error) {
	if !claims.AcceptGuests {
		return Admission{}, fmt.Errorf("%w: the key %q does not accept people", ErrRefused, claims.Caller.KID)
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
	case claims.Hop != Hop:
		return Admission{}, fmt.Errorf("%w: hop %d, this version admits only hop %d", ErrRefused, claims.Hop, Hop)
	case claims.Hop > env.FederationMaxDepth():
		return Admission{}, fmt.Errorf("%w: hop %d over the federation depth %d", ErrRefused, claims.Hop, env.FederationMaxDepth())
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

	issued := ticket{
		session: usertoken.Data{
			ID:     int(userID),
			Role:   usertoken.RoleFederationGuest,
			Kid:    claims.Caller.KID,
			Key:    claims.KeyID,
			Hop:    claims.Hop,
			Reader: claims.Reader,
			Origin: signedOrigin,
		},
		path:      claims.Path,
		nonceHash: nonceHash(nonce),
		expiresAt: time.Now().Add(TicketTTL),
	}
	env.GuestTickets().put(id, issued)
	return Admission{Ticket: id, Nonce: nonce}, nil
}

// Enter exchanges a ticket for the guest session it stands for and the path to
// open. nonce is the cookie the browser carries; only the browser that posted
// the handoff has it. Whether to store that session is the caller's: an
// existing own session wins.
func Enter(env Env, ticketID, nonce string) (usertoken.Data, string, error) {
	item, ok := env.GuestTickets().take(ticketID, nonce)
	if !ok {
		return usertoken.Data{}, "", fmt.Errorf("%w: no such ticket for this browser", ErrRefused)
	}

	return item.session, appresp.SamePath(item.path), nil
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
