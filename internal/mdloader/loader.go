package mdloader

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"
	"trip2g/internal/enclavefix"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/image"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	jsonnet "github.com/google/go-jsonnet"
	enclavecore "github.com/quailyquaily/goldmark-enclave/core"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

// SourceFile represents a markdown file.
type SourceFile struct {
	Path      string
	PathID    int64
	VersionID int64
	Content   []byte
	CreatedAt time.Time
	// local file path -> remote file path
	// empty means the file is missing
	Assets map[string]*model.NoteAssetReplace
}

type loader struct {
	nvs *model.NoteViews
	md  goldmark.Markdown
	log logger.Logger

	linkResolver *myLinkResolver

	config Config

	// basenameIndex maps lowercase basename (without extension) to notes
	// Used for O(1) lookup in extractInLinks instead of O(n) iteration
	basenameIndex map[string][]*model.NoteView

	noteCache func(source SourceFile) *model.NoteView

	frontmatterPatches []frontmatterpatch.CompiledPatch
	jsonnetVM          *jsonnet.VM
}

type Config struct {
	AutoLowerWikilinks bool // Deprecated: will be removed
	FreeParagraphs     int  // Default number of free paragraphs from config
	SoftWraps          bool
}

type Options struct {
	Sources []SourceFile
	Log     logger.Logger
	Version string
	Config  Config

	// NoteCache returns cached NoteView if content hasn't changed, nil otherwise
	NoteCache func(source SourceFile) *model.NoteView

	FrontmatterPatches []frontmatterpatch.CompiledPatch
}

// Load transforms markdown files into pages.
func Load(options Options) (*model.NoteViews, error) {
	ldr := &loader{
		log: options.Log,
		nvs: model.NewNoteViews(),

		config:    options.Config,
		noteCache: options.NoteCache,

		linkResolver: &myLinkResolver{},
	}

	ldr.nvs.Version = options.Version
	ldr.linkResolver.nvs = ldr.nvs
	ldr.linkResolver.log = options.Log

	ldr.frontmatterPatches = options.FrontmatterPatches
	if len(ldr.frontmatterPatches) > 0 {
		ldr.jsonnetVM = frontmatterpatch.NewVM()
	}

	renderOptions := []renderer.Option{
		renderer.WithNodeRenderers(util.Prioritized(&linkRenderer{
			resolver: ldr.linkResolver,
			nvs:      ldr.nvs,
		}, 198)),
		renderer.WithNodeRenderers(util.Prioritized(newImageRenderer(ldr.linkResolver), 199)),
	}

	if !options.Config.SoftWraps {
		renderOptions = append(renderOptions, html.WithHardWraps())
	}

	ldr.md = goldmark.New(
		goldmark.WithRendererOptions(renderOptions...),
		goldmark.WithExtensions(
			highlight.Highlight,
			&wikilink.Extender{
				Resolver: ldr.linkResolver,
			},
			extension.GFM,
			enclavefix.New(&enclavecore.Config{}),
			meta.Meta,
		),
	)

	for _, src := range options.Sources {
		page, err := ldr.parsePage(src)
		if err != nil {
			return nil, fmt.Errorf("failed to load page: %w (%s)", err, src.Path)
		}

		page.PathID = src.PathID
		page.VersionID = src.VersionID
		page.CreatedAt = src.CreatedAt
		page.ExtractCreatedAt(time.UTC)

		ldr.nvs.RegisterNote(page)
	}

	ldr.buildBasenameIndex()

	err := ldr.extractInLinks()
	if err != nil {
		return nil, fmt.Errorf("failed to extract in-links: %w", err)
	}

	err = ldr.generatePageHTMLs()
	if err != nil {
		return nil, fmt.Errorf("failed to generate static pages: %w", err)
	}

	ldr.nvs.ExtractNoteList()
	ldr.nvs.ExtractSubgraphs()

	err = ldr.findAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to find assets: %w", err)
	}

	return ldr.nvs, nil
}

func (ldr *loader) markAsset(p *model.NoteView, dest []byte) {
	d := string(dest)

	p.Assets[d] = struct{}{}

	if p.FirstImage == nil && image.IsMediaExtension(d) {
		p.FirstImage = &d
	}
}

