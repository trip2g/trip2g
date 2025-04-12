package mdloader

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"trip2g/internal/logger"

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
	Path    string
	Content []byte
}

// Page represents a note page with metadata.
type Page struct {
	Path  string
	Title string

	Content []byte
	HTML    template.HTML
	ast     ast.Node // hide from JSON

	Permalink string
	Free      bool // without the paywall

	InLinks map[string]struct{}
	RawMeta map[string]interface{}

	DeadLinks []string
}

type loader struct {
	pages map[string]*Page
	md    goldmark.Markdown
	log   logger.Logger

	linkResolver *myLinkResolver
}

// Load transforms markdown files into pages.
func Load(sourceFiles []SourceFile, log logger.Logger) (map[string]*Page, error) {
	ldr := &loader{
		log:   log,
		pages: make(map[string]*Page),

		linkResolver: &myLinkResolver{},
	}

	ldr.linkResolver.pages = ldr.pages
	ldr.linkResolver.log = log

	ldr.md = goldmark.New(
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(&linkRenderer{
				resolver: ldr.linkResolver,
				pages:    ldr.pages,
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

		ldr.pages[page.Permalink] = page
	}

	err := ldr.extractInLinks()
	if err != nil {
		return nil, fmt.Errorf("failed to extract in-links: %w", err)
	}

	err = ldr.generatePageHTMLs()
	if err != nil {
		return nil, fmt.Errorf("failed to generate static pages: %w", err)
	}

	return ldr.pages, nil
}

func (ldr *loader) generatePageHTMLs() error {
	for _, p := range ldr.pages {
		err := ldr.generatePageHTML(p)
		if err != nil {
			return fmt.Errorf("failed to generate page: %w (%s)", err, p.Path)
		}
	}

	return nil
}

func (ldr *loader) generatePageHTML(p *Page) error {
	var buf bytes.Buffer

	ldr.linkResolver.currentPage = p

	err := ldr.md.Renderer().Render(&buf, p.Content, p.ast)
	if err != nil {
		return fmt.Errorf("failed to render file: %w", err)
	}

	p.HTML = template.HTML(buf.String()) //nolint:gosec // it's safe from admins

	return nil
}

func (ldr *loader) extractInLinks() error {
	for _, p := range ldr.pages {
		err := ast.Walk(p.ast, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
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

				targetPage, targetOk := ldr.pages[targetPermalink]
				if targetOk {
					targetPage.InLinks[p.Permalink] = struct{}{}
					link.Target = []byte(targetPage.Permalink)
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

func (ldr *loader) parsePage(src SourceFile) (*Page, error) { //nolint:unparam // it's a placeholder
	context := parser.NewContext()

	doc := ldr.md.Parser().Parse(text.NewReader(src.Content), parser.WithContext(context))
	pp := Page{
		Path:      src.Path,
		Permalink: "/" + src.Path[:len(src.Path)-len(".md")],
		Content:   src.Content,
		ast:       doc,
		InLinks:   make(map[string]struct{}),
	}

	pp.RawMeta = meta.Get(context)
	pp.Title = pp.ExtractTitle()
	pp.Free = pp.RawMeta["free"] == true

	ldr.log.Info("read page", "path", pp.Path)

	return &pp, nil
}

func (p *Page) ExtractTitle() string {
	title, ok := p.RawMeta["title"]
	if ok {
		str, sOk := title.(string)
		if sOk {
			return str
		}
	}

	// nodeCount := 0
	// docTitle := ""
	//
	// find the first heading in .Ast
	// Need to remove the heading node before rendering
	// ast.Walk(p.Ast, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
	// 	nodeCount++
	//
	// 	if nodeCount > 5 {
	// 		return ast.WalkStop, nil
	// 	}
	//
	// 	if n.Kind() == ast.KindHeading {
	// 		heading := n.(*ast.Heading)
	//
	// 		if heading.Level != 1 {
	// 			return ast.WalkContinue, nil
	// 		}
	//
	// 		docTitle = string(heading.Text(p.Content))
	// 		return ast.WalkStop, nil
	// 	}
	//
	// 	return ast.WalkContinue, nil
	// })
	//
	// if docTitle != "" {
	// 	return docTitle
	// }

	return filepath.Base(p.Path[:len(p.Path)-len(".md")])
}
