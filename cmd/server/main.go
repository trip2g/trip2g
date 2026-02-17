package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"trip2g/assets"
	"trip2g/internal/acmecache"
	"trip2g/internal/appconfig"
	"trip2g/internal/appreq"
	"trip2g/internal/auditlogger"
	"trip2g/internal/boosty"
	"trip2g/internal/boostyjobs"
	"trip2g/internal/case/backjob/deliverchangewebhook"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/case/backjob/extractnotionpages"
	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/case/backjob/importtelegramchannel"
	"trip2g/internal/case/backjob/sendsignincode"
	"trip2g/internal/case/backjob/sendtelegramaccountmessage"
	"trip2g/internal/case/backjob/sendtelegramaccountpost"
	"trip2g/internal/case/backjob/sendtelegrammessage"
	"trip2g/internal/case/backjob/sendtelegrampost"
	"trip2g/internal/case/backjob/updateallaccounttelegrampublishposts"
	"trip2g/internal/case/backjob/updateallchattelegrampublishposts"
	"trip2g/internal/case/backjob/updatetelegramaccountmessage"
	"trip2g/internal/case/backjob/updatetelegramaccountpost"
	"trip2g/internal/case/backjob/updatetelegrammessage"
	"trip2g/internal/case/backjob/updatetelegrampost"
	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/getboostyuser"
	"trip2g/internal/case/getpatreonuser"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/handletgpublishviews"
	"trip2g/internal/case/insertnote"
	"trip2g/internal/case/listactiveusersubgraphs"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/case/signinbypurchasetoken"
	"trip2g/internal/case/signinbytgauthtoken"
	"trip2g/internal/case/updatesubgraphs"
	"trip2g/internal/case/uploadnoteasset"
	"trip2g/internal/cronjobs"
	"trip2g/internal/dataencryption"
	"trip2g/internal/db"
	"trip2g/internal/fastgql"
	"trip2g/internal/features"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/gitapi"
	"trip2g/internal/githubauth"
	"trip2g/internal/googleauth"
	"trip2g/internal/graph"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/hotauthtoken"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
	"trip2g/internal/notfoundtracker"
	"trip2g/internal/notion"
	"trip2g/internal/notiontypes"
	"trip2g/internal/nowpayments"
	"trip2g/internal/openai"
	"trip2g/internal/patreon"
	"trip2g/internal/patreonjobs"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/redirectmanager"
	"trip2g/internal/router"
	"trip2g/internal/rssfeed"
	"trip2g/internal/simplebackup"
	"trip2g/internal/tgauthtoken"
	"trip2g/internal/tgbots"
	"trip2g/internal/tgtd"
	"trip2g/internal/userbans"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
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

type txEnvKeyType struct{}

//nolint:gochecknoglobals // Context key for transactional env
var txEnvKey = txEnvKeyType{}

type graphTransactions struct {
	sync.Mutex
	EnvMap map[*app]*sql.Tx
}

type app struct {
	*db.Queries
	*db.WriteQueries

	*miniostorage.FileStorage
	*dataencryption.Manager
	*patreonjobs.PatreonJobs
	*boostyjobs.BoostyJobs
	*tgbots.TgBots
	*cronjobs.CronJobs
	*sendsignincode.SendSignInCodeJob
	*sendtelegrampost.SendTelegramPostJob
	*updatetelegrampost.UpdateTelegramPostJob
	*sendtelegrammessage.SendTelegramMessageJob
	*updatetelegrammessage.UpdateTelegramMessageJob
	*sendtelegramaccountmessage.SendTelegramAccountMessageJob
	*updatetelegramaccountmessage.UpdateTelegramAccountMessageJob
	*sendtelegramaccountpost.SendTelegramAccountPostJob
	*updatetelegramaccountpost.UpdateTelegramAccountPostJob
	*importtelegramchannel.ImportTelegramChannelJob
	*extractnotionpages.ExtractNotionPagesJob
	*updateallchattelegrampublishposts.UpdateAllChatTelegramPublishPostsJob
	*updateallaccounttelegrampublishposts.UpdateAllAccountTelegramPublishPostsJob
	GenerateNoteVersionEmbeddingJob *generatenoteversionembedding.Job
	*deliverchangewebhook.DeliverChangeWebhookJob
	*delivercronwebhook.DeliverCronWebhookJob
	webhookHTTPClient *fasthttp.Client

	webhookTestCalls []webhookTestCall
	webhookTestMu    sync.Mutex

	openaiClient *openai.Client

	sigChan     chan os.Signal
	shutdownCtx context.Context
	shutdown    context.CancelFunc
	stopped     atomic.Bool
	ctx         context.Context

	graphTxs *graphTransactions

	queries   *db.Queries
	conn      *sql.DB
	writeConn *sql.DB

	currentTx *sql.Tx

	log logger.Logger

	auditLogger logger.Logger

	globalQueue              *appQueue
	telegramBotAPIQueue      *appQueue
	telegramAccountAPIQueue  *appQueue
	telegramTaskQueue        *appQueue
	telegramLongRunningQueue *appQueue

	// mail *mailyak.MailYak

	tokenManager *usertoken.Manager

	notFoundTracker *notfoundtracker.Tracker

	redirectManager *redirectmanager.Manager

	hotAuthTokenManager *hotauthtoken.Manager
	tgAuthTokenManager  *tgauthtoken.Manager

	notionClientManager *notion.ClientManager

	config *appconfig.Config

	*userbans.UserBans

	nowpaymentsClient *nowpayments.Client

	purchaseTokenManager *purchasetoken.Manager

	assetsFS    *fasthttp.FS
	assetHashes map[string]string
	assetsMu    sync.Mutex

	timeLocation      *time.Location
	timeLocationMutex sync.Mutex

	liveNoteLoader         *noteloader.Loader
	latestNoteLoader       *noteloader.Loader
	frontmatterPatchLoader *frontmatterpatch.Loader

	patreonClientManager *patreon.ClientManager
	boostyClientManager  *boosty.ClientManager

	gitAPI *gitapi.API

	appQueues map[string]*appQueue

	simpleBackup *simplebackup.Manager

	telegramAuthManager *tgtd.AuthManager
}