func (ldr *loader) findAssets() error {
	for id, p := range ldr.nvs.Map {
		err := ast.Walk(p.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			switch n.Kind() {
			case wikilink.Kind:
				wl, ok := n.(*wikilink.Node)
				if ok && wl.Embed && image.IsMediaExtension(string(wl.Target)) {
					ldr.markAsset(p, wl.Target)
				}

			case ast.KindLink:
				l, ok := n.(*ast.Link)
				if ok && l.Destination != nil {
					url := string(l.Destination)

					// Skip external URLs (http://, https://, //)
					if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "//") {
						break
					}

					// Skip markdown links
					if strings.HasSuffix(url, ".md") {
						break
					}

					// Only mark as asset if it has a media extension (e.g., .png, .jpg)
					// This avoids treating navigation links like [Home](/) as assets
					if !image.IsMediaExtension(url) {
						break
					}

					// Mark as asset (local file)
					ldr.markAsset(p, l.Destination)
				}

			// envclare replace KindImage by their own KindEnclave node
			case enclavecore.KindEnclave:
				e, ok := n.(*enclavecore.Enclave)
				if ok {
					target := string(e.Image.Destination)

					// Skip external URLs
					if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "//") {
						break
					}

					// ignore youtube and other embeded links
					if image.IsMediaExtension(target) {
						ldr.markAsset(p, e.Image.Destination)
					}
				}

			case ast.KindImage:
				i, ok := n.(*ast.Image)
				if ok && i.Destination != nil {
					dest := string(i.Destination)

					// Skip external URLs (http://, https://, //)
					if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") || strings.HasPrefix(dest, "//") {
						break
					}

					ldr.markAsset(p, i.Destination)
				}
			}

			return ast.WalkContinue, nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk AST: %w %s", err, id)
		}
	}

	return nil
}

func (ldr *loader) generatePageHTMLs() error {
	retryNotes := []*model.NoteView{}

	// Use PathMap to iterate over unique notes (Map contains duplicates under different URLs)
	for _, p := range ldr.nvs.PathMap {
		err := ldr.generatePageHTML(p)
		if err != nil {
			if errors.Is(err, errNoHTML) {
				retryNotes = append(retryNotes, p)
				continue
			}

			return fmt.Errorf("failed to generate page: %w (%s)", err, p.Path)
		}
	}

	// Retry generating pages that failed to render HTML
	// it's possible that some pages embedded other notes that yet not processed
	for range 3 {
		for _, p := range retryNotes {
			err := ldr.generatePageHTML(p)
			if err != nil {
				if errors.Is(err, errNoHTML) {
					continue
				}

				return fmt.Errorf("failed to generate page on retry: %w (%s)", err, p.Path)
			}
		}
	}

	return nil
}

func (ldr *loader) generatePageHTML(p *model.NoteView) error {
	var buf bytes.Buffer

	ldr.linkResolver.currentPage = p

	err := ldr.md.Renderer().Render(&buf, p.Content, p.Ast())
	if err != nil {
		return fmt.Errorf("failed to render file: %w", err)
	}

	p.HTML = template.HTML(buf.String()) //nolint:gosec // it's safe from admins

	if p.HTML == "" {
		ldr.log.Warn("generated empty HTML", "path", p.Path, "content_len", len(p.Content), "ast_nil", p.Ast() == nil)
	}

	// Generate free HTML if needed
	err = ldr.generateFreeHTML(p)
	if err != nil {
		ldr.log.Warn("failed to generate free HTML", "path", p.Path, "error", err)
		// Don't fail the whole process if free HTML generation fails
	}

	return nil
}

// buildBasenameIndex creates a map from lowercase basename to notes
// for O(1) lookup in extractInLinks instead of O(n) iteration per link.
func (ldr *loader) buildBasenameIndex() {
	ldr.basenameIndex = make(map[string][]*model.NoteView, len(ldr.nvs.PathMap))
	for path, note := range ldr.nvs.PathMap {
		filename := strings.TrimSuffix(filepath.Base(path), ".md")
		key := strings.ToLower(filename)
		ldr.basenameIndex[key] = append(ldr.basenameIndex[key], note)
	}
}

