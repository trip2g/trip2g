// Package federationhandoff is the hub side of a federation handoff: a reader
// who can read a KB-note presses its button, and the hub signs, with that edge's
// secret, that the reader may look at the peer.
package federationhandoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"trip2g/internal/appresp"
	"trip2g/internal/case/federationguest"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg federationhandoff_test . Env

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	LiveNoteViews() *model.NoteViews
	CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	// FederationGuestPeer is the outbound key the KB-note's own address selects.
	FederationGuestPeer(ctx context.Context, kb *model.MCPFederationNote) (model.FederationPeer, error)
	FederationMaxDepth() int
	PublicURL() string
}

var ErrRefused = errors.New("federation handoff refused")

// Hop is the only hop this version signs: a reader opens one base from their
// own, and a guest cannot go further. The number stays in the token so that a
// later version can open chains without changing its format.
const Hop = 1

// Handoff is the form the reader's browser posts to the peer.
type Handoff struct {
	Action string
	Token  string
	Path   string
}

// ErrAnonymous is a refusal that signing in would lift.
var ErrAnonymous = errors.New("federation handoff needs a signed-in reader")

// prepared is a KB-note a reader may be handed over from, checked.
type prepared struct {
	token  *usertoken.Data
	kb     *model.MCPFederationNote
	peer   model.FederationPeer
	site   string
	target string
}

// sitePath reports whether target is a path on the site and nothing more.
func sitePath(target string) bool {
	return target != "" && appresp.SamePath(target) == target
}

// targetPage is the page the peer opens: the one asked for, else the one the
// KB-note names, else the front page. Whoever chose it, it is a path on the
// peer's site and never anything else.
func targetPage(kb *model.MCPFederationNote, asked string) (string, error) {
	switch {
	case asked != "":
		if !sitePath(asked) {
			return "", fmt.Errorf("%w: %q is not a path on the site", ErrRefused, asked)
		}
		return asked, nil
	case kb.Path != "":
		if !sitePath(kb.Path) {
			return "", fmt.Errorf("%w: the KB-note's landing page %q is not a path on the site", ErrRefused, kb.Path)
		}
		return kb.Path, nil
	}
	return "/", nil
}

func prepare(ctx context.Context, env Env, notePath, askedPage string) (prepared, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return prepared{}, fmt.Errorf("current user token: %w", err)
	}
	// An anonymous reader of a free KB-note gets a plain link instead.
	if token == nil || token.ID == 0 {
		return prepared{}, fmt.Errorf("%w: %w", ErrRefused, ErrAnonymous)
	}
	if token.IsFederationGuest() {
		return prepared{}, fmt.Errorf("%w: a guest cannot be handed further", ErrRefused)
	}

	nvs := env.LiveNoteViews()
	if nvs == nil {
		return prepared{}, fmt.Errorf("%w: no notes loaded", ErrRefused)
	}
	note := nvs.PathMap[notePath]
	if note == nil {
		return prepared{}, fmt.Errorf("%w: no note %q", ErrRefused, notePath)
	}

	canRead, err := env.CanReadNote(ctx, note)
	if err != nil {
		return prepared{}, fmt.Errorf("can read note: %w", err)
	}
	if !canRead {
		return prepared{}, fmt.Errorf("%w: reader cannot read %q", ErrRefused, notePath)
	}

	kb := model.NewMCPFederationNote(note)
	if kb == nil {
		return prepared{}, fmt.Errorf("%w: %q is not a KB-note", ErrRefused, notePath)
	}

	target, err := targetPage(kb, askedPage)
	if err != nil {
		return prepared{}, err
	}

	peer, err := env.FederationGuestPeer(ctx, kb)
	if err != nil {
		return prepared{}, fmt.Errorf("%w: %w", ErrRefused, err)
	}
	if len(peer.Secret) == 0 {
		return prepared{}, fmt.Errorf("%w: no outbound key for the address %q", ErrRefused, kb.URL)
	}

	site, ok := model.MCPFederationSite(peer.KBURL)
	if !ok {
		return prepared{}, fmt.Errorf("%w: no site address in the key's address %q", ErrRefused, peer.KBURL)
	}

	if env.FederationMaxDepth() < Hop {
		return prepared{}, fmt.Errorf("%w: federation depth %d allows no handoff", ErrRefused, env.FederationMaxDepth())
	}

	return prepared{token: token, kb: kb, peer: peer, site: site, target: target}, nil
}

// Resolve signs the handoff of a reader who may read the KB-note. askedPage is
// the page of the peer the reader came for, or empty for the KB-note's own.
func Resolve(ctx context.Context, env Env, notePath, askedPage string) (Handoff, error) {
	ready, err := prepare(ctx, env, notePath, askedPage)
	if err != nil {
		return Handoff{}, err
	}

	origin, ok := model.SiteOrigin(env.PublicURL())
	if !ok {
		return Handoff{}, fmt.Errorf("%w: no public URL to sign as this site's origin", ErrRefused)
	}

	reader, err := readerID(ctx, env, ready.token)
	if err != nil {
		return Handoff{}, err
	}

	handoff := mcp.GuestHandoff{Path: ready.target, Hop: Hop, Reader: reader, Origin: origin}
	signed, err := mcp.SignGuestHandoff(ready.peer.Secret, ready.peer.KID, ready.peer.Issuer, handoff)
	if err != nil {
		return Handoff{}, err
	}

	return Handoff{Action: ready.site + federationguest.Path, Token: signed, Path: ready.target}, nil
}

// Offer is what the open link shows before anything is signed: where the
// reader is about to go.
type Offer struct {
	Note string
	Path string
	Site string
}

// Open checks that the reader may be handed over and says where to, without
// signing anything: the page it fills only posts the handoff form on a button.
func Open(ctx context.Context, env Env, notePath, askedPage string) (Offer, error) {
	ready, err := prepare(ctx, env, notePath, askedPage)
	if err != nil {
		return Offer{}, err
	}
	return Offer{Note: notePath, Path: ready.target, Site: ready.site}, nil
}

// readerID names the reader to the peer. A local reader is the hash of their
// email, so two hubs name the same person alike; one without an email is a hash
// of this hub and their id.
func readerID(ctx context.Context, env Env, token *usertoken.Data) (string, error) {
	user, err := env.UserByID(ctx, int64(token.ID))
	if err != nil {
		return "", fmt.Errorf("user by id: %w", err)
	}

	if user.Email != nil && strings.TrimSpace(*user.Email) != "" {
		return sha256Hex(strings.ToLower(strings.TrimSpace(*user.Email))), nil
	}

	return sha256Hex(env.PublicURL() + "#user:" + strconv.FormatInt(user.ID, 10)), nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
