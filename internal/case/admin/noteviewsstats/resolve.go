package noteviewsstats

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg noteviewsstats_test . Env

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

// periodDays must match the window in the queries (date('now', '-29 days')).
const periodDays = 30

type Env interface {
	Now() time.Time
	NoteViewsPerDay(ctx context.Context) ([]db.NoteViewsPerDayRow, error)
	TopViewedNotes(ctx context.Context) ([]db.TopViewedNotesRow, error)
	TopNoteReaders(ctx context.Context) ([]db.TopNoteReadersRow, error)
	NoteViewsByCreatedVia(ctx context.Context) ([]db.NoteViewsByCreatedViaRow, error)
}

type Stats = model.AdminNoteViewsStats

func Resolve(ctx context.Context, env Env) (*Stats, error) {
	perDay, err := env.NoteViewsPerDay(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get note views per day: %w", err)
	}

	notes, err := env.TopViewedNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get top viewed notes: %w", err)
	}

	readers, err := env.TopNoteReaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get top note readers: %w", err)
	}

	sources, err := env.NoteViewsByCreatedVia(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get note views by created_via: %w", err)
	}

	stats := &Stats{
		Days:       fillDays(env.Now().UTC(), perDay),
		TopNotes:   make([]model.AdminNoteViewsNote, 0, len(notes)),
		TopReaders: make([]model.AdminNoteViewsReader, 0, len(readers)),
		BySource:   make([]model.AdminNoteViewsSource, 0, len(sources)),
	}

	for _, n := range notes {
		stats.TopNotes = append(stats.TopNotes, model.AdminNoteViewsNote{
			PathID:  n.PathID,
			Path:    n.Path,
			Views:   n.Views,
			Readers: n.Readers,
		})
	}

	for _, r := range readers {
		stats.TopReaders = append(stats.TopReaders, model.AdminNoteViewsReader{
			UserID: r.UserID,
			Email:  r.Email,
			Views:  r.Views,
		})
	}

	for _, s := range sources {
		stats.BySource = append(stats.BySource, model.AdminNoteViewsSource{
			CreatedVia: s.CreatedVia,
			Views:      s.Views,
			Readers:    s.Readers,
		})
	}

	return stats, nil
}

func fillDays(now time.Time, rows []db.NoteViewsPerDayRow) []model.AdminNoteViewsDay {
	byDay := make(map[string]int64, len(rows))
	for _, r := range rows {
		byDay[r.Day] = r.Views
	}

	days := make([]model.AdminNoteViewsDay, 0, periodDays)
	for i := periodDays - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format(time.DateOnly)
		days = append(days, model.AdminNoteViewsDay{Day: day, Views: byDay[day]})
	}

	return days
}
