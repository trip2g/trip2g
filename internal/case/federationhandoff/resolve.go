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

// targetPath is the page the peer opens. Always the peer's front page: a KB-note
// names a base, not a page in it.
const targetPath = "/"

func Resolve(ctx context.Context, env Env, notePath string) (Handoff, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return Handoff{}, fmt.Errorf("current user token: %w", err)
	}
	// An anonymous reader of a free KB-note gets a plain link instead.
	if token == nil || token.ID == 0 {
		return Handoff{}, fmt.Errorf("%w: no signed-in reader", ErrRefused)
	}
	if token.IsFederationGuest() {
		return Handoff{}, fmt.Errorf("%w: a guest cannot be handed further", ErrRefused)
	}

	nvs := env.LiveNoteViews()
	if nvs == nil {
		return Handoff{}, fmt.Errorf("%w: no notes loaded", ErrRefused)
	}
	note := nvs.PathMap[notePath]
	if note == nil {
		return Handoff{}, fmt.Errorf("%w: no note %q", ErrRefused, notePath)
	}

	canRead, err := env.CanReadNote(ctx, note)
	if err != nil {
		return Handoff{}, fmt.Errorf("can read note: %w", err)
	}
	if !canRead {
		return Handoff{}, fmt.Errorf("%w: reader cannot read %q", ErrRefused, notePath)
	}

	kb := model.NewMCPFederationNote(note)
	if kb == nil {
		return Handoff{}, fmt.Errorf("%w: %q is not a KB-note", ErrRefused, notePath)
	}

	peer, err := env.FederationGuestPeer(ctx, kb)
	if err != nil {
		return Handoff{}, fmt.Errorf("%w: %w", ErrRefused, err)
	}
	if len(peer.Secret) == 0 {
		return Handoff{}, fmt.Errorf("%w: no outbound key for the address %q", ErrRefused, kb.URL)
	}

	site, ok := model.MCPFederationSite(peer.KBURL)
	if !ok {
		return Handoff{}, fmt.Errorf("%w: no site address in the key's address %q", ErrRefused, peer.KBURL)
	}

	origin, ok := model.SiteOrigin(env.PublicURL())
	if !ok {
		return Handoff{}, fmt.Errorf("%w: no public URL to sign as this site's origin", ErrRefused)
	}

	if env.FederationMaxDepth() < Hop {
		return Handoff{}, fmt.Errorf("%w: federation depth %d allows no handoff", ErrRefused, env.FederationMaxDepth())
	}

	reader, err := readerID(ctx, env, token)
	if err != nil {
		return Handoff{}, err
	}

	handoff := mcp.GuestHandoff{Path: targetPath, Hop: Hop, Reader: reader, Origin: origin}
	signed, err := mcp.SignGuestHandoff(peer.Secret, peer.KID, peer.Issuer, handoff)
	if err != nil {
		return Handoff{}, err
	}

	return Handoff{Action: site + federationguest.Path, Token: signed, Path: targetPath}, nil
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
