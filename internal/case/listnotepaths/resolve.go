package listnotepaths

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type Env interface {
	ListNotePathsByValues(ctx context.Context, paths []string) ([]db.NotePath, error)
	ListNotePathsLike(ctx context.Context, value string) ([]db.NotePath, error)
	AllVisibleNotePaths(ctx context.Context) ([]db.NotePath, error)
	NotePathByID(ctx context.Context, id int64) (db.NotePath, error)
	SearchNotes(ctx context.Context, input model.SearchInput) (*model.SearchConnection, error)
}

func Resolve(ctx context.Context, env Env, filter *model.NotePathsFilter) ([]db.NotePath, error) {
	if filter != nil && len(filter.Paths) > 0 {
		paths, err := env.ListNotePathsByValues(ctx, filter.Paths)
		if err != nil {
			return nil, err
		}
		return filterByScope(ctx, paths), nil
	}

	if filter != nil && filter.Search != nil {
		conn, err := env.SearchNotes(ctx, model.SearchInput{Query: *filter.Search})
		if err != nil {
			return nil, err
		}

		// Search already filters by read_patterns (sitesearch enforces scope);
		// convertSearchResults only wraps the results.
		return convertSearchResults(ctx, env, conn.Nodes)
	}

	if filter != nil && filter.Like != nil {
		pattern := *filter.Like

		// Prevent potential DoS attacks with excessive wildcards
		if strings.Count(pattern, "%") > 5 || strings.Count(pattern, "_") > 10 {
			return nil, errors.New("too many wildcard characters in pattern")
		}

		paths, err := env.ListNotePathsLike(ctx, pattern)
		if err != nil {
			return nil, err
		}
		return filterByScope(ctx, paths), nil
	}

	paths, err := env.AllVisibleNotePaths(ctx)
	if err != nil {
		return nil, err
	}
	return filterByScope(ctx, paths), nil
}

// filterByScope removes any db.NotePath whose Value (filesystem path) does not
// match the read_patterns stamped on the request. When the request is not scoped
// (admin/personal-token/api-key) the slice is returned unchanged.
//
// Scoped requests with empty read_patterns are fail-closed: all paths are
// denied. Non-empty read_patterns require a glob match against the filesystem
// path (e.g. "boards/sprint.md"), the same namespace used by write_patterns.
func filterByScope(ctx context.Context, paths []db.NotePath) []db.NotePath {
	if !appreq.Scoped(ctx) {
		return paths
	}
	rp := appreq.WebhookReadPatterns(ctx)
	// Fail-closed: scoped token with empty read_patterns denies all reads.
	if len(rp) == 0 {
		return nil
	}
	out := paths[:0:0] // nil-safe empty slice sharing no backing array
	for _, p := range paths {
		if webhookutil.MatchesAny(p.Value, rp) {
			out = append(out, p)
		}
	}
	return out
}

func convertSearchResults(ctx context.Context, env Env, nodes []appmodel.SearchResult) ([]db.NotePath, error) {
	res := []db.NotePath{}

	for _, result := range nodes {
		if result.NoteView != nil {
			pathID := result.NoteView.PathID

			notePath, err := env.NotePathByID(ctx, pathID)
			if err != nil {
				return nil, fmt.Errorf("failed to get note path by ID %d: %w", pathID, err)
			}

			res = append(res, notePath)
		}
	}

	return res, nil
}
