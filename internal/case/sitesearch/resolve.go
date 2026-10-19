package sitesearch

import (
	"context"
	"fmt"

	"trip2g/internal/appreq"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"

	appmodel "trip2g/internal/model"
)

type Env interface {
	RetrieveEnv
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error)
	SiteConfig(ctx context.Context) appmodel.SiteConfig
	SearchHideUnreadable() bool
}

// hybridResultCap bounds the final hybrid result list when no reranker OutputK
// applies. It is a final-output bound, so it must run AFTER permission
// filtering — capping the fused list earlier would discard readable results
// ranked below unreadable ones.
const hybridResultCap = 20

func Resolve(ctx context.Context, env Env, input model.SearchInput) (*model.SearchConnection, error) {
	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	siteConfig := env.SiteConfig(ctx)

	useLatest := siteConfig.ShowDraftVersions || userToken.IsAdmin()

	hideUnreadable := env.SearchHideUnreadable()
	visible := visibleFilter(ctx, env)

	req := Request{
		Query:     input.Query,
		UseLatest: useLatest,
		Rerank:    input.Rerank,
	}
	if hideUnreadable {
		req.Keep = visible
	}

	results, merged, err := Retrieve(ctx, env, req)
	if err != nil {
		return nil, err
	}

	// Filter results based on permissions
	conn := model.SearchConnection{}
	hiddenResults := []appmodel.SearchResult{}

	for _, res := range results {
		if res.NoteView == nil || !searchable(ctx, res.NoteView) {
			continue
		}

		canRead, readErr := visible(res.NoteView)
		if readErr != nil {
			return nil, readErr
		}

		if canRead {
			conn.Nodes = append(conn.Nodes, res)
			continue
		}

		if hideUnreadable {
			continue
		}

		croppedResult := appmodel.SearchResult{
			HighlightedTitle:   res.HighlightedTitle,
			URL:                res.URL,
			HighlightedContent: []string{"Закрытый материал."},
		}

		hiddenResults = append(hiddenResults, croppedResult)
	}

	// Push hidden results to the end of the list
	conn.Nodes = append(conn.Nodes, hiddenResults...)

	// Size bounds are applied after permission filtering so unreadable notes
	// don't consume output slots (hidden placeholders sit at the end and get cut
	// first). OutputK takes precedence; without a reranker the hybrid path is
	// bounded by hybridResultCap. Text-only search stays unbounded, as before.
	switch cfg := env.Features().VectorSearch.Reranker; {
	case cfg.Enabled && cfg.OutputK > 0:
		if len(conn.Nodes) > cfg.OutputK {
			conn.Nodes = conn.Nodes[:cfg.OutputK]
		}
	case merged:
		if len(conn.Nodes) > hybridResultCap {
			conn.Nodes = conn.Nodes[:hybridResultCap]
		}
	}

	return &conn, nil
}

// visibleFilter reports whether a note may appear in this request's results
// as a readable hit. Answers are memoized per note: the lanes and the final
// pass ask about the same notes.
func visibleFilter(ctx context.Context, env Env) func(*appmodel.NoteView) (bool, error) {
	answers := map[*appmodel.NoteView]bool{}
	return func(note *appmodel.NoteView) (bool, error) {
		if note == nil || !searchable(ctx, note) {
			return false, nil
		}

		answer, ok := answers[note]
		if ok {
			return answer, nil
		}

		canRead, err := env.CanReadNote(ctx, note)
		if err != nil {
			return false, fmt.Errorf("failed to check CanReadNote: %w", err)
		}

		answers[note] = canRead
		return canRead, nil
	}
}

func searchable(ctx context.Context, note *appmodel.NoteView) bool {
	return !outOfScope(ctx, note.Path) && !note.IsSystem() && !note.ExcludeSearch
}

// outOfScope fails closed: a scoped shortapitoken enforces read_patterns
// strictly, and empty patterns under a scope deny everything.
func outOfScope(ctx context.Context, path string) bool {
	if !appreq.Scoped(ctx) {
		return false
	}

	rp := appreq.WebhookReadPatterns(ctx)
	return len(rp) == 0 || !webhookutil.MatchesAny(path, rp)
}
