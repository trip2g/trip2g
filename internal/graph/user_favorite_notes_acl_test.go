package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	gmodel "trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type favoriteNotesEnv struct {
	Env
	nvs       *appmodel.NoteViews
	favorites []db.ListUserFavoriteNotesRow
	token     *usertoken.Data
	subgraphs []string
}

func (e *favoriteNotesEnv) ListUserFavoriteNotes(context.Context, int64) ([]db.ListUserFavoriteNotesRow, error) {
	return e.favorites, nil
}

func (e *favoriteNotesEnv) LiveNoteViews() *appmodel.NoteViews {
	return e.nvs
}

func (e *favoriteNotesEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return e.token, nil
}

func (e *favoriteNotesEnv) ListActiveUserSubgraphs(context.Context, int64) ([]string, error) {
	return e.subgraphs, nil
}

func (e *favoriteNotesEnv) CurrentFederatedScope(context.Context) ([]string, bool) {
	return nil, false
}

func favoriteNotesCtx(env Env, readPatterns []string) context.Context {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")
	req := &appreq.Request{Req: fctx, Env: env}
	if readPatterns != nil {
		req.WebhookScoped = true
		req.WebhookReadPatterns = readPatterns
	}
	req.StoreInContext()
	return fctx
}

func favoriteNotesFixture() (*appmodel.NoteViews, []db.ListUserFavoriteNotesRow) {
	free := aclNote(1, "free.md", "/free", "a")
	free.Free = true
	granted := aclNote(2, "a/granted.md", "/a-granted", "a")
	revoked := aclNote(3, "b/revoked.md", "/b-revoked", "b")
	paid := aclNote(4, "paid/lesson.md", "/paid-lesson", "paid")

	favorites := []db.ListUserFavoriteNotesRow{
		{PathID: 1}, {PathID: 2}, {PathID: 3}, {PathID: 4}, {PathID: 99},
	}

	return aclNvs(free, granted, revoked, paid), favorites
}

func favoritePaths(notes []gmodel.PublicNote) []string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		out = append(out, n.Path)
	}
	return out
}

func TestUserFavoriteNotesRespectReadAccess(t *testing.T) {
	tests := []struct {
		name         string
		token        *usertoken.Data
		subgraphs    []string
		readPatterns []string
		want         []string
	}{
		{
			name:      "user drops revoked subgraph and unpaid note",
			token:     &usertoken.Data{ID: 5, Role: "user"},
			subgraphs: []string{"a"},
			want:      []string{"/free", "/a-granted"},
		},
		{
			name:      "user with every subgraph sees all",
			token:     &usertoken.Data{ID: 5, Role: "user"},
			subgraphs: []string{"a", "b", "paid"},
			want:      []string{"/free", "/a-granted", "/b-revoked", "/paid-lesson"},
		},
		{
			name:  "admin sees all",
			token: &usertoken.Data{ID: 1, Role: "admin"},
			want:  []string{"/free", "/a-granted", "/b-revoked", "/paid-lesson"},
		},
		{
			name:         "scoped token sees only its read patterns",
			token:        &usertoken.Data{ID: 5, Role: "user"},
			subgraphs:    []string{"a", "b", "paid"},
			readPatterns: []string{"a/**"},
			want:         []string{"/a-granted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nvs, favorites := favoriteNotesFixture()
			env := &favoriteNotesEnv{
				nvs:       nvs,
				favorites: favorites,
				token:     tt.token,
				subgraphs: tt.subgraphs,
			}
			r := &userResolver{&Resolver{DefaultEnv: env}}

			got, err := r.FavoriteNotes(favoriteNotesCtx(env, tt.readPatterns), &db.User{ID: 5})

			require.NoError(t, err)
			require.Equal(t, tt.want, favoritePaths(got))
		})
	}
}
