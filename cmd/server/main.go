package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"trip2g/assets"
	"trip2g/internal/acmecache"
	"trip2g/internal/appconfig"
	"trip2g/internal/appreq"
	"trip2g/internal/bqtask/sendsignincode"
	"trip2g/internal/case/signinbypurchasetoken"
	"trip2g/internal/db"
	"trip2g/internal/graph"
	"trip2g/internal/hotauthtoken"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
	"trip2g/internal/nowpayments"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/router"
	"trip2g/internal/userbans"
	"trip2g/internal/usertoken"
	"trip2g/internal/zerologger"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/gqlerror"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/mikestefanello/backlite"
	backliteui "github.com/mikestefanello/backlite/ui"
	"github.com/oklog/ulid/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	_ "modernc.org/sqlite"
)

type currentNVS struct {
	mu  sync.Mutex
	nvs *model.NoteViews
}

type graphTransactions struct {
	sync.Mutex
	EnvMap map[*app]*sql.Tx
}

type app struct {
	*db.Queries
	*miniostorage.FileStorage

	currentNVS *currentNVS
	graphTxs   *graphTransactions

	queries *db.Queries
	conn    *sql.DB

	log logger.Logger

	tokenManager *usertoken.Manager
	queueClient  *backlite.Client

	hotAuthTokenManager *hotauthtoken.Manager

	config *appconfig.Config

	*userbans.UserBans

	nowpaymentsClient *nowpayments.Client

	purchaseTokenManager *purchasetoken.Manager

	purchaseUpdatedMu       sync.Mutex
	purchaseUpdatedHandlers map[string]map[int]func()
	nextPurchaseHandlerID   int

	assetsFS *fasthttp.FS
}

func main() {
	config, err := appconfig.Get()
	if err != nil {
		panic(fmt.Errorf("failed to load configuration: %w", err))
	}

	log := zerologger.New(config.LogLevel, config.DevMode)

	// Setup database
	conn, err := db.Setup(db.SetupConfig{
		DatabaseFile: config.DatabaseFile,
		Logger:       log,
	})
	if err != nil {
		panic(fmt.Errorf("failed to setup database: %w", err))
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

	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "no tx:")))

	nowpaymentsClient, err := nowpayments.NewClient(config.NowpaymentsAPIKey)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	fileStorage, err := miniostorage.New(ctx, config.Storage)
	if err != nil {
		if config.DevMode {
			log.Warn("failed to create minio storage", "error", err)
		} else {
			panic(err)
		}
	}

	a := &app{
		Queries: queries,

		FileStorage: fileStorage,

		config: config,

		tokenManager: tokenManager,
		queueClient:  queueClient,
		currentNVS:   &currentNVS{},

		graphTxs: &graphTransactions{
			EnvMap: make(map[*app]*sql.Tx),
		},

		hotAuthTokenManager: hotauthtoken.NewManager([]byte("secret")),

		purchaseTokenManager: purchasetoken.NewManager("trip2g_purchase_token", []byte("secret")),

		log:     log,
		queries: queries,
		conn:    conn,

		UserBans: userbans.New(queries),

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

	err = a.setupAssets()
	if err != nil {
		panic(err)
	}

	fileStorage.OnURLExpiring(func() {
		_, err = a.PrepareNotes(ctx)
		if err != nil {
			log.Error("failed to prepare notes", "error", err)
		}
	})

	a.startServer()
}

func (a *app) AcquireTxEnvInRequest(ctx context.Context, label string) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get request from context: %w", err)
	}

	tx, err := a.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	logLabel := fmt.Sprintf("tx %s", label+":")
	queries := db.New(db.WithLogger(tx, logger.WithPrefix(a.log, logLabel)))

	newEnv := *a // copy!
	newEnv.queries = queries
	newEnv.Queries = queries

	// override the context with the new tx env
	req.Env = &newEnv

	a.graphTxs.Lock()
	defer a.graphTxs.Unlock()

	a.graphTxs.EnvMap[&newEnv] = tx

	return nil
}

