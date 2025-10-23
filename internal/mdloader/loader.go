package mdloader

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	enclave "github.com/quailyquaily/goldmark-enclave"
	enclavecore "github.com/quailyquaily/goldmark-enclave/core"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
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
}

type Config struct {
	AutoLowerWikilinks bool
	FreeParagraphs     int // Default number of free paragraphs from config
}

type Options struct {
	Sources []SourceFile
	Log     logger.Logger
	Version string
	Config  Config
}

// Load transforms markdown files into pages.
func Load(options Options) (*model.NoteViews, error) {
	ldr := &loader{
		log: options.Log,
		nvs: model.NewNoteViews(),

		config: options.Config,

		linkResolver: &myLinkResolver{
			version: options.Version,
		},
	}

	ldr.nvs.Version = options.Version
	ldr.linkResolver.nvs = ldr.nvs
	ldr.linkResolver.log = options.Log

	ldr.md = goldmark.New(
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(&linkRenderer{
				resolver: ldr.linkResolver,
				nvs:      ldr.nvs,
			}, 198)),
		),
		goldmark.WithExtensions(
			highlight.Highlight,
			&wikilink.Extender{
				Resolver: ldr.linkResolver,
			},
			extension.GFM,
			enclave.New(&enclavecore.Config{}),
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

		ldr.nvs.RegisterNote(page)
	}

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

	if p.FirstImage == nil && isImageExtension(d) {
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
				if ok && wl.Embed && isImageExtension(string(wl.Target)) {
					ldr.markAsset(p, wl.Target)
				}

			case ast.KindLink:
				l, ok := n.(*ast.Link)
				if ok && l.Destination != nil {
					url := string(l.Destination)

					// not sure if this is the right way to check for a file link
					if !strings.HasSuffix(url, ".md") {
						ldr.markAsset(p, l.Destination)
					}
				}

			// envclare replace KindImage by their own KindEnclave node
			case enclavecore.KindEnclave:
				e, ok := n.(*enclavecore.Enclave)
				if ok {
					target := string(e.Image.Destination)

					// ignore youtube and other embeded links
					if isImageExtension(target) {
						ldr.markAsset(p, e.Image.Destination)
					}
				}

			case ast.KindImage:
				i, ok := n.(*ast.Image)
				if ok && i.Destination != nil {
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

	for _, p := range ldr.nvs.Map {
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

	// Generate free HTML if needed
	err = ldr.generateFreeHTML(p)
	if err != nil {
		ldr.log.Warn("failed to generate free HTML", "path", p.Path, "error", err)
		// Don't fail the whole process if free HTML generation fails
	}

	return nil
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
					p.ResolvedLinks[string(link.Target)] = pp.Permalink
					pp.InLinks[p.Permalink] = struct{}{}
					link.Target = []byte(pp.Permalink)

					return ast.WalkContinue, nil
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
	context := parser.NewContext()

	content := src.Content

	if ldr.config.AutoLowerWikilinks {
		// replace [[Wikilink]] with [[Wikilink|wikilink]]
		// skip if . [[Wikilink]] has a dot before it
		content = NormalizeWikilinks(content)
	}

	doc := ldr.md.Parser().Parse(text.NewReader(content), parser.WithContext(context))
	pp := model.NoteView{
		Path:      src.Path,
		PathID:    src.PathID,
		Content:   content,
		InLinks:   make(map[string]struct{}),
		Subgraphs: make(map[string]*model.NoteSubgraph),
		Assets:    make(map[string]struct{}),

		PartialRenderer: &PartialRenderer{md: ldr.md},

		ResolvedLinks: make(map[string]string),

		AssetReplaces: src.Assets,
	}

	for k, v := range src.Assets {
		if v == nil {
			pp.AddWarning(model.NoteWarningInfo, "asset %s is missing in the storage", k)
			// TODO: add a placeholder
		}
	}

	pp.PreparePermalink()
	pp.SetAst(doc)

	// Set content for partial renderer
	if partialRenderer, ok := pp.PartialRenderer.(*PartialRenderer); ok {
		partialRenderer.SetContent(doc, content)
	}

	pp.RawMeta = meta.Get(context)
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
