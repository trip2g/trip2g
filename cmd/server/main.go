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
	"gopkg.in/yaml.v2"

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

// Frontmatter represents the YAML metadata at the top of markdown files
type Frontmatter struct {
	Title string `yaml:"title"`
}

type page struct {
	Path    string
	Title   string
	Content string
	InLinks []string
	RawMeta map[string]interface{}
	Ast     ast.Node
}

type app struct {
	Pages map[string]*page

	log logger.Logger
}

func main() {
	a := &app{
		Pages: make(map[string]*page),

		log: zerologger.New("debug", true),
	}

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

		resolver := myLinkResolver{}
		context := parser.NewContext()

		md := goldmark.New(
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

		doc := md.Parser().Parse(text.NewReader(bContent), parser.WithContext(context))
		pp := page{
			Path:    path[len(dirPath)+1:],
			Content: string(bContent),
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

func extractFrontmatter(content []byte) (Frontmatter, []byte, error) {
	var meta Frontmatter

	// Check if content starts with frontmatter delimiters
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return meta, content, nil
	}

	// Find the end of the frontmatter
	endIdx := bytes.Index(content[4:], []byte("\n---"))
	if endIdx == -1 {
		return meta, content, nil
	}
	endIdx += 4 // Adjust for the offset in the slice

	// Extract the frontmatter section
	frontmatterBytes := content[4:endIdx]

	// Parse the YAML
	if err := yaml.Unmarshal(frontmatterBytes, &meta); err != nil {
		return meta, content, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Return the content without the frontmatter
	return meta, content[endIdx+4:], nil
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