func (a *app) ReleaseTxEnvInRequest(ctx context.Context, commit bool) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get request from context: %w", err)
	}

	a.graphTxs.Lock()
	defer a.graphTxs.Unlock()

	envPtr := req.Env.(*app)
	tx, ok := a.graphTxs.EnvMap[envPtr]
	if !ok {
		return fmt.Errorf("transactioned env not found for request: %v", req.Env)
	}

	// Clean up the map entry regardless of commit/rollback
	delete(a.graphTxs.EnvMap, envPtr)

	if commit {
		commitErr := tx.Commit()
		if commitErr != nil {
			return fmt.Errorf("failed to commit transaction: %w", commitErr)
		}

		return nil
	}

	err = tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

func (a *app) setupAssets() error {
	a.assetsFS = &fasthttp.FS{
		FS:                 assets.FS,
		IndexNames:         []string{},
		GenerateIndexPages: false,
		Compress:           !a.config.DevMode,
		SkipCache:          a.config.DevMode,
		AcceptByteRange:    true,

		PathRewrite: func(ctx *fasthttp.RequestCtx) []byte {
			// remove /assets prefix
			return ctx.Path()[7:]
		},
	}

	return nil
}

// TODO: read all asset urls from flags.
func (a *app) assetURL(path string) string {
	if a.config.DevMode {
		path += fmt.Sprintf("?t=%d", time.Now().UnixMilli())
	}

	return path
}

func (a *app) AdminJSURL() string {
	return a.assetURL(a.config.AdminJSURL)
}

func (a *app) UserJSURLs() []string {
	return []string{
		a.assetURL("/assets/ui/user/-/web.js"),
		a.assetURL("/assets/turbo.js"),
	}
}

func (a *app) UserCSSURLs() []string {
	return []string{a.assetURL("/assets/output.css")}
}

func (a *app) NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error) {
	a.currentNVS.mu.Lock()
	defer a.currentNVS.mu.Unlock()

	noteVersion, err := a.queries.NoteVersionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get note version by ID: %w", err)
	}

	sources := []mdloader.SourceFile{{
		Path:      noteVersion.Path,
		PathID:    noteVersion.PathID,
		VersionID: noteVersion.VersionID,
		Content:   []byte(noteVersion.Content),
	}}

	pages, err := mdloader.Load(sources, logger.WithPrefix(a.log, "uploadnoteasset: mdloader:"))
	if err != nil {
		return nil, fmt.Errorf("failed to load pages: %w", err)
	}

	return pages.List[0].Assets, nil
}

func (a *app) IDHash(entity string, id int64) string {
	sha256 := sha256.New()
	sha256.Write([]byte(fmt.Sprintf("%s:%d", entity, id)))
	return fmt.Sprintf("%x", sha256.Sum(nil))
}

func (a *app) CurrentUserToken(ctx context.Context) (*usertoken.Data, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	return req.UserToken()
}

func (a *app) GenerateHotAuthToken(_ context.Context, data model.HotAuthToken) (string, error) {
	return a.hotAuthTokenManager.NewToken(data)
}

func (a *app) ParseHotAuthToken(_ context.Context, token string) (*model.HotAuthToken, error) {
	return a.hotAuthTokenManager.ParseToken(token)
}

func (a *app) CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
	return a.nowpaymentsClient.CreateInvoice(params)
}