func (ldr *loader) extractInLinks() error {
	for _, p := range ldr.nvs.PathMap {
		err := ast.Walk(p.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if n.Kind() != wikilink.Kind || !entering {
				return ast.WalkContinue, nil
			}

			link, ok := n.(*wikilink.Node)
			if !ok {
				ldr.log.Warn("failed to cast node to wikilink.Node", "page", p.Path)
				return ast.WalkContinue, nil
			}

			target := string(link.Target)

			// Skip image/video links - they should not be resolved as note links
			if resolveAsImage(link) {
				return ast.WalkContinue, nil
			}

			// Handle explicit relative paths first (./file or ../file)
			//nolint:nestif // complex path resolution with multiple cases
			if strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") {
				dir := filepath.Dir(p.Path)
				if dir == "." {
					dir = ""
				}

				// Clean and join the relative path
				resolvedPath := filepath.Join(dir, target)
				resolvedPath = filepath.Clean(resolvedPath)

				// Try with and without .md extension
				pp, found := ldr.nvs.PathMap[resolvedPath+".md"]
				if !found {
					pp, found = ldr.nvs.PathMap[resolvedPath]
				}

				if found {
					// Use PermalinkOriginal for pages with custom slug (to avoid double encoding)
					// Use Permalink for regular pages (already transliterated)
					// Store in ResolvedLinks for rendering (do NOT mutate AST - breaks caching)
					if pp.Slug != "" {
						p.ResolvedLinks[string(link.Target)] = pp.PermalinkOriginal
					} else {
						p.ResolvedLinks[string(link.Target)] = pp.Permalink
					}
					pp.InLinks[p.Permalink] = struct{}{}

					return ast.WalkContinue, nil
				}

				// Not found, fall through to mark as broken
				_, assetExists := p.AssetReplaces[target]
				if !assetExists {
					if resolveAsImage(link) {
						p.AddWarning(model.NoteWarningInfo, "broken image link: %s", target)
					} else {
						p.AddWarning(model.NoteWarningInfo, "broken link: %s", target)
					}
				}

				return ast.WalkContinue, nil
			}

			// Obsidian behavior: For simple filenames (no path separators),
			// use GLOBAL resolution with shortest-path priority.
			// For paths with '/', use relative path resolution.
			isSimpleFilename := !strings.Contains(target, "/")

			//nolint:nestif // complex filename resolution with candidate matching
			if isSimpleFilename {
				// Global filename resolution (Obsidian behavior)
				// Use pre-built index for O(1) lookup instead of O(n) iteration
				// Note: target comes without .md extension, so just lowercase it
				targetBasename := strings.ToLower(target)
				candidates := ldr.basenameIndex[targetBasename]

				// If we found exactly one match, use it
				if len(candidates) == 1 {
					pp := candidates[0]
					// Use PermalinkOriginal for pages with custom slug (to avoid double encoding)
					// Use Permalink for regular pages (already transliterated)
					// Store in ResolvedLinks for rendering (do NOT mutate AST - breaks caching)
					if pp.Slug != "" {
						p.ResolvedLinks[string(link.Target)] = pp.PermalinkOriginal
					} else {
						p.ResolvedLinks[string(link.Target)] = pp.Permalink
					}
					pp.InLinks[p.Permalink] = struct{}{}

					return ast.WalkContinue, nil
				}

				// If multiple matches, prioritize by shortest path from root
				if len(candidates) > 1 {
					shortest := candidates[0]
					shortestDepth := strings.Count(shortest.Path, "/")

					for _, candidate := range candidates[1:] {
						depth := strings.Count(candidate.Path, "/")
						if depth < shortestDepth {
							shortest = candidate
							shortestDepth = depth
						}
					}

					// Use PermalinkOriginal for pages with custom slug (to avoid double encoding)
					// Use Permalink for regular pages (already transliterated)
					// Store in ResolvedLinks for rendering (do NOT mutate AST - breaks caching)
					if shortest.Slug != "" {
						p.ResolvedLinks[string(link.Target)] = shortest.PermalinkOriginal
					} else {
						p.ResolvedLinks[string(link.Target)] = shortest.Permalink
					}
					shortest.InLinks[p.Permalink] = struct{}{}

					return ast.WalkContinue, nil
				}

				// No matches found in global search, fall through to mark as broken
			} else {
				// Path contains '/', use relative path resolution (walking up the directory tree)
				// Path: content
				// second.md: [[nested/first]]
				// nested/first.md: [[second]]

				dir := filepath.Dir(p.Path)
				if dir == "." {
					dir = ""
				}

				dirParts := strings.Split(dir, "/")

				for i := len(dirParts); i >= 0; i-- {
					targetParts := append([]string{}, dirParts[:i]...)
					targetParts = append(targetParts, target)

					targetPermalink := strings.Join(targetParts, "/")

					pp, found := ldr.nvs.PathMap[targetPermalink+".md"]
					if !found {
						pp, found = ldr.nvs.PathMap[targetPermalink]
					}

					// if p.Path == "эксперимент.md" {
					// 	fmt.Println("targetPermalink", targetPermalink, "target", target)
					// 	fmt.Println(ldr.nvs.PathMap)
					// }

					if found {
						// Use PermalinkOriginal for pages with custom slug (to avoid double encoding)
						// Use Permalink for regular pages (already transliterated)
						// Store in ResolvedLinks for rendering (do NOT mutate AST - breaks caching)
						if pp.Slug != "" {
							p.ResolvedLinks[string(link.Target)] = pp.PermalinkOriginal
						} else {
							p.ResolvedLinks[string(link.Target)] = pp.Permalink
						}
						pp.InLinks[p.Permalink] = struct{}{}

						return ast.WalkContinue, nil
					}
				}
			}

			_, assetExists := p.AssetReplaces[target]
			if !assetExists {
				if resolveAsImage(link) {
					p.AddWarning(model.NoteWarningInfo, "broken image link: %s", target)
				} else {
					p.AddWarning(model.NoteWarningInfo, "broken link: %s", target)
				}
			}

			return ast.WalkContinue, nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk AST: %w", err)
		}
	}

	return nil
}

