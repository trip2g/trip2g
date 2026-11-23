package listnotefieldhistory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/listnotefieldhistory"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

const notePath = "tasks/ship.md"

func at(hours int) time.Time {
	return time.Date(2026, 11, 23, 14, 2, 0, 0, time.UTC).Add(time.Duration(hours) * time.Hour)
}

func ptr[T any](v T) *T {
	return &v
}

func version(id int64, content string) db.ListNoteVersionsWithEditorByPathRow {
	return db.ListNoteVersionsWithEditorByPathRow{
		ID:        id,
		Version:   id,
		Content:   content,
		CreatedAt: at(int(id)),
	}
}

func byUser(row db.ListNoteVersionsWithEditorByPathRow, userID int64, email, client string) db.ListNoteVersionsWithEditorByPathRow {
	row.CreatedByUserID = &userID
	row.UserEmail = &email
	row.CreatedByClient = &client
	return row
}

func byAPIKey(row db.ListNoteVersionsWithEditorByPathRow, keyID int64, description string) db.ListNoteVersionsWithEditorByPathRow {
	row.CreatedByApiKeyID = &keyID
	row.ApiKeyDescription = &description
	return row
}

func frontmatter(versionID int64, data string) db.ListNoteVersionFrontmattersByPathRow {
	return db.ListNoteVersionFrontmattersByPathRow{VersionID: versionID, Data: data}
}

func newEnv(versions []db.ListNoteVersionsWithEditorByPathRow, frontmatters []db.ListNoteVersionFrontmattersByPathRow) *EnvMock {
	return &EnvMock{
		ListNoteVersionsWithEditorByPathFunc: func(_ context.Context, value string) ([]db.ListNoteVersionsWithEditorByPathRow, error) {
			if value != notePath {
				return nil, nil
			}
			return versions, nil
		},
		ListNoteVersionFrontmattersByPathFunc: func(_ context.Context, value string) ([]db.ListNoteVersionFrontmattersByPathRow, error) {
			if value != notePath {
				return nil, nil
			}
			return frontmatters, nil
		},
	}
}