func (a *app) PrepareNotes(ctx context.Context) (*model.NoteViews, error) {
	var env *app

	req, err := appreq.FromCtx(ctx)
	if err == nil {
		reqEnv, ok := req.Env.(*app)
		if ok {
			env = reqEnv
		}
	}

	a.log.Info("preparing notes", "tx", env != nil)

	if env == nil {
		env = a
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	notes, err := env.AllLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	assets, err := env.AllLatestNoteAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get note assets: %w", err)
	}

	assetMap := make(map[int64]map[string]string)

	for _, asset := range assets {
		noteMap, ok := assetMap[asset.VersionID]
		if !ok {
			noteMap = make(map[string]string)
			assetMap[asset.VersionID] = noteMap
		}

		assetURL, err := a.NoteAssetURL(ctx, asset.NoteAsset)
		if err != nil {
			return nil, fmt.Errorf("failed to get note asset URL: %w", err)
		}

		a.log.Debug("note asset URL", "path", asset.Path, "url", assetURL)

		noteMap[asset.Path] = assetURL
	}

	sources := []mdloader.SourceFile{}

	for _, note := range notes {
		sources = append(sources, mdloader.SourceFile{
			Path:      note.Path,
			PathID:    note.PathID,
			VersionID: note.VersionID,
			Content:   []byte(note.Content),
			Assets:    assetMap[note.VersionID],
		})
	}

	nvs, err := mdloader.Load(sources, logger.WithPrefix(a.log, "mdloader:"))
	if err != nil {
		return nil, fmt.Errorf("failed to load pages: %w", err)
	}

	a.currentNVS.mu.Lock()
	a.currentNVS.nvs = nvs
	a.currentNVS.mu.Unlock()

	return nvs, nil
}

func (a *app) AllNotes() *model.NoteViews {
	a.currentNVS.mu.Lock()
	defer a.currentNVS.mu.Unlock()
	c := a.currentNVS.nvs.Copy()

	return c
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

func (a *app) SetupUserToken(ctx context.Context, userID int64) (string, error) {
	role := "user"

	_, err := a.queries.AdminByUserID(ctx, userID)
	if err != nil {
		if !db.IsNoFound(err) {
			return "", fmt.Errorf("failed to get admin by user ID: %w", err)
		}
	} else {
		role = "admin"
	}

	data := usertoken.Data{
		ID:   int(userID),
		Role: role,
	}

	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return "", err
	}

	storeData, err := req.TokenManager.Store(req.Req, data)
	if err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	req.SetUserToken(&storeData.Data)

	return storeData.JWT, nil
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

func (a *app) GenerateApiKey() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 64

	result := make([]byte, length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			panic(err)
		}

		result[i] = alphabet[n.Int64()]
	}

	return string(result)
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
	return a.config.PublicURL
}

func (a *app) NowpaymentsIPNSecret() string {
	return a.config.NowpaymentsIPNKey
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

	if a.config.DevMode {
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
	a.currentNVS.mu.Lock()
	defer a.currentNVS.mu.Unlock()

	page, ok := a.currentNVS.nvs.Map[path]
	if !ok {
		return nil, errors.New("page not found")
	}

	return page, nil
}

func (a *app) StorePurchaseToken(ctx context.Context, data model.PurchaseToken) (string, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return "", err
	}

	return a.purchaseTokenManager.Store(req.Req, data)
}

func (a *app) ExtractPurchaseTokenIDs(ctx context.Context) ([]string, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	tokens, err := a.purchaseTokenManager.Extract(req.Req)
	if err != nil {
		return nil, fmt.Errorf("failed to extract purchase tokens: %w", err)
	}

	ids := make([]string, len(tokens))

	for i, token := range tokens {
		ids[i] = token.PurchaseID
	}

	return ids, nil
}

func (a *app) AssetVersion() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

func (a *app) NotifyPuchaseUpdated(email string) {
}

