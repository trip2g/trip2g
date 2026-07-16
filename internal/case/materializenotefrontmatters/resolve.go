// Package materializenotefrontmatters persists the effective frontmatter of
// loaded note versions and maintains the set of keys observed in latest/live
// notes.
package materializenotefrontmatters

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/yamlutil"
)

type Env interface {
	UpsertNoteVersionFrontmatter(context.Context, db.UpsertNoteVersionFrontmatterParams) error
	DeleteNoteVersionFrontmatterKeys(context.Context, int64) error
	UpsertNoteVersionFrontmatterKey(context.Context, db.UpsertNoteVersionFrontmatterKeyParams) error
	InsertNoteVersionFrontmatterKey(context.Context, db.InsertNoteVersionFrontmatterKeyParams) error
	RefreshNoteVersionFrontmatterKeyVisibility(context.Context) error
}

// ResolveSnapshots materializes the effective metadata from the latest and
// live loader snapshots. The snapshots already contain parsed, post-patch
// RawMeta, so this is safe to run as an idempotent startup backfill without
// reparsing historical note_versions. A version present in both snapshots is
// materialized only once.
func ResolveSnapshots(ctx context.Context, env Env, snapshots ...*model.NoteViews) error {
	seen := make(map[int64]struct{})
	notes := make([]*model.NoteView, 0)
	for _, views := range snapshots {
		if views == nil {
			continue
		}
		for _, note := range views.List {
			if note == nil || note.VersionID == 0 {
				continue
			}
			if _, ok := seen[note.VersionID]; ok {
				continue
			}
			seen[note.VersionID] = struct{}{}
			notes = append(notes, note)
		}
	}
	return Resolve(ctx, env, notes)
}

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

// Resolve stores effective (post-patch) RawMeta for the supplied notes. It is
// idempotent and deliberately removes old key links before inserting the new
// set, so deleted frontmatter keys disappear from the current version.
func Resolve(ctx context.Context, env Env, notes []*model.NoteView) error {
	err := Materialize(ctx, env, notes)
	if err != nil {
		return err
	}

	err = env.RefreshNoteVersionFrontmatterKeyVisibility(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh frontmatter key visibility: %w", err)
	}
	return nil
}

// Materialize persists note metadata and key links without refreshing global
// key visibility. Callers processing multiple batches can refresh once after
// all batches complete.
func Materialize(ctx context.Context, env Env, notes []*model.NoteView) error {
	for _, note := range notes {
		if note == nil || note.VersionID == 0 {
			continue
		}

		err := materializeNote(ctx, env, note)
		if err != nil {
			return fmt.Errorf("failed to materialize note version %d: %w", note.VersionID, err)
		}
	}
	return nil
}

func materializeNote(ctx context.Context, env Env, note *model.NoteView) error {
	normalizedMeta := yamlutil.Normalize(note.RawMeta)
	data, err := json.Marshal(normalizedMeta)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}

	frontmatterParams := db.UpsertNoteVersionFrontmatterParams{
		VersionID: note.VersionID,
		Data:      string(data),
	}

	err = env.UpsertNoteVersionFrontmatter(ctx, frontmatterParams)
	if err != nil {
		return fmt.Errorf("upsert frontmatter: %w", err)
	}

	err = env.DeleteNoteVersionFrontmatterKeys(ctx, note.VersionID)
	if err != nil {
		return fmt.Errorf("delete old frontmatter keys: %w", err)
	}

	for key := range note.RawMeta {
		keyParams := db.UpsertNoteVersionFrontmatterKeyParams{
			Value:              key,
			CreatedByVersionID: note.VersionID,
		}

		err = env.UpsertNoteVersionFrontmatterKey(ctx, keyParams)
		if err != nil {
			return fmt.Errorf("upsert frontmatter key %q: %w", key, err)
		}

		linkParams := db.InsertNoteVersionFrontmatterKeyParams{
			NoteVersionID: note.VersionID,
			KeyID:         key,
		}

		err = env.InsertNoteVersionFrontmatterKey(ctx, linkParams)
		if err != nil {
			return fmt.Errorf("link frontmatter key %q: %w", key, err)
		}
	}

	return nil
}
