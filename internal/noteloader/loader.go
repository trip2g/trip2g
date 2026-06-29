package noteloader

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
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

// RawNoteChunk is a chunk row as returned by the DB query (Embedding as raw bytes).
type RawNoteChunk struct {
	VersionID  int64
	ChunkIndex int64
	Content    string
	Embedding  []byte
	Path       string
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
	RawNoteChunks(ctx context.Context) ([]RawNoteChunk, error)
	NoteAssetExists(ctx context.Context, asset db.NoteAsset) (bool, error)
	NoteAssetURL(ctx context.Context, asset db.NoteAsset) (model.PresignedURL, error)
	NoteAssetPath(asset db.NoteAsset) string
	PublicURL() string
	Logger() logger.Logger
	Now() time.Time

	// IsDevMode reports whether this is a development instance. It drives
	// jet.DevelopmentMode for layout templates: false (production) caches the
	// compiled layout per reload, true re-parses every render for live reload.
	IsDevMode() bool

	// LoadFrontmatterPatches loads and compiles frontmatter patches from database
	LoadFrontmatterPatches(ctx context.Context) ([]frontmatterpatch.CompiledPatch, error)

	// LoadSiteConfig loads site-wide configuration from database (hot-reloadable)
	LoadSiteConfig(ctx context.Context) (model.SiteConfig, error)

	ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error)

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

	chunks []model.NoteChunk // per-chunk embeddings for vector search

	// prevHTML holds the rendered HTML of notes from the previous Load cycle,
	// captured just before re-rendering changed notes. Replaced entirely each Load.
	// Used by the changedBlocks GraphQL resolver to diff old vs new HTML.
	prevHTML map[int64]htmltemplate.HTML

	version            string
	config             mdloader.Config
	frontmatterPatches []frontmatterpatch.CompiledPatch
	chartData          mdloader.ChartDataProvider

	// patchCache persists frontmatter-patch results across reloads, alongside the
	// note (AST) cache held in l.nvs. It skips re-running the Jsonnet VM per note
	// when neither the note's frontmatter nor the patch set changed.
	patchCache *frontmatterpatch.ResultCache
}

func New(version string, env Env, config mdloader.Config) *Loader {
	return &Loader{
		env: env,
		log: logger.WithPrefix(env.Logger(), version+" noteloader:"),

		version:    version,
		config:     config,
		patchCache: frontmatterpatch.NewResultCache(),
	}
}

// SetFrontmatterPatches sets the frontmatter patches to apply during note loading.
func (l *Loader) SetFrontmatterPatches(patches []frontmatterpatch.CompiledPatch) {
	l.Lock()
	defer l.Unlock()
	l.frontmatterPatches = patches
}

// SetChartDataProvider wires the cached-data provider for url/internal datachart
// sources. Safe to leave unset — those charts then render a loader.
func (l *Loader) SetChartDataProvider(p mdloader.ChartDataProvider) {
	l.Lock()
	defer l.Unlock()
	l.chartData = p
}

type LoadOptions struct {
	SkipSearchIndex  bool
	ForceRefreshURLs bool // bypass URL cache, regenerate all presigned URLs
}

