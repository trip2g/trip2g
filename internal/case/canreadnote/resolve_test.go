package canreadnote_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg canreadnote_test . Env

type Env interface {
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
}

func TestResolve_AdminAccess(t *testing.T) {
	tests := []struct {
		name string
		note *model.NoteView
	}{
		{
			name: "admin can see free notes",
			note: &model.NoteView{
				Title:         "Free Note",
				Free:          true,
				SubgraphNames: []string{},
			},
		},
		{
			name: "admin can see paid notes without subgraphs",
			note: &model.NoteView{
				Title:         "Paid General Note",
				Free:          false,
				SubgraphNames: []string{},
			},
		},
		{
			name: "admin can see paid notes with subgraphs",
			note: &model.NoteView{
				Title:         "Premium Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{
						ID:   1,
						Role: "admin",
					}, nil
				},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, tt.note)

			require.NoError(t, err)
			require.True(t, hasAccess, "admin should have access to all notes")
		})
	}
}

func TestResolve_GuestAccess(t *testing.T) {
	tests := []struct {
		name           string
		note           *model.NoteView
		expectedAccess bool
	}{
		{
			name: "guest can see free notes",
			note: &model.NoteView{
				Title:         "Free Note",
				Free:          true,
				SubgraphNames: []string{},
			},
			expectedAccess: true,
		},
		{
			name: "guest cannot see paid notes without subgraphs",
			note: &model.NoteView{
				Title:         "Paid General Note",
				Free:          false,
				SubgraphNames: []string{},
			},
			expectedAccess: false,
		},
		{
			name: "guest cannot see paid notes with subgraphs",
			note: &model.NoteView{
				Title:         "Premium Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
			},
			expectedAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, nil // guest user
				},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, tt.note)

			require.NoError(t, err)
			require.Equal(t, tt.expectedAccess, hasAccess)
		})
	}
}

func TestResolve_AuthenticatedUserWithoutSubscriptions(t *testing.T) {
	tests := []struct {
		name string
		note *model.NoteView
	}{
		{
			name: "user without subscriptions cannot see paid notes without subgraphs",
			note: &model.NoteView{
				Title:         "General Knowledge Note",
				Free:          false,
				SubgraphNames: []string{},
			},
		},
		{
			name: "user without subscriptions cannot see paid notes with subgraphs",
			note: &model.NoteView{
				Title:         "Premium Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{
						ID:   123,
						Role: "user",
					}, nil
				},
				ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
					return []string{}, nil // no active subscriptions
				},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, tt.note)

			require.NoError(t, err)
			require.False(t, hasAccess, "user without subscriptions should not have access to paid content")
		})
	}
}

func TestResolve_AuthenticatedUserWithSubscriptions(t *testing.T) {
	tests := []struct {
		name           string
		note           *model.NoteView
		userSubgraphs  []string
		expectedAccess bool
		description    string
	}{
		{
			name: "user with subscription can see general knowledge notes",
			note: &model.NoteView{
				Title:         "General Knowledge Note",
				Free:          false,
				SubgraphNames: []string{},
			},
			userSubgraphs:  []string{"premium"},
			expectedAccess: true,
			description:    "notes without subgraphs should be accessible to users with any active subscription",
		},
		{
			name: "user can see notes from their subscribed subgraphs",
			note: &model.NoteView{
				Title:         "Premium Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
			},
			userSubgraphs:  []string{"premium"},
			expectedAccess: true,
			description:    "user should have access to notes from subgraphs they're subscribed to",
		},
		{
			name: "user cannot see notes from non-subscribed subgraphs",
			note: &model.NoteView{
				Title:         "VIP Note",
				Free:          false,
				SubgraphNames: []string{"vip"},
			},
			userSubgraphs:  []string{"premium"},
			expectedAccess: false,
			description:    "user should not have access to notes from subgraphs they're not subscribed to",
		},
		{
			name: "user can see notes with multiple subgraphs if they have access to at least one",
			note: &model.NoteView{
				Title:         "Multi-Subgraph Note",
				Free:          false,
				SubgraphNames: []string{"premium", "vip"},
			},
			userSubgraphs:  []string{"premium"},
			expectedAccess: true,
			description:    "user should have access if they're subscribed to any of the note's subgraphs",
		},
		{
			name: "user with multiple subscriptions can see general knowledge",
			note: &model.NoteView{
				Title:         "General Knowledge Note",
				Free:          false,
				SubgraphNames: []string{},
			},
			userSubgraphs:  []string{"premium", "vip"},
			expectedAccess: true,
			description:    "users with multiple subscriptions should see general knowledge notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{
						ID:   123,
						Role: "user",
					}, nil
				},
				ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
					return tt.userSubgraphs, nil
				},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, tt.note)

			require.NoError(t, err)
			require.Equal(t, tt.expectedAccess, hasAccess, tt.description)
		})
	}
}