func (ldr *loader) parsePage(src SourceFile) (*model.NoteView, error) {
	content := src.Content

	if ldr.config.AutoLowerWikilinks {
		// replace [[Wikilink]] with [[Wikilink|wikilink]]
		// skip if . [[Wikilink]] has a dot before it
		content = NormalizeWikilinks(content)
	}

	// Try to get cached AST and meta
	var doc ast.Node
	var rawMeta map[string]interface{}

	if ldr.noteCache != nil {
		if cached := ldr.noteCache(src); cached != nil {
			doc = cached.Ast()
			rawMeta = cached.RawMeta
		}
	}

	// Parse if not cached
	if doc == nil {
		context := parser.NewContext()
		doc = ldr.md.Parser().Parse(text.NewReader(content), parser.WithContext(context))
		rawMeta = meta.Get(context)
	}

	pp := model.NoteView{
		Path:      src.Path,
		PathID:    src.PathID,
		Content:   content,
		InLinks:   make(map[string]struct{}),
		Subgraphs: make(map[string]*model.NoteSubgraph),
		Assets:    make(map[string]struct{}),

		PartialRenderer: &PartialRenderer{md: ldr.md, resolver: ldr.linkResolver},

		ResolvedLinks: make(map[string]string),

		AssetReplaces: src.Assets,
	}

	for k, v := range src.Assets {
		if v == nil {
			pp.AddWarning(model.NoteWarningInfo, "asset %s is missing in the storage", k)
			// TODO: add a placeholder
		}
	}

	// Apply frontmatter patches.
	var appliedPatches []model.AppliedFrontmatterPatch
	rawMeta, appliedPatches = ldr.applyFrontmatterPatches(src.Path, rawMeta, &pp)

	// Use cached or freshly parsed meta
	pp.RawMeta = rawMeta
	pp.AppliedFrontmatterPatches = appliedPatches

	// Extract slug from metadata before preparing permalink
	if slugI, ok := pp.RawMeta["slug"]; ok {
		if slugStr, isString := slugI.(string); isString {
			pp.Slug = slugStr
		}
	}

	pp.PreparePermalink()
	pp.SetAst(doc)

	// Set content and page for partial renderer.
	if partialRenderer, ok := pp.PartialRenderer.(*PartialRenderer); ok {
		partialRenderer.SetContent(doc, content)
		partialRenderer.SetPage(&pp)
	}

	pp.Title = pp.ExtractTitle()
	pp.Free = pp.RawMeta["free"] == true

	err := pp.ExtractSubgraphs()
	if err != nil {
		return nil, fmt.Errorf("failed to extract subgraphs: %w", err)
	}

	err = pp.ExtractMetaData()
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// ldr.log.Debug("read page", "path", pp.Path)

	return &pp, nil
}

func (ldr *loader) applyFrontmatterPatches(path string, rawMeta map[string]interface{}, pp *model.NoteView) (map[string]interface{}, []model.AppliedFrontmatterPatch) {
	if len(ldr.frontmatterPatches) == 0 {
		return rawMeta, nil
	}

	result := frontmatterpatch.ApplyPatches(ldr.jsonnetVM, ldr.frontmatterPatches, path, rawMeta)

	// Convert AppliedPatch to model.AppliedFrontmatterPatch.
	applied := make([]model.AppliedFrontmatterPatch, len(result.AppliedPatches))
	for i, p := range result.AppliedPatches {
		applied[i] = model.AppliedFrontmatterPatch{
			PatchID:     p.PatchID,
			Description: p.Description,
		}
	}

	// Add warnings to the note.
	for _, warning := range result.Warnings {
		pp.AddWarning(model.NoteWarningWarning, "frontmatter patch: %s", warning)
	}

	return result.RawMeta, applied
}
