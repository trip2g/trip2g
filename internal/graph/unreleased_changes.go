package graph

import (
	"context"

	"trip2g/internal/case/checkapikey"
	"trip2g/internal/case/listunreleasedchanges"
	"trip2g/internal/graph/model"
)

func (r *queryResolver) UnreleasedChanges(ctx context.Context, filter model.NoteChangesFilter) (*model.UnreleasedChangesConnection, error) {
	if _, err := checkapikey.Resolve(ctx, r.env(ctx), "unreleased_changes"); err != nil {
		return nil, err
	}
	changes, err := listunreleasedchanges.Resolve(ctx, r.env(ctx), filter)
	if err != nil {
		return nil, err
	}
	return &model.UnreleasedChangesConnection{
		TotalCount: len(changes),
		Nodes:      changes,
	}, nil
}

func (r *unreleasedChangeResolver) Stats(_ context.Context, obj *model.UnreleasedChange) (*model.UnreleasedChangeStats, error) {
	d := obj.Diff()
	return &model.UnreleasedChangeStats{
		AddedLines:   int32(d.AddedLines),
		RemovedLines: int32(d.RemovedLines),
		ChangedWords: int32(d.ChangedWords),
	}, nil
}

func (r *unreleasedChangeResolver) UnifiedDiff(_ context.Context, obj *model.UnreleasedChange) (string, error) {
	return obj.Diff().Unified, nil
}

func (r *unreleasedChangeResolver) WordDiff(_ context.Context, obj *model.UnreleasedChange) (string, error) {
	return obj.Diff().Word, nil
}

func (r *unreleasedChangesConnectionResolver) TotalCount(_ context.Context, obj *model.UnreleasedChangesConnection) (int32, error) {
	return int32(obj.TotalCount), nil
}

func (r *unreleasedChangesConnectionResolver) TotalStats(_ context.Context, obj *model.UnreleasedChangesConnection) (*model.UnreleasedChangeStats, error) {
	var s model.UnreleasedChangeStats
	for _, ch := range obj.Nodes {
		d := ch.Diff()
		s.AddedLines += int32(d.AddedLines)
		s.RemovedLines += int32(d.RemovedLines)
		s.ChangedWords += int32(d.ChangedWords)
	}
	return &s, nil
}
