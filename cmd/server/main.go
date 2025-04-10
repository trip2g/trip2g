package main

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=../../views

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"

	"trip2g/internal/case/getnotehashes"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/db"
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

	err = a.PrepareNotes(ctx)
	if err != nil {
		panic(err)
	}

	if os.Getenv("SERVER") == "y" {
		a.startServer()
	}
}

func (a *app) PrepareNotes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	a.mu.Lock()
	defer a.mu.Unlock()

	notes, err := a.queries.AllLatestNotes(ctx)
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

func (a *app) handlePages(ctx *fasthttp.RequestCtx, path string, token *usertoken.Data) {
	if path == "/" {
		path = "/index"
	}

	page, ok := a.pages[path]
	if !ok {
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetBodyString("404 Not Found")
		return
	}

	if !page.Free && token == nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetContentType("text/html; charset=utf-8")

		views.WriteLayoutHeader(ctx, page.Title)
		views.WritePayWall(ctx, page)
		views.WriteLayoutFooter(ctx)

		return
	}

	// write page.HTML
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	views.WriteLayoutHeader(ctx, page.Title)
	views.WriteNote(ctx, page, a.pages)
	views.WriteLayoutFooter(ctx)
}

func (a *app) InsertNote(ctx context.Context, data db.Note) error {
	return a.queries.InsertNote(ctx, data)
}

func (a *app) AllNotePaths(ctx context.Context) ([]db.NotePath, error) {
	return a.queries.AllNotePaths(ctx)
}

func (a *app) startServer() {
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

			token, err := tokenExtractor.Extract(ctx)
			if err != nil && !errors.Is(err, usertoken.ErrTokenMissing) {
				ctx.SetStatusCode(http.StatusUnauthorized)
				ctx.SetBodyString("401 Unauthorized")
				return
			}

			switch string(ctx.Path()) {
			case "/api/getnotehashes":
				request := getnotehashes.Request{}

				response, err := getnotehashes.Resolve(ctx, a, request)
				if err != nil {
					a.log.Error("failed to resolve getnotehashes", "err", err)
					ctx.SetStatusCode(http.StatusInternalServerError)
					ctx.SetBodyString("500 Internal Server Error")
					return
				}

				ctx.SetStatusCode(http.StatusOK)
				ctx.SetContentType("application/json; charset=utf-8")

				rawBytes, err := easyjson.Marshal(response)
				if err != nil {
					a.log.Error("failed to marshal getnotehashes response", "err", err)
					ctx.SetStatusCode(http.StatusInternalServerError)
					ctx.SetBodyString("500 Internal Server Error")
					return
				}

				ctx.SetBody(rawBytes)

			case "/api/pushnotes":
				if string(ctx.Method()) != "POST" {
					ctx.SetStatusCode(http.StatusMethodNotAllowed)
					ctx.SetBodyString("405 Method Not Allowed")
				}

				request := pushnotes.Request{}

				err := easyjson.Unmarshal(ctx.PostBody(), &request)
				if err != nil {
					a.log.Error("failed to unmarshal pushnotes request", "err", err)
					ctx.SetStatusCode(http.StatusBadRequest)
					ctx.SetBodyString("400 Bad Request")
					return
				}

				response, err := pushnotes.Resolve(ctx, a, request)
				if err != nil {
					a.log.Error("failed to resolve pushnotes", "err", err)
					// send error as json
					ctx.SetStatusCode(http.StatusInternalServerError)
					ctx.SetContentType("application/json; charset=utf-8")
					rawBytes, _ := json.Marshal(map[string]string{
						"error": err.Error(),
					})
					ctx.SetBody(rawBytes)
					return
				}

				ctx.SetStatusCode(http.StatusOK)
				ctx.SetContentType("application/json; charset=utf-8")

				rawBytes, err := easyjson.Marshal(response)
				if err != nil {
					a.log.Error("failed to marshal pushnotes response", "err", err)
					ctx.SetStatusCode(http.StatusInternalServerError)
					ctx.SetBodyString("500 Internal Server Error")
				}

				ctx.SetBody(rawBytes)

			default:
				a.handlePages(ctx, path, token)
			}
		},
	}

	err := s.ListenAndServe(":8081")
	if err != nil {
		panic(err)
	}
}
