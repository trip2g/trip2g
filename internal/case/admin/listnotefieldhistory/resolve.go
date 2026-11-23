// Package listnotefieldhistory lists, newest first, which frontmatter fields
// each version of a note changed and who pushed it.
package listnotefieldhistory

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg listnotefieldhistory_test . Env

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"trip2g/internal/db"
	"trip2g/internal/frontmatteredit"
	"trip2g/internal/graph/model"
	"trip2g/internal/templateviews"
)

const defaultLimit = 50

type Env interface {
	ListNoteVersionsWithEditorByPath(ctx context.Context, value string) ([]db.ListNoteVersionsWithEditorByPathRow, error)
	ListNoteVersionFrontmattersByPath(ctx context.Context, value string) ([]db.ListNoteVersionFrontmattersByPathRow, error)
}

type Filter = model.AdminNoteFieldHistoryFilter

type Payload = model.AdminNoteFieldHistory

type frontmatter map[string]any

func Resolve(ctx context.Context, env Env, filter Filter) (*Payload, error) {
	versions, err := env.ListNoteVersionsWithEditorByPath(ctx, filter.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to list note versions: %w", err)
	}

	frontmatters, err := loadFrontmatters(ctx, env, filter.Path)
	if err != nil {
		return nil, err
	}

	includeBodyOnly := filter.IncludeBodyOnly != nil && *filter.IncludeBodyOnly

	entries := make([]model.AdminNoteFieldHistoryEntry, 0, len(versions))
	for i, version := range versions {
		var previous *db.ListNoteVersionsWithEditorByPathRow
		if i > 0 {
			previous = &versions[i-1]
		}

		entry := buildEntry(version, previous, frontmatters)
		if entry.Kind == model.AdminNoteFieldHistoryKindBodyOnly && !includeBodyOnly {
			continue
		}
		entries = append(entries, entry)
	}
	slices.Reverse(entries)

	payload := Payload{
		Nodes:      page(entries, filter.Limit, filter.Offset),
		TotalCount: int32(len(entries)),
	}
	return &payload, nil
}

func loadFrontmatters(ctx context.Context, env Env, path string) (map[int64]frontmatter, error) {
	rows, err := env.ListNoteVersionFrontmattersByPath(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list note version frontmatters: %w", err)
	}

	frontmatters := make(map[int64]frontmatter, len(rows))
	for _, row := range rows {
		var data frontmatter
		err = json.Unmarshal([]byte(row.Data), &data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode frontmatter of note version %d: %w", row.VersionID, err)
		}
		frontmatters[row.VersionID] = data
	}
	return frontmatters, nil
}

func buildEntry(
	version db.ListNoteVersionsWithEditorByPathRow,
	previous *db.ListNoteVersionsWithEditorByPathRow,
	frontmatters map[int64]frontmatter,
) model.AdminNoteFieldHistoryEntry {
	entry := model.AdminNoteFieldHistoryEntry{
		VersionID: version.ID,
		Version:   int32(version.Version),
		CreatedAt: version.CreatedAt,
		Kind:      model.AdminNoteFieldHistoryKindUnknown,
		Author:    author(version),
		Changes:   []model.AdminNoteFieldChange{},
	}

	current, known := frontmatters[version.ID]

	if previous == nil {
		if known {
			entry.Kind = model.AdminNoteFieldHistoryKindCreated
			entry.Changes = diff(nil, current)
		}
		return entry
	}

	entry.BodyChanged = frontmatteredit.Body(version.Content) != frontmatteredit.Body(previous.Content)

	before, previousKnown := frontmatters[previous.ID]
	if !known || !previousKnown {
		return entry
	}

	entry.Changes = diff(before, current)
	entry.Kind = model.AdminNoteFieldHistoryKindChanged
	if len(entry.Changes) == 0 {
		entry.Kind = model.AdminNoteFieldHistoryKindBodyOnly
	}
	return entry
}

func diff(before, after frontmatter) []model.AdminNoteFieldChange {
	keys := make([]string, 0, len(before)+len(after))
	for key := range before {
		keys = append(keys, key)
	}
	for key := range after {
		_, seen := before[key]
		if !seen {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	changes := make([]model.AdminNoteFieldChange, 0, len(keys))
	for _, key := range keys {
		from, wasSet := before[key]
		to, isSet := after[key]
		if wasSet && isSet && reflect.DeepEqual(from, to) {
			continue
		}

		change := model.AdminNoteFieldChange{
			Key:  key,
			From: from,
			To:   to,
		}
		changes = append(changes, change)
	}
	return changes
}

func author(version db.ListNoteVersionsWithEditorByPathRow) *model.AdminNoteVersionAuthor {
	editor := templateviews.NoteEditor{}
	if version.UserEmail != nil {
		editor.UserEmail = *version.UserEmail
	}
	if version.ApiKeyDescription != nil {
		editor.APIKeyDescription = *version.ApiKeyDescription
	}
	if version.CreatedByClient != nil {
		editor.Client = *version.CreatedByClient
	}

	return &model.AdminNoteVersionAuthor{
		Label:    editor.Label(),
		UserID:   version.CreatedByUserID,
		APIKeyID: version.CreatedByApiKeyID,
		Client:   version.CreatedByClient,
	}
}

func page(entries []model.AdminNoteFieldHistoryEntry, limit, offset *int32) []model.AdminNoteFieldHistoryEntry {
	size := defaultLimit
	if limit != nil {
		size = max(int(*limit), 0)
	}
	start := 0
	if offset != nil {
		start = min(max(int(*offset), 0), len(entries))
	}
	end := min(start+size, len(entries))
	return entries[start:end]
}