func initDBs(config *appconfig.Config, log logger.Logger) (*sql.DB, *sql.DB) {
	dbConfig := db.SetupConfig{
		DatabaseFile: config.DatabaseFile,
		Logger:       log,
		LogQueries:   config.LogQueries,
		ReadOnly:     true,
		DevMode:      config.DevMode,
	}

	conn, err := db.Setup(dbConfig)
	if err != nil {
		panic(fmt.Errorf("failed to setup database: %w", err))
	}

	dbConfig.ReadOnly = false
	dbConfig.CheckStatus = true

	writeConn, err := db.Setup(dbConfig)
	if err != nil {
		panic(fmt.Errorf("failed to setup database: %w", err))
	}

	return conn, writeConn
}

func initDataEncryptionManager(config *appconfig.Config) *dataencryption.Manager {
	manager, err := dataencryption.NewManager(config.DataEncryption)
	if err != nil {
		panic(fmt.Errorf("failed to create data encryption manager: %w", err))
	}
	return manager
}

func main() {
	config, err := appconfig.Get()
	if err != nil {
		panic(fmt.Errorf("failed to load configuration: %w", err))
	}

	log := zerologger.New(config.LogLevel, config.DevMode)

	// RESTORE PHASE (Pre-DB Init) - if simple backup enabled
	if config.SimpleBackup.Enabled {
		restoreBackup(log, config)
	}

	conn, writeConn := initDBs(config, log)

	tokenManager := usertoken.NewManager(config.UserToken)
	// use USER_TOKEN_INSECURE instead
	// tokenManager.SetInsecure(config.DevMode) // for k6

	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "read: no tx:")))
	writeQueries := db.NewWriteQueries(db.WithLogger(writeConn, logger.WithPrefix(log, "write: no tx:")))

	nowpaymentsClient, err := nowpayments.NewClient(config.NowpaymentsAPIKey, log)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileStorage, err := miniostorage.New(ctx, config.Storage)
	if err != nil {
		panic(err)
	}

	log.Info("using storage prefix", "prefix", config.Storage.Prefix)

	// mailAddr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	// mailAuth := smtp.PlainAuth(
	// 	"",
	// 	config.SMTPUser,
	// 	config.SMTPPass,
	// 	config.SMTPHost,
	// )

	a := &app{
		Queries:      queries,
		WriteQueries: writeQueries,

		FileStorage: fileStorage,
		Manager:     initDataEncryptionManager(config),

		config: config,

		tokenManager: tokenManager,

		graphTxs: &graphTransactions{
			EnvMap: make(map[*app]*sql.Tx),
		},

		hotAuthTokenManager: hotauthtoken.NewManager(config.HotAuthToken),
		tgAuthTokenManager:  tgauthtoken.NewManager(config.TgAuthToken),

		purchaseTokenManager: purchasetoken.NewManager(config.PurchaseToken),

		log:     log,
		queries: queries,
		conn:    conn,
		// mail:    mailyak.New(mailAddr, mailAuth),

		writeConn: writeConn,

		UserBans: userbans.New(queries),

		nowpaymentsClient: nowpaymentsClient,
	}

	a.ctx = ctx
	a.sigChan = make(chan os.Signal, 1)
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)

	a.shutdownCtx, a.shutdown = context.WithCancel(context.Background())

	a.auditLogger = auditlogger.New(ctx, a, a.config.AuditLog)

	a.initPatreon(ctx)
	a.initBoosty(ctx)

	a.globalQueue = a.createQueue(ctx, "global_jobs", QueueOpts{
		Limit:        5,
		PollInterval: time.Second * 3,
	})

	err = a.initTelegramDeps(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to initialize telegram dependencies: %w", err))
	}

	// Initialize OpenAI client if vector search is enabled
	if a.config.Features.VectorSearch.Enabled {
		a.openaiClient = openai.New(
			os.Getenv("OPENAI_API_KEY"),
			a.config.Features.VectorSearch.Model,
		)
	}

	a.initJobs(ctx)

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
	a.frontmatterPatchLoader = frontmatterpatch.NewLoader(a)

	a.gitAPI, err = gitapi.New(ctx, a.config.GitAPI, a)
	if err != nil {
		panic(err)
	}

	a.notionClientManager = notion.NewClientManager(a, a.config.Notion)

	// Initialize simple backup manager if enabled
	if config.SimpleBackup.Enabled {
		a.simpleBackup = simplebackup.New(a, config.DatabaseFile)
		log.Info("simple backup manager initialized")
	}

	err = a.createOwnerIfNotExists(ctx)
	if err != nil {
		panic(err)
	}

	err = a.loadAllNotes(ctx, noteloader.LoadOptions{})
	if err != nil {
		panic(err)
	}

	a.setupAssets()
	a.setTokenValidator()
	a.setFileStorageExpiringCallback()

	a.globalQueue.start()
	a.telegramTaskQueue.start()
	a.telegramBotAPIQueue.start()
	a.telegramAccountAPIQueue.start()
	a.telegramLongRunningQueue.start()

	a.startServer()
}

func restoreBackup(log logger.Logger, config *appconfig.Config) {
	log.Info("simple backup enabled, checking for restore")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create temporary storage client for restore
	restoreStorage, restoreErr := miniostorage.New(ctx, config.Storage)
	if restoreErr != nil {
		log.Error("FATAL: failed to init storage for restore", "error", restoreErr)
		panic(fmt.Errorf("failed to init storage for restore: %w", restoreErr))
	}

	// Create restore environment adapter
	restoreEnv := &restoreEnvAdapter{
		FileStorage: restoreStorage,
		log:         log,
	}

	restoreMgr := simplebackup.New(restoreEnv, config.DatabaseFile)

	startupErr := restoreMgr.RestoreOnStartup(ctx)
	if startupErr != nil {
		log.Error("FATAL: failed to restore database", "error", startupErr)
		panic(fmt.Errorf("failed to restore database: %w", startupErr))
	}
}