//nolint:gocognit,funlen // complex loading logic with multiple data sources
func (l *Loader) Load(ctx context.Context, options LoadOptions) error {
	// Load site config from database (hot-reloadable URL normalization method, etc.)
	siteConfig, err := l.env.LoadSiteConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load site config: %w", err)
	}
	if siteConfig.URLNormalizationMethod.Valid() {
		l.config.URLNormalizationMethod = siteConfig.URLNormalizationMethod
	}

	// Load frontmatter patches from database before loading notes
	patches, err := l.env.LoadFrontmatterPatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to load frontmatter patches: %w", err)
	}
	// Bypass note cache if patches changed so cached notes get re-parsed with new patches applied.
	patchesChanged := patchListChanged(l.frontmatterPatches, patches)
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
		case ".md", ".canvas", ".base", ".excalidraw":
			// .canvas (JSON) and .base (YAML) are forwarded to mdloader as
			// SourceFiles; mdloader's isRawFile branch stores them verbatim
			// without markdown parsing. Without this case they'd hit the
			// "unknown note extension" default and never reach NoteViews.
			mdSources = append(mdSources, mdloader.SourceFile{
				Path:      note.Path,
				PathID:    note.PathID,
				VersionID: note.VersionID,
				Content:   []byte(note.Content),
				Assets:    assetMap[note.VersionID],
				CreatedAt: note.CreatedAt,
			})

			if ext == ".md" && len(note.Embedding) > 0 {
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

	// Capture HTML of notes that are about to be re-rendered.
	// Used by changedBlocks resolver to diff old vs new HTML.
	// NOTE: when patchesChanged==true, NoteCache returns nil for all notes immediately
	// so prevLocalHTML won't be populated even for content-changed notes — acceptable edge case.
	var prevMu sync.Mutex
	prevLocalHTML := make(map[int64]htmltemplate.HTML)

	mdOptions := mdloader.Options{
		Sources:   mdSources,
		Log:       logger.WithPrefix(l.log, "mdloader:"),
		Version:   l.version,
		Config:    l.config,
		PublicURL: l.env.PublicURL(),
		// NoteCache returns cached NoteView if content matches to skip parsing.
		// NOTE: If AutoLowerWikilinks is enabled, old.Content is normalized but
		// source.Content is not, causing cache misses. This is acceptable since
		// AutoLowerWikilinks is deprecated and will be removed.
		// Bypass cache when patches changed: cached notes must be re-parsed so
		// new/updated patches are applied (routes, etc. derived from patches would
		// otherwise remain stale for content-unchanged notes).
		NoteCache: func(source mdloader.SourceFile) *model.NoteView {
			if patchesChanged || l.nvs == nil {
				return nil
			}
			old, ok := l.nvs.PathMap[source.Path]
			if !ok {
				return nil
			}
			if bytes.Equal(old.Content, source.Content) {
				return old
			}
			// Content changed: save old HTML before re-render for diff.
			// NoteCache may be called concurrently by mdloader workers.
			prevMu.Lock()
			prevLocalHTML[old.PathID] = old.HTML
			prevMu.Unlock()
			return nil
		},
		FrontmatterPatches: l.frontmatterPatches,
		ChartData:          l.chartData,
		PatchCache:         l.patchCache,
	}

	nvs, err := mdloader.Load(mdOptions)
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	// Enrich NoteSubgraphs with DB flags (require_signin).
	dbSubgraphs, err := l.env.ListAllSubgraphs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list subgraphs: %w", err)
	}
	dbSubgraphMap := make(map[string]db.Subgraph, len(dbSubgraphs))
	for _, sg := range dbSubgraphs {
		dbSubgraphMap[sg.Name] = sg
	}
	var requireSigninNames []string
	for name, noteSg := range nvs.Subgraphs {
		if dbSg, ok := dbSubgraphMap[name]; ok {
			noteSg.RequireSignin = dbSg.RequireSignin
		}
		if noteSg.RequireSignin {
			requireSigninNames = append(requireSigninNames, name)
		}
	}
	l.log.Info("subgraphs enriched", "total", len(nvs.Subgraphs), "require_signin", requireSigninNames)
	l.log.Debug("load layouts")

	layoutOptions := layoutloader.Options{
		BasePath: layoutBasePath,
	}

	layouts, err := layoutloader.Load(l.env, templateSources, layoutOptions)
	if err != nil {
		return fmt.Errorf("failed to load layouts: %w", err)
	}

	smokeRenderLayouts(layouts, nvs, l.log)

	var searchIndex bleve.Index

	if !options.SkipSearchIndex {
		l.log.Debug("build search index")

		searchIndex, err = l.buildSearchIndex(nvs)
		if err != nil {
			return fmt.Errorf("failed to build search index: %w", err)
		}
	}

	l.assignEmbeddings(nvs, embeddingMap)

	chunks, err := l.loadChunks(ctx)
	if err != nil {
		// Non-fatal: chunks may not exist yet (before first chunk embedding run)
		l.log.Warn("failed to load note chunks", "err", err)
		chunks = nil
	}

	err = l.generateSitemap(nvs)
	if err != nil {
		return err
	}

	l.log.Debug("done")

	l.Lock()
	l.nvs = nvs
	// Only replace prevHTML when this Load cycle actually re-rendered notes.
	// Spurious reloads (e.g. triggered by subgraph/webhook handling after a save)
	// have count=0 and must not clear the prevHTML that the changedHtmlSelectors
	// resolver is about to read.
	if len(prevLocalHTML) > 0 {
		l.prevHTML = prevLocalHTML
	}
	if !options.SkipSearchIndex {
		l.searchIndex = searchIndex
	}
	l.layouts = layouts
	l.chunks = chunks
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

// PreviousHTML returns the rendered HTML of a note from the previous Load cycle,
// captured just before re-rendering. Returns false if the note was not re-rendered
// in the last Load (new note, unchanged note, or patchesChanged cycle).
func (l *Loader) PreviousHTML(pathID int64) (htmltemplate.HTML, bool) {
	l.Lock()
	defer l.Unlock()
	h, ok := l.prevHTML[pathID]
	return h, ok
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
			nv.Embedding = model.BytesToFloat32Slice(rawEmb)
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

	// Generate per-domain sitemaps.
	scheme := "https"
	if strings.HasPrefix(l.env.PublicURL(), "http://") {
		scheme = "http"
	}
	for _, domain := range nvs.CustomDomains() {
		domainURL := scheme + "://" + domain
		domainSitemap, domainErr := sitemap.GenerateForDomain(nvs, domain, domainURL)
		if domainErr != nil {
			l.log.Warn("failed to generate domain sitemap", "domain", domain, "err", domainErr)
			continue
		}
		if len(domainSitemap) > 0 {
			nvs.DomainSitemaps[domain] = domainSitemap
		}
	}

	return nil
}

// NoteChunks returns the in-memory chunk list for vector search.
func (l *Loader) NoteChunks() []model.NoteChunk {
	l.Lock()
	defer l.Unlock()
	return l.chunks
}

// loadChunks fetches all chunk rows from the DB via the Env and converts them
// to model.NoteChunk (deserializing raw float32 LE bytes into []float32).
func (l *Loader) loadChunks(ctx context.Context) ([]model.NoteChunk, error) {
	rawChunks, err := l.env.RawNoteChunks(ctx)
	if err != nil {
		return nil, err
	}

	chunks := make([]model.NoteChunk, 0, len(rawChunks))
	for _, rc := range rawChunks {
		if len(rc.Embedding) == 0 {
			continue
		}
		chunks = append(chunks, model.NoteChunk{
			VersionID:  rc.VersionID,
			ChunkIndex: int(rc.ChunkIndex),
			Content:    rc.Content,
			Embedding:  model.BytesToFloat32Slice(rc.Embedding),
			NotePath:   rc.Path,
		})
	}

	l.log.Debug("chunks loaded", "count", len(chunks))
	return chunks, nil
}

// patchListChanged reports whether the new patch list differs from the old one
// by comparing lengths, IDs, and compiled source of each patch.
func patchListChanged(old, newPatches []frontmatterpatch.CompiledPatch) bool {
	if len(old) != len(newPatches) {
		return true
	}
	for i := range old {
		if old[i].ID != newPatches[i].ID || old[i].WrappedSource != newPatches[i].WrappedSource {
			return true
		}
	}
	return false
}
