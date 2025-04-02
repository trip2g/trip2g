package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/zerologger"
)

type app struct {
	Pages map[string]*mdloader.Page

	log logger.Logger
}

func main() {
	a := &app{
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
	sources, err := a.readPages()
	if err != nil {
		return fmt.Errorf("failed to read pages: %w", err)
	}

	a.Pages, err = mdloader.Load(sources, logger.WithPrefix(a.log, "mdloader:"))
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	return nil
}

// read all md files from demo/*.md recurlively.
func (a *app) readPages() ([]mdloader.SourceFile, error) {
	// const dirPath = "demo"
	const dirPath = "../secondbrain"

	sources := []mdloader.SourceFile{}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk path: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		localPath := path[len(dirPath)+1:]

		if localPath[0] == '.' {
			return nil
		}

		bContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		sources = append(sources, mdloader.SourceFile{
			Path:    localPath,
			Content: bContent,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read pages: %w", err)
	}

	return sources, nil
}

func (a *app) startServer() {
	r := gin.Default()

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("trip2g_session", store))

	// expectedHost := "localhost:8080"

	r.Use(func(c *gin.Context) {
		// if c.Request.Host != expectedHost {
		// 	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid host header"})
		// 	return
		// }

		// tmp allow all origins and cors
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'; connect-src *; font-src *; script-src-elem * 'unsafe-inline'; img-src * data:; style-src * 'unsafe-inline';")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Referrer-Policy", "strict-origin")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	})

	// Set goview as the HTML renderer
	r.HTMLRender = ginview.New(goview.Config{
		Root:      "views",
		Extension: ".html",
		Master:    "layout",
		Partials:  []string{},
		Funcs: template.FuncMap{
			"getPage": func(target string) *mdloader.Page {
				return a.Pages[target]
			},
			"getPageLinkClasses": func(target string) string {
				page, ok := a.Pages[target]

				if ok && !page.Free {
					return "paywall"
				}

				return ""
			},
		},
		DisableCache: true,
	})

	// Serve static files
	r.Static("/assets", "./assets")

	r.GET("/api/pages", func(c *gin.Context) {
		c.JSON(http.StatusOK, a.Pages)
	})

	// POST /_system/signin
	r.POST("/_system/signin", func(c *gin.Context) {
		var form struct {
			Password string `form:"password"`
			Email    string `form:"email"`
			ReturnTo string `form:"return_to"`
		}

		err := c.ShouldBind(&form)
		if err != nil {
			c.String(http.StatusBadRequest, "400 Bad Request")
			return
		}

		if form.Email == "test@example.com" && form.Password == "X173T6pThLNm" {
			session := sessions.Default(c)
			session.Set("authenticated", true)
			session.Save()

			if form.ReturnTo == "" {
				form.ReturnTo = "/"
			}

			c.Redirect(http.StatusSeeOther, form.ReturnTo)
			return
		}

		c.String(http.StatusUnauthorized, "401 Unauthorized")
	})

	// POST /_system/signout
	r.POST("/_system/signout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()

		c.Redirect(http.StatusSeeOther, "/")
	})

	// /api/graph in format {"nodes": [{ key: "key" }], "edges": [{ source: "source", target: "target" }]}
	r.GET("/api/graph", func(c *gin.Context) {
		nodes := []gin.H{}
		edges := []gin.H{}

		rand.Seed(time.Now().UnixNano())

		for _, page := range a.Pages {
			x := rand.Intn(1000)
			y := rand.Intn(1000)

			size := float32(len(page.InLinks))*0.2 + 1
			if size < 1 {
				size = 1
			}

			if size > 50 {
				size = 50
			}

			nodes = append(nodes, gin.H{
				"key": page.Title,
				"attributes": gin.H{
					"x":     x,
					"y":     y,
					"size":  size,
					"label": page.Title,
					"color": "#D8482D",
				},
			})

			x++

			for permalink := range page.InLinks {
				edges = append(edges, gin.H{
					"source": a.Pages[permalink].Title,
					"target": page.Title,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"nodes": nodes,
			"edges": edges,
		})
	})

	render := func(c *gin.Context, code int, template string, data gin.H) {
		session := sessions.Default(c)

		data["isGuest"] = session.Get("authenticated") == nil

		c.HTML(http.StatusOK, template, data)
	}

	// not found handler
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == "/" {
			path = "/index"
		}

		page, ok := a.Pages[path]
		if !ok {
			c.String(http.StatusNotFound, "404 Not Found")
			return
		}

		session := sessions.Default(c)

		if !page.Free && session.Get("authenticated") == nil {
			render(c, http.StatusUnauthorized, "paywall", gin.H{"page": page})
			return
		}

		render(c, http.StatusOK, "note", gin.H{"page": page})
	})

	err := r.Run(":8080")
	if err != nil {
		panic(err)
	}
}