func (a *app) initJobs(ctx context.Context) {
	a.SendTelegramMessageJob = sendtelegrammessage.New(a)
	a.UpdateTelegramMessageJob = updatetelegrammessage.New(a)
	a.SendTelegramAccountMessageJob = sendtelegramaccountmessage.New(a)
	a.UpdateTelegramAccountMessageJob = updatetelegramaccountmessage.New(a)
	a.SendTelegramAccountPostJob = sendtelegramaccountpost.New(a)
	a.UpdateTelegramAccountPostJob = updatetelegramaccountpost.New(a)
	a.ImportTelegramChannelJob = importtelegramchannel.New(a)

	a.SendSignInCodeJob = sendsignincode.New(a)
	a.SendTelegramPostJob = sendtelegrampost.New(a)
	a.UpdateTelegramPostJob = updatetelegrampost.New(a)
	a.ExtractNotionPagesJob = extractnotionpages.New(a)
	a.UpdateAllChatTelegramPublishPostsJob = updateallchattelegrampublishposts.New(a)
	a.UpdateAllAccountTelegramPublishPostsJob = updateallaccounttelegrampublishposts.New(a)
	a.GenerateNoteVersionEmbeddingJob = generatenoteversionembedding.New(a)
	a.DeliverChangeWebhookJob = deliverchangewebhook.New(a)
	a.DeliverCronWebhookJob = delivercronwebhook.New(a)
	a.webhookHTTPClient = webhookutil.NewClient()

	var err error

	a.CronJobs, err = cronjobs.New(ctx, a, getCronJobConfigs(a))
	if err != nil {
		panic(fmt.Errorf("failed to create cron jobs: %w", err))
	}
}

func (a *app) initPatreon(ctx context.Context) {
	a.patreonClientManager = patreon.NewClientManager(a)

	var err error

	a.PatreonJobs, err = patreonjobs.New(ctx, a, a.config.PatreonJobsConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create Patreon IO: %w", err))
	}
}

func (a *app) initBoosty(ctx context.Context) {
	a.boostyClientManager = boosty.NewClientManager(a)

	var err error

	a.BoostyJobs, err = boostyjobs.New(ctx, a, a.config.BoostyJobsConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create Boosty IO: %w", err))
	}
}

func (a *app) setTokenValidator() {
	a.tokenManager.AddValidator(func(ctx context.Context, data *usertoken.Data) error {
		ban, banErr := a.UserBanByUserID(ctx, int64(data.ID))
		if banErr != nil {
			return fmt.Errorf("failed to get user ban: %w", banErr)
		}

		if ban != nil {
			return gqlerror.Errorf("%s", ban.Reason)
		}

		return nil
	})
}

func (a *app) setFileStorageExpiringCallback() {
	a.FileStorage.OnURLExpiring(func() {
		a.log.Info("presigned URLs expiring, reloading notes")

		reloadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		options := noteloader.LoadOptions{ForceRefreshURLs: true}
		loadErr := a.loadAllNotes(reloadCtx, options)
		if loadErr != nil {
			a.log.Error("failed to reload all notes", "error", loadErr)
		} else {
			a.log.Info("notes reloaded successfully")
		}
	})
}

func (a *app) ApplyGitChanges(ctx context.Context) ([]string, error) {
	return a.gitAPI.ApplyChanges(ctx)
}

func (a *app) GitCommit() string {
	return GitCommit
}

func (a *app) SiteTitleTemplate() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.SiteConfig(ctx).SiteTitleTemplate
}

func (a *app) SiteConfig(ctx context.Context) model.SiteConfig {
	cfg := model.SiteConfig{
		SiteTitleTemplate: "%s",
		Timezone:          "UTC",
		RobotsTxt:         "opened",
		ShowDraftVersions: true,
		EnableRSS:         true,
	}

	strings, err := a.AllLatestConfigStrings(ctx)
	if err == nil {
		for _, s := range strings {
			switch s.ValueID {
			case "site_title_template":
				cfg.SiteTitleTemplate = s.Value
			case "timezone":
				cfg.Timezone = s.Value
			case "default_layout":
				cfg.DefaultLayout = s.Value
			case "robots_txt":
				cfg.RobotsTxt = s.Value
			}
		}
	}

	bools, err := a.AllLatestConfigBools(ctx)
	if err == nil {
		for _, b := range bools {
			if b.ValueID == "show_draft_versions" {
				cfg.ShowDraftVersions = b.Value
			}
			if b.ValueID == "enable_rss" {
				cfg.EnableRSS = b.Value
			}
		}
	}

	return cfg
}

