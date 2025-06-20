package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
	"trip2g/internal/noteloader"
	"trip2g/internal/notfoundtracker"
	"trip2g/internal/nowpayments"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/redirectmanager"
	"trip2g/internal/router"
	"trip2g/internal/userbans"
	"trip2g/internal/usertoken"
	"trip2g/internal/zerologger"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/gqlerror"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/oklog/ulid/v2"
	"github.com/resend/resend-go/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	_ "modernc.org/sqlite"
)

type graphTransactions struct {
	sync.Mutex
	EnvMap map[*app]*sql.Tx
}

type app struct {
	*db.Queries
	*miniostorage.FileStorage

	graphTxs *graphTransactions

	queries *db.Queries
	conn    *sql.DB

	log logger.Logger

	// mail *mailyak.MailYak

	tokenManager *usertoken.Manager

	notFoundTracker *notfoundtracker.Tracker

	redirectManager *redirectmanager.Manager

	hotAuthTokenManager *hotauthtoken.Manager

	config *appconfig.Config

	*userbans.UserBans

	nowpaymentsClient *nowpayments.Client

	purchaseTokenManager *purchasetoken.Manager

	assetsFS    *fasthttp.FS
	assetHashes map[string]string
	assetsMu    sync.Mutex

	liveNoteLoader   *noteloader.Loader
	latestNoteLoader *noteloader.Loader
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

	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "no tx:")))

	nowpaymentsClient, err := nowpayments.NewClient(config.NowpaymentsAPIKey)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	fileStorage, err := miniostorage.New(ctx, config.Storage)
	if err != nil {
		panic(err)
	}

	// mailAddr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	// mailAuth := smtp.PlainAuth(
	// 	"",
	// 	config.SMTPUser,
	// 	config.SMTPPass,
	// 	config.SMTPHost,
	// )

	a := &app{
		Queries: queries,

		FileStorage: fileStorage,

		config: config,

		tokenManager: tokenManager,

		graphTxs: &graphTransactions{
			EnvMap: make(map[*app]*sql.Tx),
		},

		hotAuthTokenManager: hotauthtoken.NewManager([]byte("secret")),

		purchaseTokenManager: purchasetoken.NewManager("trip2g_purchase_token", []byte("secret")),

		log:     log,
		queries: queries,
		conn:    conn,
		// mail:    mailyak.New(mailAddr, mailAuth),

		UserBans: userbans.New(queries),

		nowpaymentsClient: nowpaymentsClient,
	}

	a.redirectManager, err = redirectmanager.New(ctx, a)
	if err != nil {
		panic(fmt.Errorf("failed to create redirect manager: %w", err))
	}

	a.notFoundTracker, err = notfoundtracker.New(ctx, a)
	if err != nil {
		panic(fmt.Errorf("failed to create not found tracker: %w", err))
	}

	a.liveNoteLoader = noteloader.New("live", makeLiveNoteLoaderWrapper(a), a.config.MDLoaderConfig)
	a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), a.config.MDLoaderConfig)

	tokenManager.AddValidator(func(ctx context.Context, data *usertoken.Data) error {
		ban, banErr := a.UserBanByUserID(ctx, int64(data.ID))
		if banErr != nil {
			return fmt.Errorf("failed to get user ban: %w", banErr)
		}

		if ban != nil {
			return gqlerror.Errorf("%s", ban.Reason)
		}

		return nil
	})

	err = a.createOwnerIfNotExists(ctx)
	if err != nil {
		panic(err)
	}

	err = a.loadAllNotes(ctx)
	if err != nil {
		panic(err)
	}

	a.setupAssets()

	fileStorage.OnURLExpiring(func() {
		reloadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		loadErr := a.loadAllNotes(reloadCtx)
		if loadErr != nil {
			log.Error("failed to reload all notes", "error", loadErr)
		}
	})

	a.startServer()
}