func TestResolve_RequireSigninSubgraph(t *testing.T) {
	tests := []struct {
		name           string
		note           *model.NoteView
		userToken      *usertoken.Data
		userSubgraphs  []string
		expectedAccess bool
		description    string
	}{
		{
			name: "authenticated user can read note in require_signin subgraph without subscription",
			note: &model.NoteView{
				Title:         "Signin-Gated Note",
				Free:          false,
				SubgraphNames: []string{"members"},
				Subgraphs: map[string]*model.NoteSubgraph{
					"members": {Name: "members", RequireSignin: true},
				},
			},
			userToken:      &usertoken.Data{ID: 123, Role: "user"},
			userSubgraphs:  []string{},
			expectedAccess: true,
			description:    "require_signin subgraph grants access to any authenticated user",
		},
		{
			name: "guest cannot read note in require_signin subgraph",
			note: &model.NoteView{
				Title:         "Signin-Gated Note",
				Free:          false,
				SubgraphNames: []string{"members"},
				Subgraphs: map[string]*model.NoteSubgraph{
					"members": {Name: "members", RequireSignin: true},
				},
			},
			userToken:      nil,
			expectedAccess: false,
			description:    "guests still can't read require_signin notes (sign-in wall handles them)",
		},
		{
			name: "note in mixed subgraphs: one require_signin grants access",
			note: &model.NoteView{
				Title:         "Mixed Subgraph Note",
				Free:          false,
				SubgraphNames: []string{"members", "premium"},
				Subgraphs: map[string]*model.NoteSubgraph{
					"members": {Name: "members", RequireSignin: true},
					"premium": {Name: "premium", RequireSignin: false},
				},
			},
			userToken:      &usertoken.Data{ID: 123, Role: "user"},
			userSubgraphs:  []string{},
			expectedAccess: true,
			description:    "any subgraph with require_signin grants access",
		},
		{
			name: "note in subgraph without require_signin still needs subscription",
			note: &model.NoteView{
				Title:         "Premium Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
				Subgraphs: map[string]*model.NoteSubgraph{
					"premium": {Name: "premium", RequireSignin: false},
				},
			},
			userToken:      &usertoken.Data{ID: 123, Role: "user"},
			userSubgraphs:  []string{},
			expectedAccess: false,
			description:    "non-require_signin subgraphs still need subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return tt.userToken, nil
				},
				ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
					return tt.userSubgraphs, nil
				},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, tt.note)

			require.NoError(t, err)
			require.Equal(t, tt.expectedAccess, hasAccess, tt.description)
		})
	}
}

func TestResolve_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func() *EnvMock
		expectedErr string
	}{
		{
			name: "error getting current user token",
			setupEnv: func() *EnvMock {
				return &EnvMock{
					CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return nil, errors.New("database error")
					},
				}
			},
			expectedErr: "failed to get current user token: database error",
		},
		{
			name: "error listing user subgraphs",
			setupEnv: func() *EnvMock {
				return &EnvMock{
					CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return &usertoken.Data{
							ID:   123,
							Role: "user",
						}, nil
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
						return nil, errors.New("subgraph service error")
					},
				}
			},
			expectedErr: "failed to list user subgraphs: subgraph service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			note := &model.NoteView{
				Title:         "Test Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, note)

			require.Error(t, err)
			require.False(t, hasAccess)
			require.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestResolve_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		note           *model.NoteView
		userToken      *usertoken.Data
		userSubgraphs  []string
		expectedAccess bool
		description    string
	}{
		{
			name: "free note with subgraphs should be accessible to guests",
			note: &model.NoteView{
				Title:         "Free Premium Content",
				Free:          true,
				SubgraphNames: []string{"premium"},
			},
			userToken:      nil,
			expectedAccess: true,
			description:    "free notes should be accessible regardless of subgraphs",
		},
		{
			name: "note with empty subgraph name",
			note: &model.NoteView{
				Title:         "Note with Empty Subgraph",
				Free:          false,
				SubgraphNames: []string{""},
			},
			userToken: &usertoken.Data{
				ID:   123,
				Role: "user",
			},
			userSubgraphs:  []string{"premium"},
			expectedAccess: false,
			description:    "empty subgraph names should not match any user subscriptions",
		},
		{
			name: "user with empty subgraph subscription",
			note: &model.NoteView{
				Title:         "Premium Note",
				Free:          false,
				SubgraphNames: []string{"premium"},
			},
			userToken: &usertoken.Data{
				ID:   123,
				Role: "user",
			},
			userSubgraphs:  []string{""},
			expectedAccess: false,
			description:    "empty user subscriptions should not match note subgraphs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return tt.userToken, nil
				},
				ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
					return tt.userSubgraphs, nil
				},
			}

			hasAccess, err := canreadnote.Resolve(context.Background(), env, tt.note)

			require.NoError(t, err)
			require.Equal(t, tt.expectedAccess, hasAccess, tt.description)
		})
	}
}

func TestResolve_SystemAndHTMLFiles(t *testing.T) {
	guestEnv := &EnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return nil, nil
		},
	}
	adminEnv := &EnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 1, Role: "admin"}, nil
		},
	}

	cases := []struct {
		name       string
		path       string
		free       bool
		env        *EnvMock
		wantAccess bool
	}{
		// HTML templates are always blocked for non-admins.
		{"html file blocked for guest", "page.html", true, guestEnv, false},
		{"html in system dir blocked for guest", "_layouts/base.html", true, guestEnv, false},
		{"admin sees html file", "_layouts/base.html", true, adminEnv, true},
		// System files follow normal access rules (free: true → accessible).
		{"free system file accessible for guest", "_footer.md", true, guestEnv, true},
		{"free system file accessible for admin", "_footer.md", true, adminEnv, true},
		{"non-free system file blocked for guest", "docs/_internal/note.md", false, guestEnv, false},
		// Regular md not affected.
		{"regular md file not affected", "blog/post.md", true, guestEnv, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			note := &model.NoteView{Path: c.path, Free: c.free}
			got, err := canreadnote.Resolve(context.Background(), c.env, note)
			require.NoError(t, err)
			require.Equal(t, c.wantAccess, got)
		})
	}
}
