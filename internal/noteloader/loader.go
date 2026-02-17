package noteloader

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/sitemap"

	"github.com/blevesearch/bleve/v2"
	"golang.org/x/sync/errgroup"
)

type RawNote struct {
	Path      string
	PathID    int64
	VersionID int64
	Content   string
	CreatedAt time.Time
	Embedding []byte // raw embedding bytes, converted to []float32 later
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
	NoteAssetURL(ctx context.Context, asset db.NoteAsset) (model.PresignedURL, error)
	NoteAssetPath(asset db.NoteAsset) string
	PublicURL() string
	Logger() logger.Logger
	Now() time.Time

	// LoadFrontmatterPatches loads and compiles frontmatter patches from database
	LoadFrontmatterPatches(ctx context.Context) ([]frontmatterpatch.CompiledPatch, error)

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

	version            string
	config             mdloader.Config
	frontmatterPatches []frontmatterpatch.CompiledPatch
}

func New(version string, env Env, config mdloader.Config) *Loader {
	return &Loader{
		env: env,
		log: logger.WithPrefix(env.Logger(), version+" noteloader:"),

		version: version,
		config:  config,
	}
}

// SetFrontmatterPatches sets the frontmatter patches to apply during note loading.
func (l *Loader) SetFrontmatterPatches(patches []frontmatterpatch.CompiledPatch) {
	l.Lock()
	defer l.Unlock()
	l.frontmatterPatches = patches
}

type LoadOptions struct {
	SkipSearchIndex  bool
	ForceRefreshURLs bool // bypass URL cache, regenerate all presigned URLs
}

//nolint:gocognit // complex loading logic with multiple data sources
func (l *Loader) Load(ctx context.Context, options LoadOptions) error {
	// Load frontmatter patches from database before loading notes
	patches, err := l.env.LoadFrontmatterPatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to load frontmatter patches: %w", err)
	}
	l.frontmatterPatches = patches

	notes, err := l.env.RawNotes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	assets, err := l.env.RawAssets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get note assets: %w", err)
	}

	l.log.Debug("check assets")

	// Build cache of existing assets by hash for URL reuse
	cachedAssets := make(map[string]*model.NoteAssetReplace)
	if l.nvs != nil {
		for _, nv := range l.nvs.List {
			for _, ar := range nv.AssetReplaces {
				if ar.Hash == "" {
					continue
				}

				cachedAssets[ar.Hash] = ar
			}
		}
	}

	assetMap, err := l.buildAssetMap(ctx, assets, cachedAssets, options.ForceRefreshURLs)
	if err != nil {
		return fmt.Errorf("failed to build asset map: %w", err)
	}

	mdSources := []mdloader.SourceFile{}
	templateSources := []model.LayoutSourceFile{}
	embeddingMap := make(map[int64][]byte) // version_id -> raw embedding bytes

	const layoutBasePath = "_layouts"

	for _, note := range notes {
		ext := filepath.Ext(note.Path)

		if ext == ".json" && strings.HasSuffix(note.Path, ".html.json") {
			ext = ".html.json"
		}

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

			if len(note.Embedding) > 0 {
				embeddingMap[note.VersionID] = note.Embedding
			}

		case ".html", ".html.json":
			path := strings.Trim(note.Path, "/")

			// handle layouts only under _layouts/
			if !strings.HasPrefix(path, layoutBasePath) {
				continue
			}

			templateSources = append(templateSources, model.LayoutSourceFile{
				Path:            note.Path,
				VersionID:       note.VersionID,
				ID:              path[len(layoutBasePath) : len(path)-len(ext)],
				Content:         note.Content,
				OriginalContent: note.Content,
				Assets:          assetMap[note.VersionID],
			})

		default:
			l.log.Warn("unknown note extension", "path", note.Path, "ext", ext)
		}
	}

	mdOptions := mdloader.Options{
		Sources: mdSources,
		Log:     logger.WithPrefix(l.log, "mdloader:"),
		Version: l.version,
		Config:  l.config,

		// NoteCache returns cached NoteView if content matches to skip parsing.
		// NOTE: If AutoLowerWikilinks is enabled, old.Content is normalized but
		// source.Content is not, causing cache misses. This is acceptable since
		// AutoLowerWikilinks is deprecated and will be removed.
		NoteCache: func(source mdloader.SourceFile) *model.NoteView {
			if l.nvs == nil {
				return nil
			}
			old, ok := l.nvs.PathMap[source.Path]
			if !ok {
				return nil
			}
			if bytes.Equal(old.Content, source.Content) {
				return old
			}
			return nil
		},
		FrontmatterPatches: l.frontmatterPatches,
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

	l.assignEmbeddings(nvs, embeddingMap)

	err = l.generateSitemap(nvs)
	if err != nil {
		return err
	}

	l.log.Debug("done")

	l.Lock()
	l.nvs = nvs
	l.searchIndex = searchIndex
	l.layouts = layouts
	l.Unlock()

	return nil
}