func (a *app) startServer() {
	fsHandler := a.assetsFS.NewRequestHandler()

	rtr := router.New(a)

	// graphql
	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(playground.Handler("GraphQL playground", "/graphql"))
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(graph.NewHandler(a))

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			path := string(ctx.Path())

			origin := string(ctx.Request.Header.Peek("Origin"))
			if origin == "http://localhost:9081" || origin == "app://obsidian.md" {
				ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
				ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
				ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Cookie, X-API-Key")
				ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			}

			if ctx.IsOptions() {
				ctx.SetStatusCode(fasthttp.StatusNoContent)
				return
			}

			if a.config.DevMode {
				if strings.HasPrefix(path, "/debug/nvs") {
					ctx.SetContentType("application/json")
					ctx.SetStatusCode(fasthttp.StatusOK)

					data, _ := json.Marshal(a.AllNotes())
					ctx.SetBody(data)
					return
				}
			}

			req := appreq.Acquire()
			req.Env = a
			req.Req = ctx
			req.TokenManager = a.tokenManager
			req.StoreInContext() // appreq.FromCtx(ctx)
			defer appreq.Release(req)

			// hide admin JS from public
			if len(a.config.AdminJSURL) > 0 && a.config.AdminJSURL[0] == '/' && strings.HasPrefix(path, a.config.AdminJSURL) {
				userToken, err := req.UserToken()
				if err != nil || userToken == nil {
					ctx.SetStatusCode(http.StatusUnauthorized)
					ctx.SetBodyString("401 Unauthorized")
					return
				}
			}

			if strings.HasPrefix(path, "/assets/") {
				fsHandler(ctx)
				return
			}

			// handle purchase tokens from cookies
			purchaseTokens, _ := a.purchaseTokenManager.Extract(ctx)
			if len(purchaseTokens) > 0 {
				processed, err := signinbypurchasetoken.Resolve(ctx, a, purchaseTokens)
				if err != nil {
					a.log.Warn("failed to resolve purchase token", "error", err)
				} else if processed {
					a.purchaseTokenManager.Delete(ctx)
				}
			}

			// handle hot auth token from ?hot=...
			// hatAuthToken := string(ctx.QueryArgs().Peek("hat")) // TODO: use b2s
			// if hatAuthToken != "" {
			// 	hatErr := signinbyhat.Resolve(ctx, a, hatAuthToken)
			// 	if hatErr != nil {
			// 		a.log.Warn("failed to resolve hot auth token", "error", hatErr)
			// 	}
			//
			// 	parsedURL, err := url.Parse(string(ctx.Request.Header.RequestURI()))
			// 	if err != nil {
			// 		a.log.Warn("failed to parse URL", "error", err)
			// 		ctx.SetStatusCode(http.StatusBadRequest)
			// 		return
			// 	}
			//
			// 	query := parsedURL.Query()
			// 	query.Del("hat")
			// 	parsedURL.RawQuery = query.Encode()
			//
			// 	ctx.Redirect(parsedURL.String(), http.StatusFound)
			// 	return
			// }

			if strings.HasPrefix(path, "/graphql") {
				if string(ctx.Method()) == "GET" {
					playgroundHandler(ctx)
				} else {
					graphqlHandler(ctx)
				}

				return
			}

			handled, handleErr := rtr.Handle(req)
			if handleErr != nil {
				a.log.Error("failed to handle request", "error", handleErr)
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				ctx.SetBodyString("500 Internal Server Error")
				return
			}

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

	if len(a.config.AcmeDomains) == 0 {
		err := s.ListenAndServe(a.config.ListenAddr)
		if err != nil {
			panic(err)
		}

		return
	}

	allowedDomains := make(map[string]struct{}, len(a.config.AcmeDomains))

	for _, domain := range a.config.AcmeDomains {
		a.log.Info("adding domain to ACME", "domain", domain)
		allowedDomains[domain] = struct{}{}
	}

	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  acmecache.New(a),
		HostPolicy: func(ctx context.Context, host string) error {
			_, ok := allowedDomains[host]
			if ok {
				return nil
			}

			return fmt.Errorf("unauthorized domain: %s", host)
		},
	}

	tlsConfig := &tls.Config{
		GetCertificate: certManager.GetCertificate,
		NextProtos:     []string{"http/1.1", acme.ALPNProto},
	}

	ln, err := net.Listen("tcp4", ":443") // #nosec G102
	if err != nil {
		panic(err)
	}

	lnTLS := tls.NewListener(ln, tlsConfig)

	err = fasthttp.Serve(lnTLS, s.Handler)
	if err != nil {
		panic(err)
	}
}
