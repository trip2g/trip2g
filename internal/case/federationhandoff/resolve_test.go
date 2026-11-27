package federationhandoff_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"trip2g/internal/case/federationhandoff"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

const kbURL = "https://peer.test/_system/mcp"

func edgeSecret() []byte {
	return []byte("12345678901234567890123456789012")
}

func kbNote() *model.NoteView {
	return &model.NoteView{
		Path:               "peer.md",
		MCPFederationKBURL: kbURL,
		MCPFederationKBID:  "peer",
	}
}

func newEnv(token *usertoken.Data) *EnvMock {
	notes := model.NewNoteViews()
	notes.PathMap["peer.md"] = kbNote()
	notes.PathMap["plain.md"] = &model.NoteView{Path: "plain.md"}
	email := "Reader@Example.com "

	return &EnvMock{
		CurrentUserTokenFunc: func(context.Context) (*usertoken.Data, error) { return token, nil },
		LiveNoteViewsFunc:    func() *model.NoteViews { return notes },
		CanReadNoteFunc:      func(context.Context, *model.NoteView) (bool, error) { return true, nil },
		UserByIDFunc: func(_ context.Context, id int64) (db.User, error) {
			return db.User{ID: id, Email: &email}, nil
		},
		FederationGuestPeerFunc: func(_ context.Context, kb *model.MCPFederationNote) (model.FederationPeer, error) {
			return model.FederationPeer{KBID: kb.ID, KBURL: kbURL, KID: "edge", Secret: edgeSecret(), Issuer: "https://hub.test"}, nil
		},
		FederationMaxDepthFunc: func() int { return 3 },
		PublicURLFunc:          func() string { return "https://hub.test" },
	}
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// verified reads a handoff back the way the peer will.
func verified(t *testing.T, token string) mcp.GuestHandoffClaims {
	t.Helper()
	env := verifyEnv{}
	claims, err := mcp.VerifyGuestHandoff(t.Context(), env, token)
	require.NoError(t, err)
	return claims
}

func TestASignedInReaderIsHandedToThePeerTheNoteNames(t *testing.T) {
	env := newEnv(&usertoken.Data{ID: 5, Role: "user"})

	got, err := federationhandoff.Resolve(t.Context(), env, "peer.md")
	require.NoError(t, err)
	require.Equal(t, "https://peer.test/_system/federation/guest", got.Action)
	require.Equal(t, "/", got.Path)

	claims := verified(t, got.Token)
	want := mcp.GuestHandoff{Path: "/", Hop: 1, Reader: sha("reader@example.com"), Origin: "https://hub.test"}
	require.Equal(t, want, claims.GuestHandoff)
	require.Equal(t, "edge", claims.Caller.KID)
}

func TestAReaderWithoutEmailIsNamedByAHashOfTheHubUserID(t *testing.T) {
	env := newEnv(&usertoken.Data{ID: 5, Role: "user"})
	env.UserByIDFunc = func(_ context.Context, id int64) (db.User, error) { return db.User{ID: id}, nil }

	got, err := federationhandoff.Resolve(t.Context(), env, "peer.md")
	require.NoError(t, err)
	require.Equal(t, sha("https://hub.test#user:5"), verified(t, got.Token).Reader)
}

func TestTheSiteAddressComesFromTheKeyNotTheNote(t *testing.T) {
	env := newEnv(&usertoken.Data{ID: 5, Role: "user"})
	env.FederationGuestPeerFunc = func(_ context.Context, kb *model.MCPFederationNote) (model.FederationPeer, error) {
		return model.FederationPeer{
			KBID: kb.ID, KBURL: "https://keyed.test/_system/mcp", KID: "edge", Secret: edgeSecret(), Issuer: "https://hub.test",
		}, nil
	}

	got, err := federationhandoff.Resolve(t.Context(), env, "peer.md")
	require.NoError(t, err)
	require.Equal(t, "https://keyed.test/_system/federation/guest", got.Action, "the note says peer.test, the key says keyed.test")
}

func TestResolveRefuses(t *testing.T) {
	tests := []struct {
		name  string
		token *usertoken.Data
		note  string
		env   func(*EnvMock)
	}{
		{name: "anonymous reader", token: nil, note: "peer.md"},
		{name: "unknown note", token: &usertoken.Data{ID: 5, Role: "user"}, note: "missing.md"},
		{name: "note is not a KB-note", token: &usertoken.Data{ID: 5, Role: "user"}, note: "plain.md"},
		{name: "reader cannot read the KB-note", token: &usertoken.Data{ID: 5, Role: "user"}, note: "peer.md", env: func(e *EnvMock) {
			e.CanReadNoteFunc = func(context.Context, *model.NoteView) (bool, error) { return false, nil }
		}},
		{name: "no outbound key for the peer", token: &usertoken.Data{ID: 5, Role: "user"}, note: "peer.md", env: func(e *EnvMock) {
			e.FederationGuestPeerFunc = func(_ context.Context, kb *model.MCPFederationNote) (model.FederationPeer, error) {
				return model.FederationPeer{KBID: kb.ID, KBURL: kbURL}, nil
			}
		}},
		{name: "hub without a public URL to sign as its origin", token: &usertoken.Data{ID: 5, Role: "user"}, note: "peer.md", env: func(e *EnvMock) {
			e.PublicURLFunc = func() string { return "" }
		}},
		{
			name:  "a guest is not handed further",
			token: &usertoken.Data{ID: 9, Role: usertoken.RoleFederationGuest, Kid: "in", Hop: 1, Reader: sha("x")},
			note:  "peer.md",
		},
		{name: "a federation depth that allows no hop", token: &usertoken.Data{ID: 5, Role: "user"}, note: "peer.md", env: func(e *EnvMock) {
			e.FederationMaxDepthFunc = func() int { return 0 }
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(tt.token)
			if tt.env != nil {
				tt.env(env)
			}

			_, err := federationhandoff.Resolve(t.Context(), env, tt.note)
			require.ErrorIs(t, err, federationhandoff.ErrRefused)
		})
	}
}