func (l *Loader) buildAssetMap(
	ctx context.Context,
	assets []RawAsset,
	cachedAssets map[string]*model.NoteAssetReplace,
	forceRefresh bool,
) (map[int64]map[string]*model.NoteAssetReplace, error) {
	assetMap := make(map[int64]map[string]*model.NoteAssetReplace)
	minValidExpiry := l.env.Now().Add(time.Minute)

	var cachedCount int

	// Collect assets that need presigned URL generation
	var toGenerate []RawAsset

	for _, asset := range assets {
		noteMap, ok := assetMap[asset.VersionID]
		if !ok {
			noteMap = make(map[string]*model.NoteAssetReplace)
			assetMap[asset.VersionID] = noteMap
		}

		// Try to reuse cached presigned URL if hash matches and URL is still valid
		// Skip cache entirely when forceRefresh is true
		cached, found := cachedAssets[asset.NoteAsset.Sha256Hash]
		if !forceRefresh && found && cached.ExpiresAt.After(minValidExpiry) {
			noteMap[asset.Path] = &model.NoteAssetReplace{
				ID:           asset.NoteAsset.ID,
				URL:          cached.URL,
				Hash:         asset.NoteAsset.Sha256Hash,
				ExpiresAt:    cached.ExpiresAt,
				AbsolutePath: asset.AbsolutePath,
			}
			cachedCount++
			continue
		}

		toGenerate = append(toGenerate, asset)
	}

	generatedCount := len(toGenerate)
	if generatedCount > 0 {
		// Generate presigned URLs in parallel using errgroup
		numWorkers := runtime.NumCPU() * 2
		if numWorkers > generatedCount {
			numWorkers = generatedCount
		}

		type result struct {
			asset        RawAsset
			presignedURL model.PresignedURL
		}

		results := make([]result, generatedCount)
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(numWorkers)

		for i, asset := range toGenerate {
			g.Go(func() error {
				url, err := l.env.NoteAssetURL(gCtx, asset.NoteAsset)
				if err != nil {
					return fmt.Errorf("asset %s: %w", asset.Path, err)
				}
				results[i] = result{asset: asset, presignedURL: url}
				return nil
			})
		}

		err := g.Wait()
		if err != nil {
			return nil, err
		}

		// Apply results to assetMap
		for _, r := range results {
			noteMap := assetMap[r.asset.VersionID]
			noteMap[r.asset.Path] = &model.NoteAssetReplace{
				ID:           r.asset.NoteAsset.ID,
				URL:          r.presignedURL.Value,
				Hash:         r.asset.NoteAsset.Sha256Hash,
				ExpiresAt:    r.presignedURL.ExpiresAt,
				AbsolutePath: r.asset.AbsolutePath,
			}
		}
	}

	l.log.Debug("assets processed", "cached", cachedCount, "generated", generatedCount)

	return assetMap, nil
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

func (l *Loader) assignEmbeddings(nvs *model.NoteViews, embeddingMap map[int64][]byte) {
	count := 0
	for _, nv := range nvs.List {
		if rawEmb, ok := embeddingMap[nv.VersionID]; ok {
			nv.Embedding = bytesToFloat32Slice(rawEmb)
			count++
		}
	}

	l.log.Debug("embeddings loaded", "count", count, "total_notes", len(nvs.List))
}

func (l *Loader) generateSitemap(nvs *model.NoteViews) error {
	sitemapBytes, err := sitemap.Generate(nvs, l.env.PublicURL())
	if err != nil {
		return fmt.Errorf("failed to generate sitemap: %w", err)
	}

	nvs.Sitemap = sitemapBytes

	return nil
}

// bytesToFloat32Slice converts []byte back to []float32.
func bytesToFloat32Slice(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}
	floats := make([]float32, len(data)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return floats
}
