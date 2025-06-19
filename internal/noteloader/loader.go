package noteloader

import (
	"context"
	"fmt"
	"sync"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
)

type RawNote struct {
	Path      string
	PathID    int64
	VersionID int64
	Content   string
}

type RawAsset struct {
	VersionID int64
	Path      string
	NoteAsset db.NoteAsset
}

type Env interface {
	RawNotes(ctx context.Context) ([]RawNote, error)
	RawAssets(ctx context.Context) ([]RawAsset, error)
	NoteAssetURL(ctx context.Context, asset db.NoteAsset) (string, error)
	Logger() logger.Logger
}

type Loader struct {
	sync.Mutex
	env Env
	nvs *model.NoteViews
	log logger.Logger

	version string
	config  mdloader.Config
}

func New(version string, env Env, config mdloader.Config) *Loader {
	return &Loader{
		env: env,
		log: logger.WithPrefix(env.Logger(), version+" noteloader:"),

		version: version,
		config:  config,
	}
}

func (l *Loader) Load(ctx context.Context) error {
	notes, err := l.env.RawNotes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	assets, err := l.env.RawAssets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get note assets: %w", err)
	}

	assetMap := make(map[int64]map[string]string)

	for _, asset := range assets {
		noteMap, ok := assetMap[asset.VersionID]
		if !ok {
			noteMap = make(map[string]string)
			assetMap[asset.VersionID] = noteMap
		}

		assetURL, assetErr := l.env.NoteAssetURL(ctx, asset.NoteAsset)
		if assetErr != nil {
			return fmt.Errorf("failed to get note asset URL: %w", assetErr)
		}

		noteMap[asset.Path] = assetURL
	}

	sources := []mdloader.SourceFile{}

	for _, note := range notes {
		sources = append(sources, mdloader.SourceFile{
			Path:      note.Path,
			PathID:    note.PathID,
			VersionID: note.VersionID,
			Content:   []byte(note.Content),
			Assets:    assetMap[note.VersionID],
		})
	}

	options := mdloader.Options{
		Sources: sources,
		Log:     logger.WithPrefix(l.log, "mdloader:"),
		Version: l.version,
		Config:  l.config,
	}

	nvs, err := mdloader.Load(options)
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	l.Lock()
	l.nvs = nvs
	l.Unlock()

	return nil
}

func (l *Loader) NoteViews() *model.NoteViews {
	l.Lock()
	defer l.Unlock()
	return l.nvs.Copy()
}

func (l *Loader) NoteByPath(path string) *model.NoteView {
	l.Lock()
	defer l.Unlock()

	return l.nvs.Map[path]
}
