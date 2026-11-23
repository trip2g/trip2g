package listlatestnoteassets_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/listlatestnoteassets"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

func row(versionID, id int64, absolutePath string, createdAt time.Time) db.AllLatestNoteAssetsRow {
	return db.AllLatestNoteAssetsRow{
		VersionID: versionID,
		Path:      absolutePath,
		NoteAsset: db.NoteAsset{ID: id, AbsolutePath: absolutePath, CreatedAt: createdAt},
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestResolve(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []db.AllLatestNoteAssetsRow{
		row(10, 1, "/uploads/Photo.png", t0),
		row(11, 1, "/uploads/Photo.png", t0),
		row(10, 2, "/uploads/report.pdf", t0.Add(time.Hour)),
		row(12, 3, "/blog/cover.jpg", t0.Add(2*time.Hour)),
		row(12, 4, "/blog/photo-2.png", t0.Add(2*time.Hour)),
	}

	tests := []struct {
		name      string
		filter    *model.AdminLatestNoteAssetsFilter
		wantIDs   []int64
		wantTotal int32
	}{
		{
			name:      "no filter lists each asset once, newest first",
			wantIDs:   []int64{4, 3, 2, 1},
			wantTotal: 4,
		},
		{
			name:      "path prefix",
			filter:    &model.AdminLatestNoteAssetsFilter{PathPrefix: ptr("/uploads/")},
			wantIDs:   []int64{2, 1},
			wantTotal: 2,
		},
		{
			name:      "search ignores case",
			filter:    &model.AdminLatestNoteAssetsFilter{Search: ptr("PHOTO")},
			wantIDs:   []int64{4, 1},
			wantTotal: 2,
		},
		{
			name:      "prefix and search together",
			filter:    &model.AdminLatestNoteAssetsFilter{PathPrefix: ptr("/blog/"), Search: ptr("photo")},
			wantIDs:   []int64{4},
			wantTotal: 1,
		},
		{
			name:      "limit and offset page the filtered list, total counts all of it",
			filter:    &model.AdminLatestNoteAssetsFilter{Limit: ptr[int32](2), Offset: ptr[int32](1)},
			wantIDs:   []int64{3, 2},
			wantTotal: 4,
		},
		{
			name:      "offset past the end",
			filter:    &model.AdminLatestNoteAssetsFilter{Offset: ptr[int32](10)},
			wantIDs:   []int64{},
			wantTotal: 4,
		},
		{
			name:      "limit is capped",
			filter:    &model.AdminLatestNoteAssetsFilter{Limit: ptr[int32](100000)},
			wantIDs:   []int64{4, 3, 2, 1},
			wantTotal: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				AllLatestNoteAssetsFunc: func(ctx context.Context) ([]db.AllLatestNoteAssetsRow, error) {
					return rows, nil
				},
			}

			got, err := listlatestnoteassets.Resolve(context.Background(), env, tt.filter)
			require.NoError(t, err)

			ids := make([]int64, 0, len(got.Nodes))
			for _, asset := range got.Nodes {
				ids = append(ids, asset.ID)
			}

			require.Equal(t, tt.wantIDs, ids)
			require.Equal(t, tt.wantTotal, got.TotalCount)
		})
	}
}

func TestResolveNegativeLimitAndOffset(t *testing.T) {
	env := &EnvMock{
		AllLatestNoteAssetsFunc: func(ctx context.Context) ([]db.AllLatestNoteAssetsRow, error) {
			return []db.AllLatestNoteAssetsRow{row(1, 1, "/a.png", time.Time{})}, nil
		},
	}

	filter := model.AdminLatestNoteAssetsFilter{Limit: ptr[int32](-1), Offset: ptr[int32](-5)}

	got, err := listlatestnoteassets.Resolve(context.Background(), env, &filter)
	require.NoError(t, err)
	require.Empty(t, got.Nodes)
	require.Equal(t, int32(1), got.TotalCount)
}
