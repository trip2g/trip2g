package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/bqtask/sendsignincode"
	"trip2g/internal/db"
	"trip2g/internal/graph"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/nowpayments"
	"trip2g/internal/router"
	"trip2g/internal/usertoken"
	"trip2g/internal/zerologger"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/gqlerror"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/mikestefanello/backlite"
	backliteui "github.com/mikestefanello/backlite/ui"
	"github.com/oklog/ulid/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	_ "modernc.org/sqlite"
)

type app struct {
	*db.Queries

	mu sync.Mutex

	pages model.NoteViews

	queries *db.Queries
	conn    *sql.DB

	log logger.Logger

	tokenManager *usertoken.Manager
	queueClient  *backlite.Client

	devMode bool

	userBansMap map[int64]db.UserBan
	userBans    []db.UserBan
	userBansMu  sync.Mutex

	nowpaymentsClient *nowpayments.Client
}

func enablePragmas(db *sql.DB) error {
	_, err := db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 3000;
		PRAGMA strict = ON;
	`)
	return err
}

func checkForeignKeys(db *sql.DB) error {
	rows, err := db.Query("PRAGMA foreign_key_check;")
	if err != nil {
		return fmt.Errorf("failed to check foreign keys: %w", err)
	}

	defer rows.Close()

	violationCount := 0

	for rows.Next() {
		var table string
		var rowid int
		var parent string
		var fkid int

		scanErr := rows.Scan(&table, &rowid, &parent, &fkid)
		if scanErr != nil {
			return fmt.Errorf("failed to scan foreign key check: %w", scanErr)
		}

		violationCount++

		fmt.Printf("Foreign key violation in table %s (rowid %d): parent %s, fkid %d\n", table, rowid, parent, fkid)
	}

	if violationCount > 0 {
		return fmt.Errorf("found %d foreign key violations", violationCount)
	}

	return nil
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

	err = enablePragmas(conn)
	if err != nil {
		panic(err)
	}

	err = checkForeignKeys(conn)
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

	devMode := os.Getenv("DEV") == "y"

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	log := zerologger.New(logLevel, devMode)

	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "no tx:")))

	nowpaymentsClient, err := nowpayments.NewClient(os.Getenv("NOWPAYMENTS_API_KEY"))
	if err != nil {
		panic(err)
	}

	a := &app{
		Queries: queries,

		tokenManager: tokenManager,
		queueClient:  queueClient,

		log:     log,
		queries: queries,
		conn:    conn,

		devMode: devMode,

		nowpaymentsClient: nowpaymentsClient,
	}

	tokenManager.AddValidator(func(ctx context.Context, data *usertoken.Data) error {
		ban, err := a.UserBanByUserID(ctx, int64(data.ID))
		if err != nil {
			return fmt.Errorf("failed to get user ban: %w", err)
		}

		if ban != nil {
			return gqlerror.Errorf("%s", ban.Reason)
		}

		return nil
	})

	ctx := context.Background()

	queueClient.Register(sendsignincode.NewQueue(a))
	// queueClient.Start(ctx)

	err = queueClient.Add(sendsignincode.Task{Email: "test@example.com", Code: 313353}).Save()
	if err != nil {
		panic(err)
	}

	_, err = a.PrepareNotes(ctx)
	if err != nil {
		panic(err)
	}

	if os.Getenv("SERVER") == "y" {
		a.startServer()
	}
}

func (a *app) CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
	return a.nowpaymentsClient.CreateInvoice(params)
}

func (a *app) PrepareNotes(ctx context.Context) (model.NoteViews, error) {
	a.log.Info("preparing notes")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	a.mu.Lock()
	defer a.mu.Unlock()

	notes, err := a.queries.AllLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	sources := []mdloader.SourceFile{}

	for _, note := range notes {
		sources = append(sources, mdloader.SourceFile{
			Path:    note.Path,
			PathID:  note.PathID,
			Content: []byte(note.Content),
		})
	}

	pages, err := mdloader.Load(sources, logger.WithPrefix(a.log, "mdloader:"))
	if err != nil {
		return nil, fmt.Errorf("failed to load pages: %w", err)
	}

	a.pages = pages

	return pages, nil
}

func (a *app) AllNotes() model.NoteViews {
	a.mu.Lock()
	defer a.mu.Unlock()

	copy := make(model.NoteViews, len(a.pages))

	for k, v := range a.pages {
		copy[k] = v
	}

	return copy
}

func (a *app) Now() time.Time {
	return time.Now()
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

func (a *app) QueueRequestSignInEmail(_ context.Context, email string, code string) error {
	a.log.Debug("queue sign in email", "email", email, "code", code)
	return nil
}

func (a *app) UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error) {
	a.userBansMu.Lock()
	defer a.userBansMu.Unlock()

	if a.userBansMap == nil {
		userBans, err := a.queries.ListAllUserBans(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get user bans from the db: %w", err)
		}

		a.userBansMap = make(map[int64]db.UserBan, len(a.userBans))
		a.userBans = userBans

		for _, v := range userBans {
			a.userBansMap[v.UserID] = v
		}
	}

	ban, ok := a.userBansMap[userID]
	if !ok {
		return nil, nil
	}

	return &ban, nil
}

func (a *app) ResetBanCache(ctx context.Context) error {
	a.userBansMu.Lock()
	a.userBansMap = nil
	a.userBans = nil
	a.userBansMu.Unlock()

	_, err := a.UserBanByUserID(ctx, 0)

	return err
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

func (a *app) ResetUserToken(ctx context.Context) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return err
	}

	err = req.TokenManager.Delete(req.Req)
	if err != nil {
		return fmt.Errorf("failed to reset token: %w", err)
	}

	return nil
}

func (a *app) GenerateUniqID() string {
	return ulid.Make().String()
}

const purchaseAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (a *app) GeneratePurchaseID() string {
	const length = 8

	result := make([]byte, length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(purchaseAlphabet))))
		if err != nil {
			panic(err)
		}

		result[i] = purchaseAlphabet[n.Int64()]
	}

	return string(result)
}

func (a *app) PublicURL() string {
	return os.Getenv("PUBLIC_URL")
}

func (a *app) NowpaymentsIPNSecret() string {
	return os.Getenv("NOWPAYMENTS_IPN_KEY")
}

var ErrFailedGeneration = errors.New("failed to generate code")

func generateSixDigitCode() (int64, error) {
	for range 100 {
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

func (a *app) CreateSignInCode(ctx context.Context, userID int64) (string, error) {
	code, err := generateSixDigitCode()
	if err != nil {
		return "", err
	}

	if a.devMode {
		code = 111111
	}

	sCode := strconv.Itoa(int(code))

	err = a.InsertSignInCode(ctx, db.InsertSignInCodeParams{
		UserID: userID,
		Code:   sCode,
	})
	if err != nil {
		return "", fmt.Errorf("failed to insert sign-in code: %w", err)
	}

	return sCode, nil
}

func (a *app) NoteByPath(path string) (*model.NoteView, error) {
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
		Compress:           a.devMode,
		SkipCache:          a.devMode,
		AcceptByteRange:    true,

		PathRewrite: func(ctx *fasthttp.RequestCtx) []byte {
			// remove /assets prefix
			return ctx.Path()[7:]
		},
	}

	fs2 := &fasthttp.FS{
		Root:               "./ui",
		IndexNames:         []string{},
		GenerateIndexPages: false,
		Compress:           a.devMode,
		SkipCache:          a.devMode,
		AcceptByteRange:    true,

		PathRewrite: func(ctx *fasthttp.RequestCtx) []byte {
			// remove /ui
			return ctx.Path()[3:]
		},
	}

	fsHandler := fs.NewRequestHandler()
	fs2Handler := fs2.NewRequestHandler()

	rtr := router.New(a)

	resolver := graph.Resolver{DefaultEnv: a}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &resolver}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	graphqlErr := func(err error) graphql.ResponseHandler {
		a.log.Error("graphql error", "error", err)

		return func(ctx context.Context) *graphql.Response {
			return graphql.ErrorResponse(ctx, err.Error())
		}
	}

	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		operationContext := graphql.GetOperationContext(ctx)

		if operationContext.Operation.Operation == ast.Mutation {
			req, err := appreq.FromCtx(ctx)
			if err != nil {
				return graphqlErr(err)
			}

			tx, err := a.conn.BeginTx(ctx, nil)
			if err != nil {
				return graphqlErr(err)
			}

			// TODO: use a pool
			logLabel := fmt.Sprintf("tx %s", operationContext.Operation.Name+":")
			queries := db.New(db.WithLogger(tx, logger.WithPrefix(a.log, logLabel)))

			newEnv := *a
			newEnv.queries = queries
			newEnv.Queries = queries

			// override the context with the new tx env
			req.Env = &newEnv

			rh := next(ctx)

			return func(ctx context.Context) *graphql.Response {
				resp := rh(ctx)

				// А тут интересно, нужно ли отказывать транзакции только в случае ошибок
				// или в случае ErrorPayload так же нужно? Похоже нужно откатывать в случае
				// непредвиденных ошибок и дополнительно вводить специальную ошибку для rollback.
				if len(resp.Errors) > 0 {
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						a.log.Error("failed to rollback transaction", "error", rollbackErr)
					} else {
						a.log.Debug("rolled back transaction")
					}
				} else {
					commitErr := tx.Commit()
					if commitErr != nil {
						a.log.Error("failed to commit transaction", "error", commitErr)
					} else {
						a.log.Debug("committed transaction")
					}
				}

				return resp
			}

			fmt.Println("Mutation", req)
		}

		return next(ctx)
	})

	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(playground.Handler("GraphQL playground", "/graphql"))
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(srv)

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			path := string(ctx.Path())

			if a.devMode {
				origin := string(ctx.Request.Header.Peek("Origin"))
				if origin == "http://localhost:9080" {
					ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
					ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
					ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie")
					ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
				}

				if ctx.IsOptions() {
					ctx.SetStatusCode(fasthttp.StatusNoContent)
					return
				}
			}

			if strings.HasPrefix(path, "/assets/") {
				fsHandler(ctx)
				return
			}

			if strings.HasPrefix(path, "/ui/") {
				fs2Handler(ctx)
				return
			}

			req := appreq.Acquire()
			req.Env = a
			req.Req = ctx
			req.TokenManager = a.tokenManager
			req.StoreInContext() // appreq.FromCtx(ctx)
			defer appreq.Release(req)

			if strings.HasPrefix(path, "/graphql") {
				if string(ctx.Method()) == "GET" {
					playgroundHandler(ctx)
				} else {
					graphqlHandler(ctx)
				}

				return
			}

			// tx, err := a.conn.BeginTx(ctx, nil)
			// if err != nil {
			// 	ctx.SetStatusCode(http.StatusServiceUnavailable)
			// 	ctx.SetBodyString("500 Internal Server Error")
			// 	return
			// }
			//
			// defer tx.Rollback()

			// TODO: use a pool
			// newEnv := *a
			// newEnv.queries = db.New(tx)
			// newEnv.Queries = newEnv.queries

			handled, handleErr := rtr.Handle(req)
			if handleErr != nil {
				a.log.Error("failed to handle request", "error", handleErr)
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				ctx.SetBodyString("500 Internal Server Error")
				return
			}

			// TODO: refactor this
			// if handleErr == nil {
			// 	commitErr := tx.Commit()
			// 	if commitErr != nil {
			// 		a.log.Error("failed to commit transaction", "error", commitErr)
			// 		ctx.SetStatusCode(http.StatusServiceUnavailable)
			// 		ctx.SetBodyString("500 Internal Server Error")
			// 		return
			// 	}
			// } else {
			// 	rollbackErr := tx.Rollback()
			// 	if rollbackErr != nil {
			// 		a.log.Error("failed to rollback transaction", "error", rollbackErr)
			// 		ctx.SetStatusCode(http.StatusServiceUnavailable)
			// 		ctx.SetBodyString("500 Internal Server Error")
			// 		return
			// 	}
			// }

			if handled {
				a.log.Debug("router handled request", "path", path)
				return
			}

			ctx.SetStatusCode(http.StatusNotFound)
			ctx.SetBodyString("404 Not Found")
		},
	}

	go func() {
		mux := http.DefaultServeMux
		backliteui.NewHandler(a.conn).Register(mux)

		server := &http.Server{
			Addr:         ":8082",
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		backErr := server.ListenAndServe()
		if backErr != nil {
			panic(backErr)
		}
	}()

	err := s.ListenAndServe(":8081")
	if err != nil {
		panic(err)
	}
}
