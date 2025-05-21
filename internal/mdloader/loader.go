package mdloader

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"trip2g/internal/logger"
	"trip2g/internal/model"

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
	Assets    map[string]string // local file path -> remote file path
}

type loader struct {
	nvs *model.NoteViews
	md  goldmark.Markdown
	log logger.Logger

	linkResolver *myLinkResolver
}

// Load transforms markdown files into pages.
func Load(sourceFiles []SourceFile, log logger.Logger) (*model.NoteViews, error) {
	ldr := &loader{
		log: log,
		nvs: model.NewNoteViews(),

		linkResolver: &myLinkResolver{},
	}

	ldr.linkResolver.nvs = ldr.nvs
	ldr.linkResolver.log = log

	ldr.md = goldmark.New(
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(&linkRenderer{
				resolver: ldr.linkResolver,
				nvs:      ldr.nvs,
			}, 198)),
		),
		goldmark.WithExtensions(
			&wikilink.Extender{
				Resolver: ldr.linkResolver,
			},
			extension.GFM,
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	for _, src := range sourceFiles {
		page, err := ldr.parsePage(src)
		if err != nil {
			return nil, fmt.Errorf("failed to load page: %w (%s)", err, src.Path)
		}

		page.PathID = src.PathID
		page.VersionID = src.VersionID

		ldr.nvs.Map[page.Permalink] = page
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

func (ldr *loader) findAssets() error {
	for id, p := range ldr.nvs.Map {
		err := ast.Walk(p.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			switch n.Kind() {
			case wikilink.Kind:

				wl := n.(*wikilink.Node)

				if wl.Embed {
					p.Assets[string(wl.Target)] = struct{}{}
				}

			case ast.KindLink:
				l := n.(*ast.Link)

				if l.Destination != nil {
					url := string(l.Destination)

					// not sure if this is the right way to check for a file link
					if !strings.HasSuffix(url, ".md") {
						p.Assets[string(l.Destination)] = struct{}{}
					}
				}

			case ast.KindImage:
				i := n.(*ast.Image)

				if i.Destination != nil {
					p.Assets[string(i.Destination)] = struct{}{}
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
	for _, p := range ldr.nvs.Map {
		err := ldr.generatePageHTML(p)
		if err != nil {
			return fmt.Errorf("failed to generate page: %w (%s)", err, p.Path)
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

	return nil
}

func (ldr *loader) extractInLinks() error {
	for _, p := range ldr.nvs.Map {
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
			if target[0] != '/' {
				target = "/" + target
			}

			// resolve relative links
			currentParts := strings.Split(p.Permalink, "/")

			for i := len(currentParts) - 1; i >= 0; i-- {
				targetPermalink := strings.Join(currentParts[:i], "/") + target

				targetNote := ldr.nvs.GetByPath(targetPermalink)
				if targetNote != nil {
					targetNote.InLinks[p.Permalink] = struct{}{}
					link.Target = []byte(targetNote.Permalink)
					return ast.WalkContinue, nil
				}
			}

			p.DeadLinks = append(p.DeadLinks, target)

			return ast.WalkContinue, nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk AST: %w", err)
		}
	}

	return nil
}

func (ldr *loader) parsePage(src SourceFile) (*model.NoteView, error) { //nolint:unparam // it's a placeholder
	context := parser.NewContext()

	doc := ldr.md.Parser().Parse(text.NewReader(src.Content), parser.WithContext(context))
	pp := model.NoteView{
		Path:      src.Path,
		Permalink: "/" + src.Path[:len(src.Path)-len(".md")],
		Content:   src.Content,
		InLinks:   make(map[string]struct{}),
		Subgraphs: make(map[string]*model.NoteSubgraph),
		Assets:    make(map[string]struct{}),

		AssetReplaces: src.Assets,
	}

	pp.SetAst(doc)

	pp.RawMeta = meta.Get(context)
	pp.Title = pp.ExtractTitle()
	pp.Free = pp.RawMeta["free"] == true

	err := pp.ExtractSubgraphs()
	if err != nil {
		return nil, fmt.Errorf("failed to extract subgraphs: %w", err)
	}

	ldr.log.Info("read page", "path", pp.Path)

	return &pp, nil
}
