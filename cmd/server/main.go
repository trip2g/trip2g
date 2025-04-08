package main

//go:generate go get -u github.com/valyala/quicktemplate/qtc
//go:generate qtc -dir=views

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"

	"trip2g/internal/db"
	"trip2g/internal/graph"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/usertoken"
	"trip2g/internal/zerologger"
	"trip2g/views"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	_ "modernc.org/sqlite"
)

type app struct {
	mu sync.Mutex

	pages map[string]*mdloader.Page

	queries *db.Queries
	conn    *sql.DB

	log logger.Logger
}

type updateRequest struct {
	Updates []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"updates"`
}

func main() {
	u, _ := url.Parse("sqlite:data.sqlite3")
	dbm := dbmate.New(u)

	err := dbm.CreateAndMigrate()
	if err != nil {
		panic(err)
	}

	conn, err := sql.Open("sqlite", "data.sqlite3")
	if err != nil {
		panic(err)
	}

	a := &app{
		log:     zerologger.New("debug", true),
		queries: db.New(conn),
		conn:    conn,
	}

	ctx := context.Background()

	err = a.prepare(ctx, a.queries)
	if err != nil {
		panic(err)
	}

	if os.Getenv("SERVER") == "y" {
		go a.startServer()
		a.startServer2()
	}
}

func (a *app) prepare(ctx context.Context, queries *db.Queries) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	a.mu.Lock()
	defer a.mu.Unlock()

	notes, err := queries.AllLatestNotes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	sources := []mdloader.SourceFile{}

	for _, note := range notes {
		sources = append(sources, mdloader.SourceFile{
			Path:    note.Path,
			Content: []byte(note.Content),
		})
	}

	pages, err := mdloader.Load(sources, logger.WithPrefix(a.log, "mdloader:"))
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	a.pages = pages

	return nil
}

func (a *app) insertNotes(ctx context.Context, data *updateRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := a.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	queries := a.queries.WithTx(tx)

	for _, update := range data.Updates {
		note := db.Note{
			Path:    update.Path,
			Content: update.Content,
		}

		err = queries.InsertNote(ctx, note)
		if err != nil {
			return fmt.Errorf("failed to insert note: %w", err)
		}
	}

	err = a.prepare(ctx, queries)
	if err != nil {
		return fmt.Errorf("failed to prepare notes: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (a *app) handlePages(ctx *fasthttp.RequestCtx, path string) {
	if path == "/" {
		path = "/index"
	}

	page, ok := a.pages[path]
	if !ok {
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetBodyString("404 Not Found")
		return
	}

	// write page.HTML
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	views.WriteLayoutHeader(ctx)
	views.WriteNote(ctx, page, a.pages)
	views.WriteLayoutFooter(ctx)
}

// startServer2 with fasthttp
func (a *app) startServer2() {
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(playground.Handler("GraphQL playground", "/graphql"))
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(srv)

	fs := &fasthttp.FS{
		Root:               "./assets",
		IndexNames:         []string{},
		GenerateIndexPages: false,
		Compress:           true,
		AcceptByteRange:    true,

		PathRewrite: func(ctx *fasthttp.RequestCtx) []byte {
			// remove /assets prefix
			return ctx.Path()[7:]
		},
	}

	fsHandler := fs.NewRequestHandler()

	tokenExtractor := usertoken.NewExtractor("trip2g_token", []byte("secret"))

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			path := string(ctx.Path())

			// TODO: only for dev
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")

			if strings.HasPrefix(path, "/assets/") {
				fsHandler(ctx)
				return
			}

			err := tokenExtractor.Extract(ctx)
			if err != nil {
				ctx.SetStatusCode(http.StatusUnauthorized)
				ctx.SetBodyString("401 Unauthorized")
				return
			}

			switch string(ctx.Path()) {
			case "/graphql":
				if string(ctx.Method()) == "GET" {
					playgroundHandler(ctx)
				} else {
					graphqlHandler(ctx)
				}
				return
			default:
				a.handlePages(ctx, path)
			}
		},
	}

	err := s.ListenAndServe(":8081")
	if err != nil {
		panic(err)
	}
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
				return a.pages[target]
			},
			"getPageLinkClasses": func(target string) string {
				page, ok := a.pages[target]

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

	// GET /api/note_paths
	r.GET("/api/note_paths", func(c *gin.Context) {
		paths, err := a.queries.AllNotePaths(c.Request.Context())
		if err != nil {
			a.log.Error("failed to get note paths", "err", err)
			c.String(http.StatusInternalServerError, "500 Internal Server Error")
			return
		}

		c.JSON(http.StatusOK, gin.H{"paths": paths})
	})

	// GET /api/note_paths/:id/note_versions
	r.GET("/api/note_paths/:id/note_versions", func(c *gin.Context) {
		rawID := c.Param("id")
		if rawID == "" {
			c.String(http.StatusBadRequest, "400 Bad Request")
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			c.String(http.StatusBadRequest, "400 Bad Request")
			return
		}

		versions, err := a.queries.AllNoteVersionsByPathID(c.Request.Context(), int64(id))
		if err != nil {
			a.log.Error("failed to get note versions", "err", err)
			c.String(http.StatusInternalServerError, "500 Internal Server Error")
			return
		}

		c.JSON(http.StatusOK, gin.H{"versions": versions})
	})

	// POST /api/notes that takes a JSON object in the format {"path": "path", "content": "content"}
	r.POST("/api/notes", func(c *gin.Context) {
		var form updateRequest

		err := c.ShouldBindJSON(&form)
		if err != nil {
			c.String(http.StatusBadRequest, "400 Bad Request")
			return
		}

		err = a.insertNotes(c.Request.Context(), &form)
		if err != nil {
			a.log.Error("failed to insert note", "err", err)
			c.String(http.StatusInternalServerError, "500 Internal Server Error")
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

		for _, page := range a.pages {
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
					"source": a.pages[permalink].Title,
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

		page, ok := a.pages[path]
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
