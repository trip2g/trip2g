package main

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=../../views

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/bqtask/sendsignincode"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/router"
	"trip2g/internal/usertoken"
	"trip2g/internal/zerologger"
	"trip2g/views"

	"github.com/mikestefanello/backlite"
	backliteui "github.com/mikestefanello/backlite/ui"
	"github.com/valyala/fasthttp"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	_ "modernc.org/sqlite"
)

type app struct {
	*db.Queries

	mu sync.Mutex

	pages map[string]*mdloader.Page

	queries *db.Queries
	conn    *sql.DB

	log logger.Logger

	tokenManager *usertoken.Manager
	queueClient  *backlite.Client
}

func main() {
	u, _ := url.Parse("sqlite:data.sqlite3")
	dbm := dbmate.New(u)

	err := dbm.CreateAndMigrate()
	if err != nil {
		panic(err)
	}

	conn, err := sql.Open("sqlite", "data.sqlite3?_journal=WAL&_timeout=5000")
	if err != nil {
		panic(err)
	}

	tokenManager := usertoken.NewManager("trip2g_token", []byte("secret"))

	queueConfig := backlite.ClientConfig{
		DB:              conn,
		Logger:          slog.Default(),
		ReleaseAfter:    10 * time.Minute,
		NumWorkers:      10,
		CleanupInterval: time.Hour,
	}

	queueClient, err := backlite.NewClient(queueConfig)
	if err != nil {
		panic(err)
	}

	a := &app{
		Queries: db.New(conn),

		tokenManager: tokenManager,
		queueClient:  queueClient,

		log:     zerologger.New("debug", true),
		queries: db.New(conn),
		conn:    conn,
	}

	ctx := context.Background()

	queueClient.Register(sendsignincode.NewQueue(a))
	queueClient.Start(ctx)

	err = queueClient.Add(sendsignincode.Task{Email: "test@example.com", Code: 313353}).Save()
	if err != nil {
		panic(err)
	}

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

func (a *app) Logger() logger.Logger {
	return a.log
}

func (a *app) QueueRequestSignInEmail(_ context.Context, email string, code int64) error {
	a.log.Debug("queue sign in email", "email", email, "code", code)
	return nil
}

func (a *app) SetupUserToken(ctx context.Context, userID int64) (string, error) {
	data := usertoken.Data{
		ID: int(userID),
	}

	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return "", err
	}

	token, err := req.TokenManager.Store(req.Req, data)
	if err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	return token, nil
}

var ErrFailedGeneration = errors.New("failed to generate code")

func generateSixDigitCode() (int64, error) {
	for i := 0; i < 100; i++ {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			return 0, fmt.Errorf("failed to read random bytes: %w", err)
		}
		n := binary.BigEndian.Uint32(b[:]) % 1000000
		if n >= 100000 {
			return int64(n), nil
		}
	}

	return 0, ErrFailedGeneration
}

func (a *app) CreateSignInCode(ctx context.Context, userID int64) (int64, error) {
	code, err := generateSixDigitCode()
	if err != nil {
		return 0, err
	}

	err = a.queries.InsertSignInCode(ctx, db.InsertSignInCodeParams{
		UserID: userID,
		Code:   code,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to insert sign-in code: %w", err)
	}

	return code, nil
}

func (a *app) PageByPath(path string) (*mdloader.Page, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	page, ok := a.pages[path]
	if !ok {
		return nil, errors.New("page not found")
	}

	return page, nil
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

	rtr := router.New(a, "/api/")

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			path := string(ctx.Path())

			// TODO: only for dev
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")

			if strings.HasPrefix(path, "/assets/") {
				fsHandler(ctx)
				return
			}

			req := appreq.Acquire()
			req.Env = a
			req.Req = ctx
			req.TokenManager = a.tokenManager
			req.StoreInContext()
			defer appreq.Release(req)

			if rtr.Handle(req) {
				a.log.Debug("router handled request", "path", path)
				return
			}

			token, err := req.UserToken()
			if err != nil {
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				ctx.SetBodyString("500 Internal Server Error")
				return
			}

			a.handlePages(ctx, path, token)
		},
	}

	go func() {
		mux := http.DefaultServeMux
		backliteui.NewHandler(a.conn).Register(mux)

		backErr := http.ListenAndServe(":8082", mux)
		if backErr != nil {
			panic(backErr)
		}
	}()

	err := s.ListenAndServe(":8081")
	if err != nil {
		panic(err)
	}
}