func (a *app) TimeLocation() *time.Location {
	a.timeLocationMutex.Lock()
	defer a.timeLocationMutex.Unlock()

	if a.timeLocation != nil {
		return a.timeLocation
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timezone := a.SiteConfig(ctx).Timezone

	var loadErr error

	a.timeLocation, loadErr = time.LoadLocation(timezone)
	if loadErr != nil {
		a.timeLocation = time.UTC
		a.log.Error("failed to load timezone location", "timezone", timezone, "error", loadErr)
	}

	return a.timeLocation
}

func (a *app) NotionClientByIntegrationID(integrationID int64) notiontypes.Client {
	client, err := a.notionClientManager.Get(a.ctx, a, integrationID)
	if err != nil {
		a.log.Error("failed to get notion client by integration ID", "integrationID", integrationID, "error", err)
		return nil
	}

	return client
}

func (a *app) PushNotes(ctx context.Context, input graphmodel.PushNotesInput) (graphmodel.PushNotesOrErrorPayload, error) {
	return pushnotes.Resolve(ctx, a, input)
}

func (a *app) UploadNoteAsset(ctx context.Context, input graphmodel.UploadNoteAssetInput) (graphmodel.UploadNoteAssetOrErrorPayload, error) {
	return uploadnoteasset.Resolve(ctx, a, input)
}

func (a *app) InsertNote(ctx context.Context, note model.RawNote) (int64, error) {
	return insertnote.Resolve(ctx, a, note)
}

func (a *app) InsertUncommittedPath(ctx context.Context, notePathID int64) error {
	return a.WriteQueries.InsertUncommittedPath(ctx, notePathID)
}

func (a *app) ListUncommittedPaths(ctx context.Context) ([]int64, error) {
	return a.Queries.ListUncommittedPaths(ctx)
}

func (a *app) ClearUncommittedPaths(ctx context.Context) error {
	return a.WriteQueries.ClearUncommittedPaths(ctx)
}

func (a *app) createOwnerIfNotExists(ctx context.Context) error {
	if a.config.OwnerEmail == "" {
		a.log.Warn("no owner email configured, skipping owner creation")
		return nil // No owner email configured
	}

	user, err := a.Queries.UserByEmail(ctx, a.config.OwnerEmail)
	if err != nil {
		if db.IsNoFound(err) {
			params := db.InsertUserWithEmailParams{
				Email:      a.config.OwnerEmail,
				CreatedVia: "bootstrap",
			}
			user, err = a.InsertUserWithEmail(ctx, params)
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

func (a *app) PatreonClientByID(ctx context.Context, credentialsID int64) (patreon.Client, error) {
	env, err := getEnvOrDefault[patreon.ClientManagerEnv](ctx, a)
	if err != nil {
		return nil, fmt.Errorf("failed to get Patreon client manager environment: %w", err)
	}

	client, err := a.patreonClientManager.Get(ctx, env, credentialsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Patreon client: %w", err)
	}

	return client, nil
}

func (a *app) UpdateBoostyCredentials(ctx context.Context, args db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
	a.boostyClientManager.Reset(ctx, args.ID)

	return a.WriteQueries.UpdateBoostyCredentials(ctx, args)
}

func (a *app) BoostyClientByCredentialsID(ctx context.Context, credentialID int64) (boosty.Client, error) {
	env, err := getEnvOrDefault[boosty.ClientManagerEnv](ctx, a)
	if err != nil {
		return nil, err
	}

	return a.boostyClientManager.Get(ctx, env, credentialID)
}

func (a *app) ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error) {
	// TODO: add caching for this method
	return listactiveusersubgraphs.Resolve(ctx, a, userID)
}

func (a *app) SendMail(ctx context.Context, data model.Mail) error {
	if a.config.DevMode {
		a.log.Info("send email", "to", data.To, "subject", data.Subject, "plain", string(data.Plain))
		return nil
	}

	client := resend.NewClient(a.config.ResendAPIKey)

	params := &resend.SendEmailRequest{
		From:    a.config.MailFrom,
		To:      []string{data.To},
		Subject: data.Subject,
		Text:    string(data.Plain),
	}

	_, err := client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (a *app) CalculateSha256(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// LoadFrontmatterPatches implements noteloader.Env interface.
// It delegates to the frontmatterPatchLoader which handles loading and compilation.
func (a *app) LoadFrontmatterPatches(ctx context.Context) ([]frontmatterpatch.CompiledPatch, error) {
	return a.frontmatterPatchLoader.LoadFrontmatterPatches(ctx)
}

func (a *app) loadAllNotes(ctx context.Context, options noteloader.LoadOptions) error {
	startCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Patches are now loaded automatically by noteloader.Load()
	err := a.liveNoteLoader.Load(startCtx, options)
	if err != nil {
		return fmt.Errorf("failed to load live notes: %w", err)
	}

	a.log.Info("loaded live notes", "count", len(a.liveNoteLoader.NoteViews().List))

	err = a.latestNoteLoader.Load(startCtx, options)
	if err != nil {
		return fmt.Errorf("failed to load latest notes: %w", err)
	}

	a.log.Info("loaded latest notes", "count", len(a.latestNoteLoader.NoteViews().List))

	return nil
}

func (a *app) CurrentTx() *sql.Tx {
	return a.currentTx
}

// WithTransaction runs the given function within a database transaction.
// fn should return true to commit the transaction, false to rollback.
func (a *app) WithTransaction(ctx context.Context, fn func(context.Context, *app) (bool, error)) error {
	// not sure but I guess transactions should run on writeConn
	tx, err := a.writeConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to BeginTx: %w", err)
	}

	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			a.log.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	queries := db.NewWriteQueries(db.WithLogger(tx, logger.WithPrefix(a.log, "tx")))

	newEnv := *a //nolint:govet // I will fix this later (copy mutex)
	newEnv.queries = queries.Queries
	newEnv.Queries = queries.Queries
	newEnv.WriteQueries = queries
	newEnv.currentTx = tx

	// Store transactional env in context so background jobs can access it
	txCtx := context.WithValue(ctx, txEnvKey, &newEnv)

	commit, err := fn(txCtx, &newEnv)
	if commit {
		commitErr := tx.Commit()
		if commitErr != nil {
			return fmt.Errorf("failed to commit transaction: %w", commitErr)
		}
	} else {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			a.log.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}

	return err
}

func (a *app) AcquireTxEnvInRequest(ctx context.Context, label string) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get request from context: %w", err)
	}

	tx, err := a.writeConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	logLabel := fmt.Sprintf("tx %s", label+":")
	queries := db.NewWriteQueries(db.WithLogger(tx, logger.WithPrefix(a.log, logLabel)))

	newEnv := *a //nolint:govet // I will fix this later (copy mutex)
	newEnv.queries = queries.Queries
	newEnv.Queries = queries.Queries
	newEnv.WriteQueries = queries
	newEnv.currentTx = tx

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
		a.log.Error("failed to rollback transaction", "error", err)
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
	}
}

func (a *app) UserCSSURLs() []string {
	return []string{a.assetURL("/assets/output.css")}
}

func (a *app) LoadNoteViewByVersionID(ctx context.Context, id int64) (*model.NoteView, error) {
	wrapper := makeSingleNoteLoaderWrapper(a, id)
	loader := noteloader.New("single", wrapper, a.config.MDLoaderConfig)

	err := loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true})
	if err != nil {
		return nil, fmt.Errorf("failed to load note version %d: %w", id, err)
	}

	return loader.NoteViews().List[0], nil
}

func (a *app) UpsertAPIKeyLogAction(ctx context.Context, name string) error {
	if txEnv, ok := ctx.Value(txEnvKey).(*app); ok && txEnv.currentTx != nil {
		return txEnv.WriteQueries.UpsertAPIKeyLogAction(ctx, name)
	}
	return a.WriteQueries.UpsertAPIKeyLogAction(ctx, name)
}

func (a *app) UpsertAPIKeyLogIP(ctx context.Context, ip string) error {
	if txEnv, ok := ctx.Value(txEnvKey).(*app); ok && txEnv.currentTx != nil {
		return txEnv.WriteQueries.UpsertAPIKeyLogIP(ctx, ip)
	}
	return a.WriteQueries.UpsertAPIKeyLogIP(ctx, ip)
}

