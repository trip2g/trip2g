package getpublicnote

import (
	"context"

	"trip2g/internal/appreq"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
	RenderNotePage(ctx context.Context, request rendernotepage.Request) (*rendernotepage.Response, error)
}

func Resolve(
	ctx context.Context,
	env Env,
	input model.NoteInput,
	userToken *usertoken.Data,
) (*model.PublicNote, error) {
	var path string
	if input.Path != nil {
		path = *input.Path
	}

	// fsPath is the filesystem path (e.g. "boards/task.md") used to match
	// read_patterns. It is distinct from `path` which is the URL path used
	// by rendernotepage. The two share one namespace so the same glob
	// patterns cover both reads and writes.
	var fsPath string

	if input.PathID != nil {
		latestViews := env.LatestNoteViews()
		for _, view := range latestViews.List {
			if view.PathID == *input.PathID {
				path = view.Permalink
				fsPath = view.Path
				break
			}
		}
	}

	// When the caller supplies input.Path (a URL), resolve the filesystem path
	// for the read-pattern check. The NoteViews.Map is keyed by Permalink, but
	// the value's Path field carries the filesystem path, ensuring scope globs
	// match the same namespace as write_patterns.
	if fsPath == "" && path != "" {
		fsPath = resolveFsPathFromPermalink(env.LatestNoteViews(), path)
	}

	// Enforce read_patterns using the filesystem path so scope globs (e.g.
	// "boards/**") match the same namespace as write_patterns.
	// Fail-closed: a scoped token with empty read_patterns denies all reads.
	if appreq.Scoped(ctx) {
		rp := appreq.WebhookReadPatterns(ctx)
		checkPath := fsPath
		if checkPath == "" {
			checkPath = path // fallback: URL path if filesystem path unknown
		}
		if len(rp) == 0 || !webhookutil.MatchesAny(checkPath, rp) {
			return nil, nil
		}
	}

	request := rendernotepage.Request{
		Path:      path,
		Referrer:  input.Referer,
		UserToken: userToken,
	}

	response, err := env.RenderNotePage(ctx, request)
	if err != nil {
		return nil, err
	}

	if response.Note == nil {
		return nil, nil
	}

	return model.ConvertNoteToPublic(response.Note), nil
}

// resolveFsPathFromPermalink looks up the filesystem path of a note given its
// permalink (URL path). Returns "" when the view is not found. The NoteViews.Map
// is keyed by Permalink, but the value's Path field carries the filesystem path.
func resolveFsPathFromPermalink(views *appmodel.NoteViews, permalink string) string {
	if views == nil || permalink == "" {
		return ""
	}
	if nv, ok := views.Map[permalink]; ok {
		return nv.Path
	}
	return ""
}
