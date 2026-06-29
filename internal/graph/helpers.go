package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

func resolveOne[T any, K any](
	ctx context.Context,
	id K,
	fetch func(context.Context, K) (T, error),
) (*T, error) {
	row, err := fetch(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func resolveOnePtr[T any, K any](
	ctx context.Context,
	id *K,
	fetch func(context.Context, K) (T, error),
) (*T, error) {
	if id == nil {
		return nil, nil
	}

	return resolveOne(ctx, *id, fetch)
}

var errUnauthorized = errors.New("unauthorized")

func checkAdmin(ctx context.Context) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return err
	}

	token, err := req.UserToken()
	if err != nil {
		return err
	}

	if !token.IsAdmin() {
		return errUnauthorized
	}

	return nil
}

// filterNotePathsByScope removes any db.NotePath whose Value (filesystem path)
// does not match the read_patterns stamped on the request. When the request is
// not scoped (admin/personal-token/api-key) the slice is returned unchanged.
//
// Scoped requests with empty read_patterns are fail-closed: all paths are
// denied. Non-empty read_patterns require a glob match against the filesystem
// path (e.g. "boards/sprint.md"), the same namespace used by write_patterns.
func filterNotePathsByScope(ctx context.Context, paths []db.NotePath) []db.NotePath {
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

// wikilinkTargetAllowed reports whether a resolved wikilink target at
// targetPath may be returned to the caller under the current request's scope.
// When the request is not scoped (admin/personal-token/api-key) the target is
// always allowed. Scoped requests require the target's filesystem path to match
// at least one read_pattern; an empty pattern list is fail-closed (denies all).
func wikilinkTargetAllowed(ctx context.Context, targetPath string) bool {
	if !appreq.Scoped(ctx) {
		return true
	}
	rp := appreq.WebhookReadPatterns(ctx)
	if len(rp) == 0 {
		return false // fail-closed
	}
	return webhookutil.MatchesAny(targetPath, rp)
}

// resolveFsPathFromPermalink looks up the filesystem path of a note given its
// permalink (URL path). Returns "" when the view is not found.
//
// Extracted from the Note resolver to make the fsPath lookup independently
// testable. The scope check in the Note resolver uses the filesystem path so
// that read_patterns globs ("boards/**") match the same namespace as
// write_patterns — the NoteViews.Map is keyed by Permalink, but the value's
// Path field carries the filesystem path.
func resolveFsPathFromPermalink(views *model.NoteViews, permalink string) string {
	if views == nil || permalink == "" {
		return ""
	}
	if nv, ok := views.Map[permalink]; ok {
		return nv.Path
	}
	return ""
}

func (r *queryResolver) convertSearchResultsToNotePath(ctx context.Context, nodes []model.SearchResult) ([]db.NotePath, error) {
	res := []db.NotePath{}

	for _, result := range nodes {
		if result.NoteView != nil {
			pathID := result.NoteView.PathID

			notePath, selectErr := r.env(ctx).NotePathByID(ctx, pathID)
			if selectErr != nil {
				return nil, fmt.Errorf("failed to get note path by ID %d: %w", pathID, selectErr)
			}

			res = append(res, notePath)
		}
	}

	return res, nil
}