func (a *app) NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error) {
	wrapper := makeSingleNoteLoaderWrapper(a, id)
	loader := noteloader.New("single", wrapper, a.config.MDLoaderConfig)

	err := loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true})
	if err != nil {
		return nil, fmt.Errorf("failed to load note version %d: %w", id, err)
	}

	nvs := loader.NoteViews()
	layouts := loader.Layouts()

	res := map[string]struct{}{}

	if len(layouts.Map) > 0 {
		for _, layout := range layouts.Map {
			// TODO: fix it. the singleNoteLoaderEnv loads all notes for _layouts
			if layout.VersionID != id {
				continue
			}

			for _, asset := range layout.Assets {
				res[asset.Path] = struct{}{}
			}
		}

		return res, nil
	}

	if len(res) == 0 && len(nvs.List) > 0 {
		return nvs.List[0].Assets, nil
	}

	// something strange happened
	return nil, fmt.Errorf("unknown source type #%d not found", id)
}

func (a *app) IDHash(entity string, id int64) string {
	sha256 := sha256.New()
	fmt.Fprintf(sha256, "%s:%d", entity, id)
	return hex.EncodeToString(sha256.Sum(nil))
}

func (a *app) HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error {
	err := updatesubgraphs.Resolve(ctx, a)
	if err != nil {
		return fmt.Errorf("failed to update subgraphs: %w", err)
	}

	err = handletgpublishviews.Resolve(ctx, a, changedPathIDs)
	if err != nil {
		return fmt.Errorf("failed to handle Telegram publish views: %w", err)
	}

	// Enqueue embedding generation for changed notes (if vector search enabled)
	if a.config.Features.VectorSearch.Enabled {
		nvs := a.LatestNoteViews()
		for _, pathID := range changedPathIDs {
			noteView := nvs.GetByPathID(pathID)
			if noteView != nil {
				enqueueErr := a.GenerateNoteVersionEmbeddingJob.Enqueue(ctx, noteView.VersionID)
				if enqueueErr != nil {
					a.log.Error("failed to enqueue embedding generation", "version_id", noteView.VersionID, "error", enqueueErr)
				}
			}
		}
	}

	// Trigger change webhook deliveries for changed notes.
	// Check if the current request has skip_webhooks enabled.
	req, reqErr := appreq.FromCtx(ctx)
	skipWebhooks := reqErr == nil && req.SkipWebhooks
	webhookDepth := 0
	if reqErr == nil {
		webhookDepth = req.WebhookDepth
	}

	if skipWebhooks {
		return nil
	}

	webhookChanges := make([]handlenotewebhooks.NoteChange, 0, len(changedPathIDs))
	for _, pathID := range changedPathIDs {
		notePath, npErr := a.NotePathByID(ctx, pathID)
		if npErr != nil {
			a.log.Error("failed to get note path for webhook", "path_id", pathID, "error", npErr)
			continue
		}

		event := "update"
		if notePath.VersionCount == 1 {
			event = "create"
		}

		webhookChanges = append(webhookChanges, handlenotewebhooks.NoteChange{
			PathID: pathID,
			Event:  event,
		})
	}

	if len(webhookChanges) > 0 {
		webhookErr := handlenotewebhooks.Resolve(ctx, a, webhookChanges, webhookDepth)
		if webhookErr != nil {
			a.log.Error("failed to handle note webhooks", "error", webhookErr)
		}
	}

	return nil
}

func (a *app) CurrentUserToken(ctx context.Context) (*usertoken.Data, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	return req.UserToken()
}

var ErrNotAdmin = errors.New("unauthorized")

func (a *app) CanReadNote(ctx context.Context, note *model.NoteView) (bool, error) {
	return canreadnote.Resolve(ctx, a, note)
}

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

func (a *app) GenerateTgAuthURL(_ context.Context, path string, data model.TgAuthToken) (string, error) {
	rawToken, err := a.tgAuthTokenManager.NewToken(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate Telegram auth token: %w", err)
	}

	publicURL := a.PublicURL()
	if publicURL == "" {
		publicURL = "https://example.com" // Fallback URL, must has a https scheme
	}

	u, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse public URL: %w", err)
	}

	u.Path = path

	query := u.Query()
	query.Set(signinbytgauthtoken.QueryParam, rawToken)

	u.RawQuery = query.Encode()

	return u.String(), nil
}

func (a *app) ParseTgAuthToken(ctx context.Context, token string) (*model.TgAuthToken, error) {
	return a.tgAuthTokenManager.ParseToken(token)
}

func (a *app) CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
	return a.nowpaymentsClient.CreateInvoice(params)
}

func (a *app) SearchLiveNotes(query string) ([]model.SearchResult, error) {
	return a.liveNoteLoader.Search(query)
}

func (a *app) SearchLatestNotes(query string) ([]model.SearchResult, error) {
	return a.latestNoteLoader.Search(query)
}

func (a *app) PrepareLatestNotes(ctx context.Context, partial bool) (*model.NoteViews, error) {
	options := noteloader.LoadOptions{}

	if partial {
		options.SkipSearchIndex = true
	}

	// Patches are now loaded automatically by noteloader.Load()
	err := a.latestNoteLoader.Load(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest notes: %w", err)
	}

	return a.latestNoteLoader.NoteViews(), nil
}

func (a *app) PrepareLiveNotes(ctx context.Context) (*model.NoteViews, error) {
	err := a.liveNoteLoader.Load(ctx, noteloader.LoadOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load live notes: %w", err)
	}

	return a.liveNoteLoader.NoteViews(), nil
}

func (a *app) Layouts() *model.Layouts {
	return a.latestNoteLoader.Layouts()
}

func (a *app) LatestNoteViews() *model.NoteViews {
	return a.latestNoteLoader.NoteViews()
}

func (a *app) LiveNoteViews() *model.NoteViews {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := a.SiteConfig(ctx)
	if cfg.ShowDraftVersions {
		return a.latestNoteLoader.NoteViews()
	}

	return a.liveNoteLoader.NoteViews()
}

func (a *app) Now() time.Time {
	return time.Now()
}

func (a *app) AllNotePaths(ctx context.Context) ([]db.NotePath, error) {
	return a.queries.AllNotePaths(ctx)
}

func (a *app) Logger() logger.Logger {
	return a.log
}

func (a *app) LogLevel() string {
	return a.config.LogLevel
}

func (a *app) Features() features.Features {
	return a.config.Features
}

func (a *app) EnqueueGenerateNoteVersionEmbedding(ctx context.Context, versionID int64) error {
	return a.GenerateNoteVersionEmbeddingJob.Enqueue(ctx, versionID)
}

func (a *app) OpenAI() *openai.Client {
	return a.openaiClient
}

