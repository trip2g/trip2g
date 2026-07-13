package resolvewikilinks

import (
	"context"

	"trip2g/internal/appreq"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
}

func Resolve(ctx context.Context, env Env, filter model.ResolveWikilinksFilter) ([]model.WikilinkResolution, error) {
	nvs := env.LatestNoteViews()
	var source *appmodel.NoteView
	if nvs != nil {
		source = nvs.GetByPathID(filter.NotePathID)
	}

	results := make([]model.WikilinkResolution, len(filter.Links))
	for i, link := range filter.Links {
		res := model.WikilinkResolution{Link: link}
		if nvs != nil { //nolint:nestif // wikilink resolution requires nil-guard, target resolution, and read-pattern enforcement
			if target := nvs.ResolveWikilinkTarget(source, link); target != nil {
				// Enforce read_patterns: a scoped token must not learn the
				// existence or path of notes outside its scope.
				if targetAllowed(ctx, target.Path) {
					res.Path = &target.Path
					var url string
					if target.Slug != "" {
						url = target.PermalinkOriginal
					} else {
						url = target.Permalink
					}
					res.URL = &url
				}
			}
		}
		results[i] = res
	}
	return results, nil
}

// targetAllowed reports whether a resolved wikilink target at targetPath may be
// returned to the caller under the current request's scope. When the request is
// not scoped (admin/personal-token/api-key) the target is always allowed. Scoped
// requests require the target's filesystem path to match at least one
// read_pattern; an empty pattern list is fail-closed (denies all).
func targetAllowed(ctx context.Context, targetPath string) bool {
	if !appreq.Scoped(ctx) {
		return true
	}
	rp := appreq.WebhookReadPatterns(ctx)
	if len(rp) == 0 {
		return false // fail-closed
	}
	return webhookutil.MatchesAny(targetPath, rp)
}
