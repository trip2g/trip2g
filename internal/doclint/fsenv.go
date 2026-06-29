// Package doclint provides a DB-free doc/note linter that runs the real
// noteloader pipeline over a filesystem directory and reports warnings.
package doclint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
)

// fsEnv implements noteloader.Env backed by the local filesystem.
// It is DB-free: it walks a directory tree for notes, returns no assets,
// and provides no-op stubs for DB-dependent operations.
type fsEnv struct {
	dir string
	log logger.Logger
}

// compile-time check
var _ noteloader.Env = (*fsEnv)(nil)

func newFsEnv(dir string, log logger.Logger) *fsEnv {
	return &fsEnv{dir: dir, log: log}
}

// noteExts is the set of file extensions treated as note sources.
var noteExts = map[string]bool{
	".md":          true,
	".canvas":      true,
	".base":        true,
	".excalidraw":  true,
}

// RawNotes walks dir and returns every *.md, *.canvas, *.base, *.excalidraw file
// plus any _layouts/*.html and _layouts/*.html.json file as synthetic RawNotes.
// PathID and VersionID are simple counters (1-based, unique per file).
func (e *fsEnv) RawNotes(_ context.Context) ([]noteloader.RawNote, error) {
	var notes []noteloader.RawNote
	var counter int64

	err := filepath.WalkDir(e.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(e.dir, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		ext := strings.ToLower(filepath.Ext(rel))
		isHTMLExt := ext == ".html"
		isHTMLJSONExt := strings.HasSuffix(rel, ".html.json")
		isLayout := strings.HasPrefix(rel, "_layouts/") && (isHTMLExt || isHTMLJSONExt)

		if !noteExts[ext] && !isLayout {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		counter++
		notes = append(notes, noteloader.RawNote{
			Path:      rel,
			PathID:    counter,
			VersionID: counter,
			Content:   string(content),
			CreatedAt: time.Now(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return notes, nil
}

// RawAssets returns nil: the FS env does not map note assets.
// This means any ![[image.png]] reference will emit a broken-image warning;
// that is acceptable ("noisier") for the lint use-case.
func (e *fsEnv) RawAssets(_ context.Context) ([]noteloader.RawAsset, error) {
	return nil, nil
}

// RawNoteChunks returns nil: vector-search chunks are not needed for linting.
func (e *fsEnv) RawNoteChunks(_ context.Context) ([]noteloader.RawNoteChunk, error) {
	return nil, nil
}

// NoteAssetExists always returns false: assets are not tracked in the FS env.
func (e *fsEnv) NoteAssetExists(_ context.Context, _ db.NoteAsset) (bool, error) {
	return false, nil
}

// NoteAssetURL returns an empty presigned URL: no object storage in the FS env.
func (e *fsEnv) NoteAssetURL(_ context.Context, _ db.NoteAsset) (model.PresignedURL, error) {
	return model.PresignedURL{}, nil
}

// NoteAssetPath returns an empty string: no asset storage path in the FS env.
func (e *fsEnv) NoteAssetPath(_ db.NoteAsset) string {
	return ""
}

// PublicURL returns an empty string: no server URL in the FS env.
func (e *fsEnv) PublicURL() string {
	return ""
}

// Logger returns the logger provided at construction time.
func (e *fsEnv) Logger() logger.Logger {
	return e.log
}

// Now returns the current wall-clock time.
func (e *fsEnv) Now() time.Time {
	return time.Now()
}

// IsDevMode returns false: lint always runs in production (caching) mode.
func (e *fsEnv) IsDevMode() bool {
	return false
}

// LoadFrontmatterPatches returns nil: no DB-backed patches in the FS env.
func (e *fsEnv) LoadFrontmatterPatches(_ context.Context) ([]frontmatterpatch.CompiledPatch, error) {
	return nil, nil
}

// LoadSiteConfig returns a zero SiteConfig: no site configuration in the FS env.
func (e *fsEnv) LoadSiteConfig(_ context.Context) (model.SiteConfig, error) {
	return model.SiteConfig{}, nil
}

// ListAllSubgraphs returns nil: no subgraph metadata in the FS env.
func (e *fsEnv) ListAllSubgraphs(_ context.Context) ([]db.Subgraph, error) {
	return nil, nil
}
