package listunreleasedchanges_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/listunreleasedchanges"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func makeNoteViews(views ...*appmodel.NoteView) *appmodel.NoteViews {
	nv := &appmodel.NoteViews{}
	nv.List = append(nv.List, views...)
	return nv
}

func liveNote(pathID, versionID int64, path, content string) db.AllLiveNotesRow {
	return db.AllLiveNotesRow{
		PathID:    pathID,
		VersionID: versionID,
		Path:      path,
		Content:   content,
	}
}

func latestNote(pathID, versionID int64, path, content string) *appmodel.NoteView {
	return &appmodel.NoteView{
		PathID:    pathID,
		VersionID: versionID,
		Path:      path,
		Content:   []byte(content),
		Title:     path,
	}
}

func TestResolve_NoLiveRelease_AllAdded(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return nil, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(latestNote(1, 10, "posts/a.md", "hello world"))
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, model.NoteChangeTypeAdded, changes[0].ChangeType)
	require.Equal(t, int64(10), *changes[0].LatestVersionID)
	require.Nil(t, changes[0].LiveVersionID)
	require.Nil(t, changes[0].OldContent)
	require.Equal(t, "hello world", *changes[0].NewContent)
}

func TestResolve_Identical_EmptyResult(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return []db.AllLiveNotesRow{liveNote(1, 10, "posts/a.md", "hello")}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(latestNote(1, 10, "posts/a.md", "hello"))
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestResolve_Modified(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return []db.AllLiveNotesRow{liveNote(1, 10, "posts/a.md", "old line\n")}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(latestNote(1, 11, "posts/a.md", "new line\n"))
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	ch := changes[0]
	require.Equal(t, model.NoteChangeTypeModified, ch.ChangeType)
	require.Equal(t, int64(10), *ch.LiveVersionID)
	require.Equal(t, int64(11), *ch.LatestVersionID)
	require.Equal(t, "old line\n", *ch.OldContent)
	require.Equal(t, "new line\n", *ch.NewContent)
}

func TestResolve_Removed(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return []db.AllLiveNotesRow{liveNote(1, 10, "posts/a.md", "old content")}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews()
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	ch := changes[0]
	require.Equal(t, model.NoteChangeTypeRemoved, ch.ChangeType)
	require.Equal(t, int64(10), *ch.LiveVersionID)
	require.Nil(t, ch.LatestVersionID)
	require.Equal(t, "old content", *ch.OldContent)
	require.Nil(t, ch.NewContent)
}

func TestResolve_GlobFilter(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return nil, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(
				latestNote(1, 10, "posts/a.md", "A"),
				latestNote(2, 20, "drafts/b.md", "B"),
			)
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"posts/**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "posts/a.md", changes[0].Path)
}
