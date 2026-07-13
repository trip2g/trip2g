package listnotepaths_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/listnotepaths"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func ptr[T any](v T) *T { return &v }

func scopedCtx(readPatterns []string) context.Context {
	return appreq.NewContext(context.Background(), &appreq.Request{
		WebhookScoped:       true,
		WebhookReadPatterns: readPatterns,
	})
}

func np(value string) db.NotePath { return db.NotePath{Value: value} }

func TestResolve_PathsBranch(t *testing.T) {
	env := &listnotepaths.EnvMock{
		ListNotePathsByValuesFunc: func(ctx context.Context, paths []string) ([]db.NotePath, error) {
			require.Equal(t, []string{"a.md"}, paths)
			return []db.NotePath{np("a.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(context.Background(), env, &model.NotePathsFilter{Paths: []string{"a.md"}})
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestResolve_LikeBranch(t *testing.T) {
	env := &listnotepaths.EnvMock{
		ListNotePathsLikeFunc: func(ctx context.Context, value string) ([]db.NotePath, error) {
			require.Equal(t, "posts/%", value)
			return []db.NotePath{np("posts/x.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(context.Background(), env, &model.NotePathsFilter{Like: ptr("posts/%")})
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestResolve_AllBranch(t *testing.T) {
	env := &listnotepaths.EnvMock{
		AllVisibleNotePathsFunc: func(ctx context.Context) ([]db.NotePath, error) {
			return []db.NotePath{np("a.md"), np("b.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(context.Background(), env, nil)
	require.NoError(t, err)
	require.Len(t, out, 2)
}

func TestResolve_SearchBranch(t *testing.T) {
	env := &listnotepaths.EnvMock{
		SearchNotesFunc: func(ctx context.Context, input model.SearchInput) (*model.SearchConnection, error) {
			require.Equal(t, "hello", input.Query)
			return &model.SearchConnection{Nodes: []appmodel.SearchResult{
				{NoteView: &appmodel.NoteView{PathID: 7}},
				{NoteView: nil}, // nil NoteView skipped
			}}, nil
		},
		NotePathByIDFunc: func(ctx context.Context, id int64) (db.NotePath, error) {
			require.Equal(t, int64(7), id)
			return np("found.md"), nil
		},
	}
	out, err := listnotepaths.Resolve(context.Background(), env, &model.NotePathsFilter{Search: ptr("hello")})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "found.md", out[0].Value)
}

func TestResolve_WildcardDoSLimits(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"exactly 5 percent ok", strings.Repeat("%", 5), false},
		{"6 percent rejected", strings.Repeat("%", 6), true},
		{"exactly 10 underscore ok", strings.Repeat("_", 10), false},
		{"11 underscore rejected", strings.Repeat("_", 11), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &listnotepaths.EnvMock{
				ListNotePathsLikeFunc: func(ctx context.Context, value string) ([]db.NotePath, error) {
					return nil, nil
				},
			}
			_, err := listnotepaths.Resolve(context.Background(), env, &model.NotePathsFilter{Like: ptr(tt.pattern)})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolve_ScopeFilterAppliedOnPaths(t *testing.T) {
	env := &listnotepaths.EnvMock{
		ListNotePathsByValuesFunc: func(ctx context.Context, paths []string) ([]db.NotePath, error) {
			return []db.NotePath{np("boards/a.md"), np("posts/b.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(scopedCtx([]string{"boards/**"}), env, &model.NotePathsFilter{Paths: []string{"boards/a.md", "posts/b.md"}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "boards/a.md", out[0].Value)
}

func TestResolve_ScopeFilterAppliedOnLike(t *testing.T) {
	env := &listnotepaths.EnvMock{
		ListNotePathsLikeFunc: func(ctx context.Context, value string) ([]db.NotePath, error) {
			return []db.NotePath{np("boards/a.md"), np("posts/b.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(scopedCtx([]string{"boards/**"}), env, &model.NotePathsFilter{Like: ptr("%")})
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestResolve_ScopeFilterAppliedOnAll(t *testing.T) {
	env := &listnotepaths.EnvMock{
		AllVisibleNotePathsFunc: func(ctx context.Context) ([]db.NotePath, error) {
			return []db.NotePath{np("boards/a.md"), np("posts/b.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(scopedCtx([]string{"boards/**"}), env, nil)
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestResolve_ScopeEmptyPatternsFailClosed(t *testing.T) {
	env := &listnotepaths.EnvMock{
		AllVisibleNotePathsFunc: func(ctx context.Context) ([]db.NotePath, error) {
			return []db.NotePath{np("boards/a.md")}, nil
		},
	}
	out, err := listnotepaths.Resolve(scopedCtx(nil), env, nil)
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestResolve_ScopeNotAppliedToSearch(t *testing.T) {
	// Search results are already scope-filtered by sitesearch; the case must
	// not re-filter them.
	env := &listnotepaths.EnvMock{
		SearchNotesFunc: func(ctx context.Context, input model.SearchInput) (*model.SearchConnection, error) {
			return &model.SearchConnection{Nodes: []appmodel.SearchResult{
				{NoteView: &appmodel.NoteView{PathID: 1}},
			}}, nil
		},
		NotePathByIDFunc: func(ctx context.Context, id int64) (db.NotePath, error) {
			return np("posts/out-of-scope.md"), nil
		},
	}
	out, err := listnotepaths.Resolve(scopedCtx([]string{"boards/**"}), env, &model.NotePathsFilter{Search: ptr("q")})
	require.NoError(t, err)
	require.Len(t, out, 1) // not filtered out despite scope mismatch
}

func TestResolve_ListError(t *testing.T) {
	env := &listnotepaths.EnvMock{
		AllVisibleNotePathsFunc: func(ctx context.Context) ([]db.NotePath, error) {
			return nil, errors.New("db down")
		},
	}
	_, err := listnotepaths.Resolve(context.Background(), env, nil)
	require.Error(t, err)
}
