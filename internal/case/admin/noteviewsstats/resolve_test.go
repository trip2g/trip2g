package noteviewsstats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/noteviewsstats"
	"trip2g/internal/db"
)

func email(s string) *string { return &s }

func TestResolve(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		mock    func() *EnvMock
		check   func(t *testing.T, got *noteviewsstats.Stats)
		wantErr string
	}{
		{
			name: "fills missing days with zero and keeps order",
			mock: func() *EnvMock {
				return &EnvMock{
					NowFunc: func() time.Time { return now },
					NoteViewsPerDayFunc: func(context.Context) ([]db.NoteViewsPerDayRow, error) {
						return []db.NoteViewsPerDayRow{
							{Day: "2026-10-15", Views: 3},
							{Day: "2026-11-13", Views: 7},
						}, nil
					},
					TopViewedNotesFunc:        func(context.Context) ([]db.TopViewedNotesRow, error) { return nil, nil },
					TopNoteReadersFunc:        func(context.Context) ([]db.TopNoteReadersRow, error) { return nil, nil },
					NoteViewsByCreatedViaFunc: func(context.Context) ([]db.NoteViewsByCreatedViaRow, error) { return nil, nil },
				}
			},
			check: func(t *testing.T, got *noteviewsstats.Stats) {
				require.Len(t, got.Days, 30)
				require.Equal(t, "2026-10-15", got.Days[0].Day)
				require.Equal(t, int64(3), got.Days[0].Views)
				require.Equal(t, "2026-10-16", got.Days[1].Day)
				require.Equal(t, int64(0), got.Days[1].Views)
				require.Equal(t, "2026-11-13", got.Days[29].Day)
				require.Equal(t, int64(7), got.Days[29].Views)
			},
		},
		{
			name: "maps rows",
			mock: func() *EnvMock {
				return &EnvMock{
					NowFunc: func() time.Time { return now },
					NoteViewsPerDayFunc: func(context.Context) ([]db.NoteViewsPerDayRow, error) {
						return nil, nil
					},
					TopViewedNotesFunc: func(context.Context) ([]db.TopViewedNotesRow, error) {
						return []db.TopViewedNotesRow{{PathID: 4, Path: "a.md", Views: 9, Readers: 2}}, nil
					},
					TopNoteReadersFunc: func(context.Context) ([]db.TopNoteReadersRow, error) {
						return []db.TopNoteReadersRow{{UserID: 8, Email: email("x@y.z"), Views: 5}, {UserID: 9, Views: 1}}, nil
					},
					NoteViewsByCreatedViaFunc: func(context.Context) ([]db.NoteViewsByCreatedViaRow, error) {
						return []db.NoteViewsByCreatedViaRow{{CreatedVia: "email", Views: 10, Readers: 3}}, nil
					},
				}
			},
			check: func(t *testing.T, got *noteviewsstats.Stats) {
				require.Len(t, got.TopNotes, 1)
				require.Equal(t, int64(4), got.TopNotes[0].PathID)
				require.Equal(t, "a.md", got.TopNotes[0].Path)
				require.Equal(t, int64(9), got.TopNotes[0].Views)
				require.Equal(t, int64(2), got.TopNotes[0].Readers)
				require.Len(t, got.TopReaders, 2)
				require.Equal(t, "x@y.z", *got.TopReaders[0].Email)
				require.Nil(t, got.TopReaders[1].Email)
				require.Equal(t, "email", got.BySource[0].CreatedVia)
				require.Equal(t, int64(3), got.BySource[0].Readers)
			},
		},
		{
			name: "query error is returned",
			mock: func() *EnvMock {
				return &EnvMock{
					NowFunc: func() time.Time { return now },
					NoteViewsPerDayFunc: func(context.Context) ([]db.NoteViewsPerDayRow, error) {
						return nil, errors.New("boom")
					},
				}
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := noteviewsstats.Resolve(context.Background(), tt.mock())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, got)
		})
	}
}