func (a *app) AuditLogger() logger.Logger {
	return a.auditLogger
}

func (a *app) DB() *sql.DB {
	return a.conn
}

func (a *app) VacuumDB(ctx context.Context) error {
	// 1. Checkpoint WAL file before vacuum
	_, err := a.conn.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}

	// 2. Reclaim unused space
	_, err = a.conn.ExecContext(ctx, "VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum: %w", err)
	}

	// 3. Update query planner statistics
	_, err = a.conn.ExecContext(ctx, "ANALYZE")
	if err != nil {
		return fmt.Errorf("failed to analyze: %w", err)
	}

	return nil
}

func (a *app) RecordUserNoteView(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
	err := db.WithRetry(ctx, 3, func() error {
		return a.doRecordUserNoteView(ctx, userID, note, referrerVersionID)
	})

	if err != nil {
		a.log.Error(
			"failed to record user note view",
			"error", err,
			"user_id", userID,
			"note_id", note.ID,
		)

		return
	}
}

func (a *app) doRecordUserNoteView(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := a.recordUserNoteViewTx(txCtx, env.WriteQueries, userID, note, referrerVersionID)
		return err == nil, err
	})
}

func (a *app) recordUserNoteViewTx(
	ctx context.Context,
	queries *db.WriteQueries,
	userID int64,
	note *model.NoteView,
	referrerVersionID *int64,
) error {
	const maxCount = int64(100)

	dailyParams := db.UpsertUserNoteDailyViewParams{
		UserID: userID,
		PathID: note.PathID,
	}

	dailyCount, err := queries.UpsertUserNoteDailyView(ctx, dailyParams)
	if err != nil {
		return fmt.Errorf("failed to upsert user note daily view: %w", err)
	}

	// TODO: read from the app config
	if dailyCount < maxCount {
		viewParams := db.InsertUserNoteViewParams{
			UserID:           userID,
			VersionID:        note.VersionID,
			RefererVersionID: referrerVersionID,
		}

		err = queries.InsertUserNoteView(ctx, viewParams)
		if err != nil {
			return fmt.Errorf("failed to insert user note view: %w", err)
		}

		err = queries.IncreaseUserNoteViewCount(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to increase user note view count: %w", err)
		}
	}

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

func (a *app) GenerateGitToken() string {
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

func (a *app) Insecure() bool {
	return a.config.UserToken.Insecure
}

// BuildGoogleAuthURL returns (callbackURL, authURL, error).
// callbackURL is always returned for admin UI display.
// authURL is only returned if OAuth is configured (or dry=true for just getting callbackURL).
//
//nolint:nonamedreturns // named returns document the multiple string return values
func (a *app) BuildGoogleAuthURL(ctx context.Context, redirectURL string, dry bool) (callbackURL string, authURL string, err error) {
	publicURL := a.GetPublicURLForRequest(ctx)
	callbackURL = fmt.Sprintf("%s/_system/auth/google/callback", publicURL)

	if dry {
		return callbackURL, "", nil
	}

	creds, err := a.GetActiveGoogleOAuthCredentials(ctx)
	if err != nil {
		// No active credentials - OAuth not configured
		return callbackURL, "", nil //nolint:nilerr // expected: missing credentials returns empty authURL
	}
	if creds.ClientID == "" {
		return callbackURL, "", nil
	}

	authURL = fmt.Sprintf("%s/_system/auth/google?redirect=%s", publicURL, url.QueryEscape(redirectURL))
	return callbackURL, authURL, nil
}

// BuildGitHubAuthURL returns (callbackURL, authURL, error).
// callbackURL is always returned for admin UI display.
// authURL is only returned if OAuth is configured (or dry=true for just getting callbackURL).
//
//nolint:nonamedreturns // named returns document the multiple string return values
func (a *app) BuildGitHubAuthURL(ctx context.Context, redirectURL string, dry bool) (callbackURL string, authURL string, err error) {
	publicURL := a.GetPublicURLForRequest(ctx)
	callbackURL = fmt.Sprintf("%s/_system/auth/github/callback", publicURL)

	if dry {
		return callbackURL, "", nil
	}

	creds, err := a.GetActiveGitHubOAuthCredentials(ctx)
	if err != nil {
		// No active credentials - OAuth not configured
		return callbackURL, "", nil //nolint:nilerr // expected: missing credentials returns empty authURL
	}
	if creds.ClientID == "" {
		return callbackURL, "", nil
	}

	authURL = fmt.Sprintf("%s/_system/auth/github?redirect=%s", publicURL, url.QueryEscape(redirectURL))
	return callbackURL, authURL, nil
}

// ValidateGoogleOAuthCredentials validates Google OAuth credentials by making a test API call.
func (a *app) ValidateGoogleOAuthCredentials(ctx context.Context, clientID, clientSecret string) error {
	redirectURI := fmt.Sprintf("%s/_system/auth/google/callback", a.GetPublicURLForRequest(ctx))
	return googleauth.ValidateCredentials(clientID, clientSecret, redirectURI)
}

// ValidateGitHubOAuthCredentials validates GitHub OAuth credentials by making a test API call.
func (a *app) ValidateGitHubOAuthCredentials(ctx context.Context, clientID, clientSecret string) error {
	return githubauth.ValidateCredentials(clientID, clientSecret)
}

func (a *app) GetPublicURLForRequest(ctx context.Context) string {
	// If PublicURL is configured, use it
	if publicURL := a.config.PublicURL; publicURL != "" {
		return publicURL
	}

	// Otherwise, extract URL from the current request
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		// Fallback to empty string if no request context
		return ""
	}

	if req.Req == nil {
		return ""
	}

	// Get scheme (http or https)
	scheme := "http"
	if req.Req.IsTLS() {
		scheme = "https"
	}

	// Get host from request
	host := string(req.Req.Host())

	return fmt.Sprintf("%s://%s", scheme, host)
}

func (a *app) TrustedDomains() []string {
	domains := []string{}

	// Always add the public URL domain
	if publicURL := a.config.PublicURL; publicURL != "" {
		if u, err := url.Parse(publicURL); err == nil && u.Host != "" {
			domains = append(domains, u.Host)
		}
	}

	// In dev mode, also add localhost:8081
	if a.config.DevMode {
		domains = append(domains, "localhost:8081")
	}

	return domains
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

func generateEightCharCode() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
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

	err = appreq.CtxEnv(ctx, a).InsertSignInCode(ctx, db.InsertSignInCodeParams{
		UserID: userID,
		Code:   sCode,
	})
	if err != nil {
		return "", fmt.Errorf("failed to insert sign-in code: %w", err)
	}

	return sCode, nil
}

func (a *app) GenerateTgAttachCode() string {
	code, err := generateEightCharCode()
	if err != nil {
		// Log error and generate a fallback code
		a.Logger().Error("failed to generate attach code", "error", err)
		// Fallback to timestamp-based code if random generation fails
		return fmt.Sprintf("%08x", time.Now().Unix()%0xFFFFFFFF)
	}
	return code
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

func (a *app) LoadLatestLayout(source model.LayoutSourceFile) model.Layout {
	load := a.latestNoteLoader.Layouts().Load
	if load != nil {
		return load(source)
	}
	return model.Layout{}
}

func (a *app) IsDevMode() bool {
	return a.config.DevMode
}

func (a *app) MaxRequestBodySize() int {
	return a.config.MaxRequestBodySize
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
	if a.config.DevMode {
		a.log.Warn("page not found", "path", path)
	}

	err := a.notFoundTracker.Track(path, ip)
	if err != nil {
		a.log.Error("failed to track not found", "path", path, "error", err)
	}
}

func (a *app) NotifyPuchaseUpdated(email string) {
}

func (a *app) TryToAutoRegisterUser(ctx context.Context, email string) (*db.User, error) {
	user, err := getboostyuser.Resolve(ctx, a, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check Boosty user: %w", err)
	}

	if user != nil {
		return user, nil
	}

	user, err = getpatreonuser.Resolve(ctx, a, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check Patreon user: %w", err)
	}

	// etc

	return user, nil
}

func (a *app) RequestIP(ctx context.Context) string {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return ""
	}

	return string(req.Req.RemoteIP())
}

func (a *app) handleCors(ctx *fasthttp.RequestCtx) bool {
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "http://localhost:9081" || origin == "app://obsidian.md" {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Cookie, X-API-Key, X-Plugin-Version")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	}

	if ctx.IsOptions() {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
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

const robotsTxtContentOpened = `User-agent: *
Disallow:`

const robotsTxtContentClosed = `User-agent: *
Disallow: /`

func (a *app) handleRobotsTxt(req *appreq.Request) bool {
	if req.Path == "/robots.txt" {
		req.Req.SetContentType("text/plain")
		req.Req.SetStatusCode(http.StatusOK)

		txt := a.SiteConfig(context.Background()).RobotsTxt

		switch txt {
		case "closed":
			req.Req.SetBodyString(robotsTxtContentClosed)
		case "opened":
			req.Req.SetBodyString(robotsTxtContentOpened)
		default:
			req.Req.SetBodyString(txt)
		}

		return true
	}

	return false
}

func (a *app) handleRSSFeed(req *appreq.Request) bool {
	if !strings.HasSuffix(req.Path, ".rss.xml") {
		return false
	}

	cfg := a.SiteConfig(context.Background())
	if !cfg.EnableRSS {
		return false
	}

	// Strip .rss.xml suffix to get the note path.
	notePath := strings.TrimSuffix(req.Path, ".rss.xml")
	if notePath == "" {
		notePath = "/"
	}

	notes := a.LiveNoteViews()
	note := notes.GetByPath(notePath)
	if note == nil {
		return false
	}

	xmlBytes, err := rssfeed.Generate(note, a.PublicURL(), notes)
	if err != nil {
		a.log.Error("failed to generate RSS feed", "error", err, "path", req.Path)
		return false
	}

	req.Req.SetContentType("application/rss+xml; charset=utf-8")
	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetBody(xmlBytes)

	return true
}

func (a *app) handleSitemap(req *appreq.Request) bool {
	if req.Path != "/sitemap.xml" {
		return false
	}

	nvs := a.LiveNoteViews()
	if nvs == nil || len(nvs.Sitemap) == 0 {
		return false
	}

	req.Req.SetContentType("application/xml; charset=utf-8")
	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetBody(nvs.Sitemap)

	return true
}

// Middleware should return true if the request is fully handled.
type Middleware func(req *appreq.Request) bool

func (a *app) prepareMiddlewares() []Middleware {
	fsHandler := a.assetsFS.NewRequestHandler()

	return []Middleware{
		a.handleRobotsTxt,
		a.handleSitemap,
		a.handleRSSFeed,
		func(req *appreq.Request) bool {
			return a.handleCors(req.Req)
		},
		func(req *appreq.Request) bool {
			return a.handleDebugAPI(req.Req)
		},
		func(req *appreq.Request) bool {
			return a.gitAPI.HandleRequest(req.Req)
		},
		func(req *appreq.Request) bool {
			return a.handleAdminAssets(req, req.Path)
		},
		func(req *appreq.Request) bool {
			if strings.HasPrefix(req.Path, "/assets/") {
				fsHandler(req.Req)
				return true
			}

			return false
		},
		func(req *appreq.Request) bool {
			return a.handlePurchaseTokens(req.Req)
		},
		func(req *appreq.Request) bool {
			return signinbytgauthtoken.Process(req.Req, a)
		},
		func(req *appreq.Request) bool {
			return a.TgBots.ProcessWebhookRequest(req.Path, func() []byte { return req.Req.PostBody() })
		},
	}
}

func (a *app) startServer() {
	handleGraphQL := a.prepareGraphQLHandler()

	rtr := router.New(a)

	middlewares := a.prepareMiddlewares()

	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		req := appreq.Acquire()
		req.Env = a
		req.Req = ctx
		req.Path = path
		req.TokenManager = a.tokenManager
		req.StoreInContext() // appreq.FromCtx(ctx)
		defer appreq.Release(req)

		for _, mw := range middlewares {
			if mw(req) {
				return
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

		if handleGraphQL(ctx, path) {
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
	}

	handlerTimeout := 60 * time.Second
	if a.config.DevMode {
		handlerTimeout = 10 * time.Minute
	}

	timeoutHandler := fasthttp.TimeoutHandler(handler, handlerTimeout, "timeout")

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			// SSE connections are long-lived, skip timeout handler for them.
			if strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream") {
				handler(ctx)
				return
			}
			timeoutHandler(ctx)
		},
		MaxRequestBodySize: a.config.MaxRequestBodySize * 1024 * 1024,
		ReadTimeout:        handlerTimeout,
		WriteTimeout:       handlerTimeout,
		IdleTimeout:        handlerTimeout,
	}

	runServer := func() {
		if len(a.config.AcmeDomains) == 0 {
			err := s.ListenAndServe(a.config.ListenAddr)
			if err != nil {
				panic(err)
			}

			return
		}

		a.startACMEServer(s)
	}

	go a.startInternalServer()

	if a.config.DevMode {
		runServer()
	} else {
		go runServer()
		a.waitForShutdown(s)
	}
}

func (a *app) prepareGraphQLHandler() func(ctx *fasthttp.RequestCtx, path string) bool {
	// graphql.
	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(playground.Handler("GraphQL playground", "/graphql"))

	gqlMetrics := metrics.NewGraphQLMetrics()

	gqlHandler := graph.NewHandler(a)
	gqlHandler.Use(gqlMetrics)
	gqlHandler.AroundOperations(gqlMetrics.Middleware())
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(gqlHandler)
	compressedGraphqlHandler := fasthttp.CompressHandler(graphqlHandler)

	// SSE uses a custom handler that does not pool the response writer,
	// avoiding a data race in fasthttpadaptor where the pooled writer
	// is recycled while the SSE goroutine is still writing to it.
	sseHandler := fastgql.NewSSEHandler(gqlHandler)

	return func(ctx *fasthttp.RequestCtx, path string) bool {
		if strings.HasPrefix(path, "/graphql") {
			switch {
			case string(ctx.Method()) == "GET":
				playgroundHandler(ctx)
			case strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream"):
				sseHandler(ctx)
			default:
				compressedGraphqlHandler(ctx)
			}

			return true
		}

		return false
	}
}

func (a *app) waitForShutdown(s *fasthttp.Server) {
	<-a.sigChan

	a.stopped.Store(true)
	a.shutdown() // notify all shutdown listeners

	a.log.Info("shutting down in", "grace_period", a.config.ShutdownGracePeriod)

	time.Sleep(a.config.ShutdownGracePeriod)

	// Perform shutdown backup if simple backup enabled
	if a.simpleBackup != nil {
		a.log.Info("performing shutdown backup...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.simpleBackup.PerformBackup(ctx); err != nil {
			a.log.Error("shutdown backup failed", "error", err)
		} else {
			a.log.Info("shutdown backup completed")
		}
	}

	a.log.Info("shutting down server", "timeout", a.config.ShutdownTimeout)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer shutdownCancel()

	err := s.ShutdownWithContext(shutdownCtx)
	if err != nil {
		a.log.Error("failed to shutdown server gracefully", "error", err)
		return
	}

	a.log.Info("server stopped")
}

func (a *app) startInternalServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if a.stopped.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("shutting down"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", metrics.Setup())

	// Start metrics updater
	metricsUpdater := metrics.NewUpdater(a, a.config.Metrics.UpdateInterval)
	go func() {
		err := metricsUpdater.Run(a.ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			a.log.Error("metrics updater stopped", "error", err)
		}
	}()

	// Register pprof endpoints
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	server := &http.Server{
		Addr:    a.config.InternalListenAddr,
		Handler: mux,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-a.shutdownCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			a.log.Error("failed to shutdown internal server", "error", err)
		}
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func (a *app) startACMEServer(s *fasthttp.Server) {
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

	// Start HTTP server on port 80 for ACME challenges and HTTPS redirect
	httpServer := &http.Server{
		Addr:         ":80",
		Handler:      certManager.HTTPHandler(nil),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		<-a.shutdownCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()

		err := httpServer.Shutdown(shutdownCtx)
		if err != nil {
			a.log.Error("failed to shutdown HTTP redirect server", "error", err)
		}
	}()

	go func() {
		a.log.Info("starting HTTP redirect server on port 80")
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			a.log.Error("HTTP redirect server failed", "error", err)
		}
	}()

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

	a.log.Info("starting HTTPS server on port 443")
	err = fasthttp.Serve(lnTLS, s.Handler)
	if err != nil {
		panic(err)
	}
}

// getEnvOrDefault retrieves the environment from the request context or returns a default environment.
// the context from the request wrapped all queries in a transaction.
func getEnvOrDefault[T any](ctx context.Context, defaultEnv *app) (T, error) {
	var zero T

	req, err := appreq.FromCtx(ctx)
	if err != nil {
		if errors.Is(err, appreq.ErrNotFound) {
			env, ok := any(defaultEnv).(T)
			if ok {
				return env, nil
			}

			return zero, fmt.Errorf("app does not implement required type: %T", zero)
		}

		return zero, fmt.Errorf("failed to get request from context: %w", err)
	}

	env, ok := req.Env.(T)
	if ok {
		return env, nil
	}

	return zero, fmt.Errorf("request env does not implement required type: %T", zero)
}

// restoreEnvAdapter adapts dependencies for the restore phase (before DB init).
type restoreEnvAdapter struct {
	*miniostorage.FileStorage
	log logger.Logger
}

func (r *restoreEnvAdapter) Logger() logger.Logger {
	return r.log
}

func (r *restoreEnvAdapter) DB() *sql.DB {
	return nil // Not needed for restore
}

// BackupManager returns the backup manager for cronjob env interface.
func (a *app) BackupManager() *simplebackup.Manager {
	return a.simpleBackup
}

// ShortAPITokenSecret returns the secret used for signing short API tokens.
func (a *app) ShortAPITokenSecret() string {
	return a.config.UserToken.Secret
}

// WebhookHTTPClient returns the shared HTTP client for webhook deliveries.
func (a *app) WebhookHTTPClient() *fasthttp.Client {
	return a.webhookHTTPClient
}

// EnqueueDeliverChangeWebhook enqueues a change webhook delivery job.
func (a *app) EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error {
	return a.DeliverChangeWebhookJob.EnqueueDeliverChangeWebhook(ctx, params)
}

// EnqueueDeliverCronWebhook enqueues a cron webhook delivery job.
func (a *app) EnqueueDeliverCronWebhook(ctx context.Context, params delivercronwebhook.DeliverCronParams) error {
	return a.DeliverCronWebhookJob.EnqueueDeliverCronWebhook(ctx, params)
}
