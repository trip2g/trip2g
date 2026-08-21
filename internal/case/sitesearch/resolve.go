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

	results, merged, err := Retrieve(ctx, env, input.Query, useLatest, input.Rerank)
	if err != nil {
		return nil, err
	}

	// Filter results based on permissions
	conn := model.SearchConnection{}
	hiddenResults := []appmodel.SearchResult{}

	for _, res := range results {
		if res.NoteView != nil { //nolint:nestif // per-result auth checks require nil-guard, scope check, and read-pattern gate
			// Fail-closed: scoped shortapitoken → enforce read_patterns strictly.
			// Empty patterns + scoped = deny-all (not "no restriction").
			if appreq.Scoped(ctx) {
				rp := appreq.WebhookReadPatterns(ctx)
				if len(rp) == 0 || !webhookutil.MatchesAny(res.NoteView.Path, rp) {
					continue
				}
			}

			if res.NoteView.IsSystem() || res.NoteView.ExcludeSearch {
				continue
			}

			canRead, readErr := env.CanReadNote(ctx, res.NoteView)
			if readErr != nil {
				return nil, fmt.Errorf("failed to check CanReadNote: %w", readErr)
			}

			if canRead {
				conn.Nodes = append(conn.Nodes, res)
				continue
			}

			croppedResult := appmodel.SearchResult{
				HighlightedTitle:   res.HighlightedTitle,
				URL:                res.URL,
				HighlightedContent: []string{"Закрытый материал."},
			}

			hiddenResults = append(hiddenResults, croppedResult)
		}
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