func (a *app) createOwnerIfNotExists(ctx context.Context) error {
	if a.config.OwnerEmail == "" {
		return nil // No owner email configured
	}

	user, err := a.Queries.UserByEmail(ctx, a.config.OwnerEmail)
	if err != nil {
		if db.IsNoFound(err) {
			user, err = a.InsertUser(ctx, a.config.OwnerEmail)
			if err != nil {
				return fmt.Errorf("failed to insert owner user: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check if owner exists: %w", err)
		}
	}

	_, err = a.AdminByUserID(ctx, user.ID)
	if err != nil {
		if db.IsNoFound(err) {
			_, insertErr := a.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
			if insertErr != nil {
				return fmt.Errorf("failed to insert owner admin: %w", insertErr)
			}
		} else {
			return fmt.Errorf("failed to check if owner admin exists: %w", err)
		}
	}

	a.log.Info("owner exists", "email", a.config.OwnerEmail)

	return nil
}

func (a *app) SendMail(ctx context.Context, data model.Mail) error {
	client := resend.NewClient(a.config.ResendAPIKey)

	params := &resend.SendEmailRequest{
		From:    a.config.MailFrom,
		To:      []string{data.To},
		Subject: data.Subject,
		Text:    string(data.Plain),
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (a *app) loadAllNotes(ctx context.Context) error {
	startCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := a.liveNoteLoader.Load(startCtx)
	if err != nil {
		return fmt.Errorf("failed to load live notes: %w", err)
	}

	a.log.Info("loaded live notes", "count", len(a.liveNoteLoader.NoteViews().List))

	err = a.latestNoteLoader.Load(startCtx)
	if err != nil {
		return fmt.Errorf("failed to load latest notes: %w", err)
	}

	a.log.Info("loaded latest notes", "count", len(a.latestNoteLoader.NoteViews().List))

	return nil
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

	newEnv := *a //nolint:govet // I will fix this later (copy mutex)
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

	envPtr, ok := req.Env.(*app)
	if !ok {
		return errors.New("failed to cast env to *app")
	}
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

func (a *app) setupAssets() {
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

	// initialize asset hashes map
	a.assetHashes = make(map[string]string)
}

// TODO: read all asset urls from flags.
func (a *app) assetURL(path string) string {
	// Remove leading / if it exists
	assetPath := path
	assetPath = strings.TrimPrefix(assetPath, "/")

	// Remove /assets/ prefix if it exists
	assetPath = strings.TrimPrefix(assetPath, "assets/")

	a.assetsMu.Lock()
	defer a.assetsMu.Unlock()

	// Check if hash already calculated (non-dev mode only)
	if hash, exists := a.assetHashes[assetPath]; exists && !a.config.DevMode {
		return path + "?h=" + hash[:8]
	}

	// Calculate hash on the fly
	content, err := fs.ReadFile(assets.FS, assetPath)
	if err != nil {
		a.log.Debug("asset file not found", "path", assetPath, "original", path)
		return path
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Store hash for future use (non-dev mode only)
	if !a.config.DevMode {
		a.assetHashes[assetPath] = hashStr
	}

	return path + "?h=" + hashStr[:8]
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

	options := mdloader.Options{
		Sources: sources,
		Log:     logger.WithPrefix(a.log, "uploadnoteasset: mdloader:"),
	}

	pages, err := mdloader.Load(options)
	if err != nil {
		return nil, fmt.Errorf("failed to load pages: %w", err)
	}

	return pages.List[0].Assets, nil
}

func (a *app) IDHash(entity string, id int64) string {
	sha256 := sha256.New()
	fmt.Fprintf(sha256, "%s:%d", entity, id)
	return hex.EncodeToString(sha256.Sum(nil))
}

func (a *app) CurrentUserToken(ctx context.Context) (*usertoken.Data, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	return req.UserToken()
}

var ErrNotAdmin = errors.New("unauthorized")

func (a *app) CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	data, err := req.UserToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	if !data.IsAdmin() {
		a.log.Warn("unauthorized access attempt", "user_id", data.ID, "role", data.Role)
		return nil, ErrNotAdmin
	}

	return data, nil
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

func (a *app) PrepareLatestNotes(ctx context.Context) (*model.NoteViews, error) {
	err := a.latestNoteLoader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest notes: %w", err)
	}

	return a.latestNoteLoader.NoteViews(), nil
}

func (a *app) PrepareLiveNotes(ctx context.Context) (*model.NoteViews, error) {
	err := a.liveNoteLoader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load live notes: %w", err)
	}

	return a.liveNoteLoader.NoteViews(), nil
}

func (a *app) LatestNoteViews() *model.NoteViews {
	return a.latestNoteLoader.NoteViews()
}

func (a *app) LiveNoteViews() *model.NoteViews {
	if a.config.LatestLive {
		return a.latestNoteLoader.NoteViews()
	}

	return a.liveNoteLoader.NoteViews()
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

func (a *app) QueueRequestSignInEmail(ctx context.Context, email string, code string) error {
	params := sendsignincode.Params{
		Email: email,
		Code:  code,
	}

	// TODO: add a background jobs
	go func() {
		err := sendsignincode.Resolve(ctx, a, params)
		if err != nil {
			a.log.Error("failed to send sign-in code", "error", err, "email", email)
			return
		}
	}()

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

func (a *app) GenerateAPIKey() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 64

	result := make([]byte, length)

	for i := range length {
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

	for i := range length {
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

func (a *app) NoteByPath(path string) *model.NoteView {
	return a.latestNoteLoader.NoteByPath(path)
}

func (a *app) noteLoaderForRequest(ctx context.Context) *noteloader.Loader {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return a.liveNoteLoader
	}

	token, err := req.UserToken()
	if err != nil || token == nil {
		return a.liveNoteLoader
	}

	// Check if admin wants to see latest version
	if token.IsAdmin() {
		showLatest := string(req.Req.QueryArgs().Peek("version")) == "latest"
		if showLatest {
			return a.latestNoteLoader
		}
	}

	// Default to live for everyone (including admins without ?version=latest)
	return a.liveNoteLoader
}

func (a *app) NoteByPathForRequest(ctx context.Context, path string) *model.NoteView {
	loader := a.noteLoaderForRequest(ctx)
	return loader.NoteByPath(path)
}

func (a *app) NoteViewsForRequest(ctx context.Context) *model.NoteViews {
	loader := a.noteLoaderForRequest(ctx)
	return loader.NoteViews()
}

func (a *app) StorePurchaseToken(ctx context.Context, data model.PurchaseToken) (string, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return "", err
	}

	return a.purchaseTokenManager.Store(req.Req, data)
}

func (a *app) IsDevMode() bool {
	return a.config.DevMode
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

func (a *app) RefreshNotFoundTracker(ctx context.Context) error {
	return a.notFoundTracker.Refresh(ctx)
}

func (a *app) TrackNotFound(path string, ip string) {
	err := a.notFoundTracker.Track(path, ip)
	if err != nil {
		a.log.Error("failed to track not found", "path", path, "error", err)
	}
}

func (a *app) NotifyPuchaseUpdated(email string) {
}

func (a *app) handleCors(ctx *fasthttp.RequestCtx) bool {
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "http://localhost:9081" || origin == "app://obsidian.md" {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Cookie, X-API-Key")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	}

	if ctx.IsOptions() {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return true
	}

	return false
}

func (a *app) handleDebugAPI(ctx *fasthttp.RequestCtx) bool {
	if !a.config.DevMode {
		fmt.Println("skipping debug API in production mode")
		return false
	}

	path := string(ctx.Path())

	if strings.HasPrefix(path, "/debug/nvs/latest") {
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)

		data, err := json.Marshal(a.LatestNoteViews()) //nolint:musttag // debug endpoint
		if err != nil {
			a.log.Error("failed to marshal latest note views", "error", err)
			return true
		}

		ctx.SetBody(data)
		return true
	}

	return false
}

func (a *app) handleAdminAssets(req *appreq.Request, path string) bool {
	if len(a.config.AdminJSURL) > 0 && a.config.AdminJSURL[0] == '/' &&
		strings.HasPrefix(path, a.config.AdminJSURL) {
		userToken, err := req.UserToken()
		if err != nil || userToken == nil {
			req.Req.SetStatusCode(http.StatusUnauthorized)
			req.Req.SetBodyString("401 Unauthorized")
			return true
		}
	}

	return false
}

func (a *app) handlePurchaseTokens(ctx *fasthttp.RequestCtx) bool {
	purchaseTokens, _ := a.purchaseTokenManager.Extract(ctx)
	if len(purchaseTokens) > 0 {
		processed, err := signinbypurchasetoken.Resolve(ctx, a, purchaseTokens)
		if err != nil {
			a.log.Warn("failed to resolve purchase token", "error", err)
		} else if processed {
			err = a.purchaseTokenManager.Delete(ctx)
			if err != nil {
				a.log.Warn("failed to delete purchase token", "error", err)
			}
		}
	}

	return false
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

			if a.handleCors(ctx) {
				return
			}

			if a.handleDebugAPI(ctx) {
				return
			}

			req := appreq.Acquire()
			req.Env = a
			req.Req = ctx
			req.TokenManager = a.tokenManager
			req.StoreInContext() // appreq.FromCtx(ctx)
			defer appreq.Release(req)

			if a.handleAdminAssets(req, path) {
				return
			}

			if strings.HasPrefix(path, "/assets/") {
				fsHandler(ctx)
				return
			}

			if a.handlePurchaseTokens(ctx) {
				return
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

			newPath := a.redirectManager.Match(path)
			if newPath != nil {
				ctx.SetStatusCode(http.StatusFound)
				ctx.Response.Header.Set("Location", *newPath)
				return
			}

			handled, handleErr := rtr.Handle(req)
			if handleErr != nil {
				a.log.Error("failed to handle request", "error", handleErr)
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				ctx.SetBodyString("500 Internal Server Error")
				return
			}

			// TODO: remove this code because rendernotepage handles 404
			if handled {
				a.log.Debug("router handled request", "path", path)
				return
			}

			ctx.SetStatusCode(http.StatusNotFound)
			ctx.SetBodyString("404 Not Found")
		},
	}

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
		MinVersion:     tls.VersionTLS12,
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
