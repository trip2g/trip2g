package canreadnote_test

import (
	"context"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/canreadnote"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func TestNewReaderMatchesResolve(t *testing.T) {
	signinSubgraph := &model.NoteSubgraph{Name: "members", RequireSignin: true}
	notes := []*model.NoteView{
		{Path: "free.md", Free: true},
		{Path: "free.html", Free: true},
		{Path: "general.md"},
		{Path: "premium.md", SubgraphNames: []string{"premium"}},
		{Path: "other.md", SubgraphNames: []string{"other"}},
		{Path: "members.md", SubgraphNames: []string{"members"}, Subgraphs: map[string]*model.NoteSubgraph{"members": signinSubgraph}},
	}

	scoped := appreq.NewContext(context.Background(), &appreq.Request{
		WebhookScoped:       true,
		WebhookReadPatterns: []string{"premium.md"},
	})

	tests := []struct {
		name      string
		ctx       context.Context
		token     *usertoken.Data
		subgraphs []string
		federated []string
		lookups   int
	}{
		{name: "guest", ctx: context.Background()},
		{name: "admin", ctx: context.Background(), token: &usertoken.Data{ID: 1, Role: "admin"}},
		{name: "subscriber", ctx: context.Background(), token: &usertoken.Data{ID: 2}, subgraphs: []string{"premium"}, lookups: 1},
		{name: "user without subscriptions", ctx: context.Background(), token: &usertoken.Data{ID: 3}, lookups: 1},
		{name: "scoped token", ctx: scoped},
		{name: "federated", ctx: context.Background(), federated: []string{"premium"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				CurrentUserTokenFunc: func(context.Context) (*usertoken.Data, error) {
					return tt.token, nil
				},
				ListActiveUserSubgraphsFunc: func(context.Context, int64) ([]string, error) {
					return tt.subgraphs, nil
				},
				CurrentFederatedScopeFunc: func(context.Context) ([]string, bool) {
					return tt.federated, tt.federated != nil
				},
			}

			var want []bool
			for _, note := range notes {
				ok, err := canreadnote.Resolve(tt.ctx, env, note)
				require.NoError(t, err)
				want = append(want, ok)
			}
			before := len(env.ListActiveUserSubgraphsCalls())

			read, err := canreadnote.NewReader(tt.ctx, env)
			require.NoError(t, err)

			var got []bool
			for _, note := range notes {
				ok, readErr := read(note)
				require.NoError(t, readErr)
				got = append(got, ok)
			}

			require.Equal(t, want, got)
			require.Equal(t, tt.lookups, len(env.ListActiveUserSubgraphsCalls())-before)
		})
	}
}
