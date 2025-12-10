package noteloader

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/blevesearch/bleve/v2"
)

type RawNote struct {
	Path      string
	PathID    int64
	VersionID int64
	Content   string
	CreatedAt time.Time
}

type RawAsset struct {
	VersionID int64
	Path      string
	NoteAsset db.NoteAsset

	AbsolutePath string
}

type Env interface {
	RawNotes(ctx context.Context) ([]RawNote, error)
	RawAssets(ctx context.Context) ([]RawAsset, error)
	NoteAssetExists(ctx context.Context, asset db.NoteAsset) (bool, error)
	NoteAssetURL(ctx context.Context, asset db.NoteAsset) (string, error)
	NoteAssetPath(asset db.NoteAsset) string
	Logger() logger.Logger

	layoutloader.Env
}

type Loader struct {
	sync.Mutex
	env Env
	nvs *model.NoteViews
	log logger.Logger

	layouts *model.Layouts

	searchIndex   bleve.Index
	contentHashes map[int64][32]byte // PathID -> content hash for incremental indexing

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

type LoadOptions struct {
	SkipSearchIndex bool
}

func (l *Loader) Load(ctx context.Context, options LoadOptions) error {
	notes, err := l.env.RawNotes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	assets, err := l.env.RawAssets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get note assets: %w", err)
	}

	assetMap := make(map[int64]map[string]*model.NoteAssetReplace)

	l.log.Debug("check assets")

	for _, asset := range assets {
		noteMap, ok := assetMap[asset.VersionID]
		if !ok {
			noteMap = make(map[string]*model.NoteAssetReplace)
			assetMap[asset.VersionID] = noteMap
		}

		// TODO: re-enable after fixing startup timeout with many assets
		// Disabled because NoteAssetExists makes HEAD request for each asset,
		// which causes timeout on startup when there are many files.
		//
		// exists, existsErr := l.env.NoteAssetExists(ctx, asset.NoteAsset)
		// if existsErr != nil {
		// 	return fmt.Errorf("failed to check if note asset exists: %w", existsErr)
		// }
		//
		// if !exists {
		// 	l.log.Warn("note asset not exists", "asset", asset, "object_id", l.env.NoteAssetPath(asset.NoteAsset))
		//
		// 	// is not always image... TODO: fix it
		// 	noteMap[asset.Path] = &model.NoteAssetReplace{
		// 		URL:  "/assets/missed_image.png",
		// 		Hash: fmt.Sprintf("%+v", asset),
		//
		// 		AbsolutePath: asset.AbsolutePath,
		// 	}
		//
		// 	continue
		// }

		assetURL, assetErr := l.env.NoteAssetURL(ctx, asset.NoteAsset)
		if assetErr != nil {
			return fmt.Errorf("failed to get note asset URL: %w", assetErr)
		}

		noteMap[asset.Path] = &model.NoteAssetReplace{
			ID:   asset.NoteAsset.ID,
			URL:  assetURL,
			Hash: asset.NoteAsset.Sha256Hash,

			AbsolutePath: asset.AbsolutePath,
		}
	}

	l.log.Debug("load markdown files")

	mdSources := []mdloader.SourceFile{}
	templateSources := []layoutloader.SourceFile{}

	const layoutBasePath = "_layouts"

	for _, note := range notes {
		ext := filepath.Ext(note.Path)

		switch ext {
		case ".md":
			mdSources = append(mdSources, mdloader.SourceFile{
				Path:      note.Path,
				PathID:    note.PathID,
				VersionID: note.VersionID,
				Content:   []byte(note.Content),
				Assets:    assetMap[note.VersionID],
				CreatedAt: note.CreatedAt,
			})

		case ".html":
			path := strings.Trim(note.Path, "/")

			if strings.HasPrefix(path, layoutBasePath) {
				templateSources = append(templateSources, layoutloader.SourceFile{
					Path:      note.Path,
					VersionID: note.VersionID,
					// without prefix and ext, starts with /
					ID:      path[len(layoutBasePath) : len(path)-len(ext)],
					Content: note.Content,
					Assets:  assetMap[note.VersionID],
				})
			}

		default:
			l.log.Warn("unknown note extension", "path", note.Path, "ext", ext)
		}
	}

	mdOptions := mdloader.Options{
		Sources: mdSources,
		Log:     logger.WithPrefix(l.log, "mdloader:"),
		Version: l.version,
		Config:  l.config,
	}

	nvs, err := mdloader.Load(mdOptions)
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	l.log.Debug("load layouts")

	layoutOptions := layoutloader.Options{
		BasePath: layoutBasePath,
	}

	layouts, err := layoutloader.Load(l.env, templateSources, layoutOptions)
	if err != nil {
		return fmt.Errorf("failed to load layouts: %w", err)
	}

	var searchIndex bleve.Index

	if !options.SkipSearchIndex {
		l.log.Debug("build search index")

		searchIndex, err = l.buildSearchIndex(nvs)
		if err != nil {
			return fmt.Errorf("failed to build search index: %w", err)
		}
	}

	l.log.Debug("done")

	l.Lock()
	l.nvs = nvs
	l.searchIndex = searchIndex
	l.layouts = layouts
	l.Unlock()

	return nil
}

func (l *Loader) NoteViews() *model.NoteViews {
	l.Lock()
	defer l.Unlock()
	return l.nvs.Copy() // TODO: optimize
}

func (l *Loader) Layouts() *model.Layouts {
	l.Lock()
	defer l.Unlock()
	return l.layouts
}

func (l *Loader) NoteByPath(path string) *model.NoteView {
	l.Lock()
	defer l.Unlock()

	return l.nvs.Map[path]
}