func TestResolve(t *testing.T) {
	alexey := &model.AdminNoteVersionAuthor{
		Label:  "alexey@example.com · via web",
		UserID: ptr(int64(7)),
		Client: ptr("web"),
	}
	syncKey := &model.AdminNoteVersionAuthor{
		Label:    "Obsidian sync",
		APIKeyID: ptr(int64(3)),
	}

	versions := []db.ListNoteVersionsWithEditorByPathRow{
		byAPIKey(version(1, "---\nstatus: Planned\nowner: Anna\n---\nShip it\n"), 3, "Obsidian sync"),
		byUser(version(2, "---\nstatus: In progress\nowner: Anna\ndue: 2026-11-24\n---\nShip it\n"), 7, "alexey@example.com", "web"),
		byAPIKey(version(3, "---\nstatus: In progress\nowner: Anna\ndue: 2026-11-24\n---\nShip it today\n"), 3, "Obsidian sync"),
		byUser(version(4, "---\nstatus: Done\ndue: 2026-11-24\n---\nShipped\n"), 7, "alexey@example.com", "web"),
	}
	frontmatters := []db.ListNoteVersionFrontmattersByPathRow{
		frontmatter(1, `{"owner":"Anna","status":"Planned"}`),
		frontmatter(2, `{"due":"2026-11-24","owner":"Anna","status":"In progress"}`),
		frontmatter(3, `{"due":"2026-11-24","owner":"Anna","status":"In progress"}`),
		frontmatter(4, `{"due":"2026-11-24","status":"Done"}`),
	}

	created := model.AdminNoteFieldHistoryEntry{
		VersionID: 1,
		Version:   1,
		CreatedAt: at(1),
		Kind:      model.AdminNoteFieldHistoryKindCreated,
		Author:    syncKey,
		Changes: []model.AdminNoteFieldChange{
			{Key: "owner", To: "Anna"},
			{Key: "status", To: "Planned"},
		},
	}
	statusAndDue := model.AdminNoteFieldHistoryEntry{
		VersionID: 2,
		Version:   2,
		CreatedAt: at(2),
		Kind:      model.AdminNoteFieldHistoryKindChanged,
		Author:    alexey,
		Changes: []model.AdminNoteFieldChange{
			{Key: "due", To: "2026-11-24"},
			{Key: "status", From: "Planned", To: "In progress"},
		},
	}
	bodyOnly := model.AdminNoteFieldHistoryEntry{
		VersionID:   3,
		Version:     3,
		CreatedAt:   at(3),
		Kind:        model.AdminNoteFieldHistoryKindBodyOnly,
		Author:      syncKey,
		Changes:     []model.AdminNoteFieldChange{},
		BodyChanged: true,
	}
	doneWithoutOwner := model.AdminNoteFieldHistoryEntry{
		VersionID: 4,
		Version:   4,
		CreatedAt: at(4),
		Kind:      model.AdminNoteFieldHistoryKindChanged,
		Author:    alexey,
		Changes: []model.AdminNoteFieldChange{
			{Key: "owner", From: "Anna"},
			{Key: "status", From: "In progress", To: "Done"},
		},
		BodyChanged: true,
	}

	tests := []struct {
		name         string
		versions     []db.ListNoteVersionsWithEditorByPathRow
		frontmatters []db.ListNoteVersionFrontmattersByPathRow
		filter       listnotefieldhistory.Filter
		want         *listnotefieldhistory.Payload
	}{
		{
			name:         "field changes newest first, body-only versions skipped",
			versions:     versions,
			frontmatters: frontmatters,
			filter:       listnotefieldhistory.Filter{Path: notePath},
			want: &listnotefieldhistory.Payload{
				Nodes:      []model.AdminNoteFieldHistoryEntry{doneWithoutOwner, statusAndDue, created},
				TotalCount: 3,
			},
		},
		{
			name:         "body-only versions on request",
			versions:     versions,
			frontmatters: frontmatters,
			filter:       listnotefieldhistory.Filter{Path: notePath, IncludeBodyOnly: ptr(true)},
			want: &listnotefieldhistory.Payload{
				Nodes:      []model.AdminNoteFieldHistoryEntry{doneWithoutOwner, bodyOnly, statusAndDue, created},
				TotalCount: 4,
			},
		},
		{
			name:         "limit and offset page the entries",
			versions:     versions,
			frontmatters: frontmatters,
			filter:       listnotefieldhistory.Filter{Path: notePath, Limit: ptr(int32(1)), Offset: ptr(int32(1))},
			want: &listnotefieldhistory.Payload{
				Nodes:      []model.AdminNoteFieldHistoryEntry{statusAndDue},
				TotalCount: 3,
			},
		},
		{
			name:         "offset past the end",
			versions:     versions,
			frontmatters: frontmatters,
			filter:       listnotefieldhistory.Filter{Path: notePath, Offset: ptr(int32(10))},
			want: &listnotefieldhistory.Payload{
				Nodes:      []model.AdminNoteFieldHistoryEntry{},
				TotalCount: 3,
			},
		},
		{
			name: "versions without materialized frontmatter are unknown",
			versions: []db.ListNoteVersionsWithEditorByPathRow{
				version(1, "---\nstatus: Planned\n---\nShip it\n"),
				version(2, "---\nstatus: Done\n---\nShip it\n"),
				version(3, "---\nstatus: Done\nowner: Anna\n---\nShip it\n"),
			},
			frontmatters: []db.ListNoteVersionFrontmattersByPathRow{
				frontmatter(3, `{"owner":"Anna","status":"Done"}`),
			},
			filter: listnotefieldhistory.Filter{Path: notePath},
			want: &listnotefieldhistory.Payload{
				Nodes: []model.AdminNoteFieldHistoryEntry{
					{
						VersionID: 3,
						Version:   3,
						CreatedAt: at(3),
						Kind:      model.AdminNoteFieldHistoryKindUnknown,
						Author:    &model.AdminNoteVersionAuthor{},
						Changes:   []model.AdminNoteFieldChange{},
					},
					{
						VersionID: 2,
						Version:   2,
						CreatedAt: at(2),
						Kind:      model.AdminNoteFieldHistoryKindUnknown,
						Author:    &model.AdminNoteVersionAuthor{},
						Changes:   []model.AdminNoteFieldChange{},
					},
					{
						VersionID: 1,
						Version:   1,
						CreatedAt: at(1),
						Kind:      model.AdminNoteFieldHistoryKindUnknown,
						Author:    &model.AdminNoteVersionAuthor{},
						Changes:   []model.AdminNoteFieldChange{},
					},
				},
				TotalCount: 3,
			},
		},
		{
			name:     "unknown path",
			versions: versions,
			filter:   listnotefieldhistory.Filter{Path: "missing.md"},
			want: &listnotefieldhistory.Payload{
				Nodes:      []model.AdminNoteFieldHistoryEntry{},
				TotalCount: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(tt.versions, tt.frontmatters)

			got, err := listnotefieldhistory.Resolve(context.Background(), env, tt.filter)
			require.NoError(t, err)
			require.Equal(t, tt.want, got, pretty.Diff(tt.want, got))
		})
	}
}

func TestResolveFailsOnStorageError(t *testing.T) {
	env := newEnv(nil, nil)
	env.ListNoteVersionFrontmattersByPathFunc = func(_ context.Context, _ string) ([]db.ListNoteVersionFrontmattersByPathRow, error) {
		return nil, errors.New("disk I/O error")
	}

	_, err := listnotefieldhistory.Resolve(context.Background(), env, listnotefieldhistory.Filter{Path: notePath})
	require.ErrorContains(t, err, "disk I/O error")
}
