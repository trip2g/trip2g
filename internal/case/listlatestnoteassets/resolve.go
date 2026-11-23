// Package listlatestnoteassets lists the assets of the latest note versions,
// each asset once, filtered and paged for admin.allLatestNoteAssets.
package listlatestnoteassets

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg listlatestnoteassets_test . Env

type Env interface {
	AllLatestNoteAssets(ctx context.Context) ([]db.AllLatestNoteAssetsRow, error)
}

const maxLimit = 500

func Resolve(ctx context.Context, env Env, filter *model.AdminLatestNoteAssetsFilter) (*model.AdminLatestNoteAssetsConnection, error) {
	rows, err := env.AllLatestNoteAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest note assets: %w", err)
	}

	if filter == nil {
		filter = &model.AdminLatestNoteAssetsFilter{}
	}

	assets := matching(rows, filter)

	slices.SortFunc(assets, func(a, b db.NoteAsset) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.ID, a.ID))
	})

	connection := model.AdminLatestNoteAssetsConnection{
		Nodes:      page(assets, filter),
		TotalCount: int32(len(assets)),
	}

	return &connection, nil
}

func matching(rows []db.AllLatestNoteAssetsRow, filter *model.AdminLatestNoteAssetsFilter) []db.NoteAsset {
	prefix := ""
	if filter.PathPrefix != nil {
		prefix = *filter.PathPrefix
	}

	search := ""
	if filter.Search != nil {
		search = strings.ToLower(*filter.Search)
	}

	seen := make(map[int64]struct{}, len(rows))
	assets := make([]db.NoteAsset, 0, len(rows))

	for _, row := range rows {
		asset := row.NoteAsset

		_, duplicate := seen[asset.ID]
		if duplicate {
			continue
		}

		if !strings.HasPrefix(asset.AbsolutePath, prefix) {
			continue
		}

		if !strings.Contains(strings.ToLower(asset.AbsolutePath), search) {
			continue
		}

		seen[asset.ID] = struct{}{}
		assets = append(assets, asset)
	}

	return assets
}

func page(assets []db.NoteAsset, filter *model.AdminLatestNoteAssetsFilter) []db.NoteAsset {
	offset := 0
	if filter.Offset != nil {
		offset = int(max(*filter.Offset, 0))
	}

	start := min(offset, len(assets))
	end := len(assets)

	if filter.Limit != nil {
		limit := int(min(max(*filter.Limit, 0), maxLimit))
		end = min(start+limit, len(assets))
	}

	return assets[start:end]
}
