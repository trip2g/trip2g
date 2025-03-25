package main

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
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
	Path      string
	Permalink string
	Title     string

	InLinks map[string]struct{}
	Content []byte
	HTML    htmltemplate.HTML
	RawMeta map[string]interface{}
	Ast     ast.Node
}

func (p *page) ExtractTitle() string {
	title, ok := p.RawMeta["title"]
	if ok {
		str, ok := title.(string)
		if ok {
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

type app struct {
	Pages map[string]*page

	md goldmark.Markdown

	linkResolver *myLinkResolver

	log logger.Logger
}

func main() {
	a := &app{
		Pages: make(map[string]*page),

		log: zerologger.New("debug", true),

		linkResolver: &myLinkResolver{},
	}

	a.md = goldmark.New(
		goldmark.WithExtensions(
			&wikilink.Extender{
				Resolver: a.linkResolver,
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

	err = a.extractInLinks()
	if err != nil {
		return fmt.Errorf("failed to extract in-links: %s", err)
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

func (a *app) extractInLinks() error {
	for _, p := range a.Pages {
		err := ast.Walk(p.Ast, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if n.Kind() != wikilink.Kind {
				return ast.WalkContinue, nil
			}

			link := n.(*wikilink.Node)
			target := string(link.Target) + ".md"

			targetPage, ok := a.Pages[target]
			if !ok {
				fmt.Println("page not found", target)
				return ast.WalkContinue, nil
			}

			targetPage.InLinks[p.Path] = struct{}{}

			return ast.WalkContinue, nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk AST: %s", err)
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

	var buf bytes.Buffer

	err = a.md.Renderer().Render(&buf, p.Content, p.Ast)
	if err != nil {
		return fmt.Errorf("failed to render file: %s", err)
	}

	_, err = f.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write file: %s", err)
	}

	p.HTML = htmltemplate.HTML(buf.String())

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
			Path:      path[len(dirPath)+1:],
			Permalink: "/" + path[len(dirPath)+1:len(path)-len(".md")],
			Content:   bContent,
			Ast:       doc,
			InLinks:   make(map[string]struct{}),
		}

		pp.RawMeta = meta.Get(context)
		pp.Title = pp.ExtractTitle()

		a.log.Info("read page", "path", pp.Path, "links", pp.InLinks)

		a.Pages[pp.Path] = &pp

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to read pages: %s", err)
	}

	return nil
}

func (a *app) startServer() {
	r := gin.Default()

	// Set goview as the HTML renderer
	r.HTMLRender = ginview.New(goview.Config{
		Root:      "views",
		Extension: ".html",
		Master:    "layout",
		Partials:  []string{},
		Funcs: template.FuncMap{
			"getPage": func(target string) *page {
				return a.Pages[target]
			},
		},
		DisableCache: true,
	})

	// Serve static files
	r.Static("/assets", "./assets")

	// not found handler
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path[1:]

		if path == "" {
			path = "index"
		}

		page, ok := a.Pages[path+".md"]
		if !ok {
			c.String(http.StatusNotFound, "404 Not Found")
			return
		}

		inLinks := map[string]string{}

		c.HTML(http.StatusOK, "note", gin.H{
			"page":    page,
			"inLinks": inLinks,
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
	return dest[:i], nil
}
