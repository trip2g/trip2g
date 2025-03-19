package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"

	"trip2g/internal/logger"
	"trip2g/internal/zerologger"
)

type page struct {
	Path    string
	Title   string
	Content string
	InLinks []string
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

	a.startServer()
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

		localPath := path[len(dirPath)+1:]

		a.log.Info("read page", "path", localPath)

		a.Pages[localPath] = &page{
			Path:    localPath,
			Content: string(bContent),
		}

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
