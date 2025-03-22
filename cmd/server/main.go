package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/wikilink"

	"trip2g/internal/logger"
	"trip2g/internal/zerologger"
)

type page struct {
	Path    string
	Title   string
	InLinks []string
	Content []byte
	RawMeta map[string]interface{}
	Ast     ast.Node
}

type app struct {
	Pages map[string]*page

	md goldmark.Markdown

	log logger.Logger
}

func main() {
	a := &app{
		Pages: make(map[string]*page),

		log: zerologger.New("debug", true),
	}

	resolver := myLinkResolver{}

	a.md = goldmark.New(
		goldmark.WithExtensions(
			&wikilink.Extender{
				Resolver: &resolver,
			},
			extension.GFM,
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	err := a.prepare()
	if err != nil {
		panic(err)
	}

	if os.Getenv("SERVER") == "y" {
		a.startServer()
	}
}

func (a *app) prepare() error {
	err := a.readPages()
	if err != nil {
		return fmt.Errorf("failed to read pages: %s", err)
	}

	err = a.generateStaticPages()
	if err != nil {
		return fmt.Errorf("failed to generate static pages: %s", err)
	}

	return nil
}

func (a *app) generateStaticPages() error {
	for _, p := range a.Pages {
		err := a.generatePage(p)
		if err != nil {
			return fmt.Errorf("failed to generate page: %s %s", err, p.Path)
		}
	}

	return nil
}

func (a *app) generatePage(p *page) error {
	const dirPath = "out"

	// replace .md to .html
	htmlPath := p.Path[:len(p.Path)-len(".md")] + ".html"

	// Create the directory if it doesn't exist
	err := os.MkdirAll(filepath.Join(dirPath, filepath.Dir(htmlPath)), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s", err)
	}

	// Create the file
	f, err := os.Create(filepath.Join(dirPath, htmlPath))
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}

	defer f.Close()

	err = a.md.Renderer().Render(f, p.Content, p.Ast)
	if err != nil {
		return fmt.Errorf("failed to render file: %s", err)
	}

	return nil
}

// read all md files from demo/*.md recurlively
func (a *app) readPages() error {
	const dirPath = "demo"

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk path: %s", err)
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		// read file
		bContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %s", err)
		}

		context := parser.NewContext()

		doc := a.md.Parser().Parse(text.NewReader(bContent), parser.WithContext(context))
		pp := page{
			Path:    path[len(dirPath)+1:],
			Content: bContent,
			Ast:     doc,
		}

		err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if n.Kind() != wikilink.Kind {
				return ast.WalkContinue, nil
			}

			link := n.(*wikilink.Node)
			pp.InLinks = append(pp.InLinks, string(link.Target))

			return ast.WalkContinue, nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk AST: %s", err)
		}

		pp.RawMeta = meta.Get(context)

		a.log.Info("read page", "path", pp.Path, "links", pp.InLinks)

		a.Pages[pp.Path] = &pp

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to read pages: %s", err)
	}

	return nil
}

func (*app) startServer() {
	r := gin.Default()

	// Set goview as the HTML renderer
	r.HTMLRender = ginview.New(goview.Config{
		Root:         "views",
		Extension:    ".html",
		Master:       "layout",
		Partials:     []string{},
		Funcs:        template.FuncMap{},
		DisableCache: true,
	})

	// Serve static files
	r.Static("/assets", "./assets")

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "note", gin.H{
			"title": "Note Page",
		})
	})

	r.Run(":8080")
}

type myLinkResolver struct {
	links []string
}

const _html = ".html"
const _hash = "#"

func (r *myLinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	// Remove .html extension if present in the target
	target := n.Target
	if bytes.HasSuffix(target, []byte(_html)) {
		target = target[:len(target)-len(_html)]
	}

	dest := make([]byte, len(target)+len(_hash)+len(n.Fragment))
	var i int
	if len(target) > 0 {
		i += copy(dest, target)
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}
	r.links = append(r.links, string(dest[:i]))
	return dest[:i], nil
}
