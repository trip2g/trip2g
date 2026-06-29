package main

import (
	"context"
	"database/sql"
	"fmt"
	htmltemplate "html/template"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	_ "time/tzdata" // embed IANA tz DB so time.LoadLocation works on any base image

	"trip2g/internal/appconfig"
	"trip2g/internal/auditlogger"
	"trip2g/internal/boosty"
	"trip2g/internal/boostyjobs"
	"trip2g/internal/case/admin/renderpreview"
	"trip2g/internal/case/backjob/deliverchangewebhook"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/case/backjob/extractnotionpages"
	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/case/backjob/importtelegramchannel"
	"trip2g/internal/case/backjob/refreshchartdata"
	"trip2g/internal/case/backjob/sendformsubmit"
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
	"trip2g/internal/case/mcp"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/chartdata"
	"trip2g/internal/configregistry"
	"trip2g/internal/cronjobs"
	"trip2g/internal/dataencryption"
	"trip2g/internal/db"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/gitapi"
	"trip2g/internal/hotauthtoken"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/notebus"
	"trip2g/internal/noteloader"
	"trip2g/internal/notfoundtracker"
	"trip2g/internal/notion"
	"trip2g/internal/nowpayments"
	"trip2g/internal/openai"
	"trip2g/internal/pagecache"
	"trip2g/internal/patreon"
	"trip2g/internal/patreonjobs"
	"trip2g/internal/personaltoken"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/readreplica"
	"trip2g/internal/redirectmanager"
	"trip2g/internal/replicareload"
	"trip2g/internal/simplebackup"
	"trip2g/internal/tgauthtoken"
	"trip2g/internal/tgbots"
	"trip2g/internal/tgtd"
	"trip2g/internal/turnstile"
	"trip2g/internal/userbans"
	"trip2g/internal/usertoken"
	"trip2g/internal/zerologger"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/valyala/fasthttp"

	_ "modernc.org/sqlite"
)

var _ mcp.Env = (*app)(nil)
var _ replicareload.Env = (*app)(nil)

// replicaNoteReloadInterval is how often a read replica polls for note changes.
// A few seconds of staleness on the public read path is acceptable (see docs/dev/readreplica.md).
// A longer interval reduces reload churn (bleve IO + note-loader work) on resource-constrained hosts.
const replicaNoteReloadInterval = 5 * time.Second

type app struct {
	*db.Queries
	*db.WriteQueries

	Storage
	*dataencryption.Manager
	*patreonjobs.PatreonJobs
	*boostyjobs.BoostyJobs
	*tgbots.TgBots
	*cronjobs.CronJobs
	*sendsignincode.SendSignInCodeJob
	*sendformsubmit.SendFormSubmitEmailJob
	refreshChartDataJob *refreshchartdata.Job
	replicaReload       *replicareload.ReplicaReload
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
	fedHTTPClient     *fasthttp.Client

	webhookTestCalls []webhookTestCall
	webhookTestMu    *sync.Mutex

	debugJobLog []debugJobRecord
	debugJobMu  *sync.Mutex

	// noteWriteMu is a POINTER, not a value: app is shallow-copied per request
	// (`newEnv := *a` in AcquireTxEnvInRequest/WithTransaction). A value mutex
	// would be copied — defeating cross-request serialization and, worse,
	// deadlocking if copied while locked. A pointer makes every copy share the
	// one real mutex so plugin pushNotes and git apply/materialize truly serialize.
	noteWriteMu *sync.Mutex

	openaiClient *openai.Client

	sigChan     chan os.Signal
	shutdownCtx context.Context
	shutdown    context.CancelFunc
	// Pointer so the per-request shallow copy (newEnv := *a) shares the one flag
	// rather than copying a noCopy atomic value.
	stopped *atomic.Bool
	// ready reports whether the instance can fully serve, including writes. It
	// flips true only after the writer slot is acquired and writer subsystems
	// (queues, cron, patreon/boosty refresh) have started. /readyz returns 503
	// until then so Nomad/Traefik route traffic only to fully-ready instances.
	// Pointer for the same reason as stopped (shared across per-request copies).
	ready          *atomic.Bool
	internalServer *fasthttp.Server
	// appHandler holds the full public request handler once startServer has
	// built it. The internal server's leader-side replica intake reads it to run
	// forwarded writes through the real pipeline; nil (→ 503) until ready.
	// Pointer (like stopped/ready) so per-request app copies share one atomic.
	appHandler *atomic.Pointer[fasthttp.RequestHandler]
	ctx        context.Context

	graphTxs *graphTransactions

	queries   *db.Queries
	conn      *sql.DB
	writeConn *sql.DB
	queueConn *sql.DB // separate connection for goqite so queue polling never blocks app writes

	currentTx *sql.Tx

	noteBus *notebus.Bus

	log logger.Logger

	auditLogger logger.Logger

	globalQueue              *appQueue
	telegramBotAPIQueue      *appQueue
	telegramAccountAPIQueue  *appQueue
	telegramTaskQueue        *appQueue
	telegramLongRunningQueue *appQueue

	// mail *mailyak.MailYak

	tokenManager          *usertoken.Manager
	personalTokenResolver *personaltoken.Resolver

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
	// localAssetsFS serves note assets from the local-storage dir via the
	// /_assets/ route. nil unless the local storage backend is active.
	localAssetsFS *fasthttp.FS
	// Pointer, not value: app is shallow-copied per request (newEnv := *a), and
	// assetHashes is shared by reference. A copied value mutex would not guard
	// the shared map, risking a fatal concurrent map write. See noteWriteMu.
	assetsMu *sync.Mutex

	*configregistry.SiteConfigBuilder

	// pageCache holds pre-gzipped anonymous rendered-page responses. Named (not
	// embedded) so its generic Get/Set/Clear don't promote onto app and collide
	// with other embedded types; reached via the CachedPage/StoreCachedPage/
	// ClearPageCache accessors below.
	pageCache *pagecache.PageCache

	liveNoteLoader         *noteloader.Loader
	latestNoteLoader       *noteloader.Loader
	frontmatterPatchLoader *frontmatterpatch.Loader

	*chartdata.ChartData // server-side data for url/internal datachart sources (promotes ChartRows, SaveChartData)

	patreonClientManager *patreon.ClientManager
	boostyClientManager  *boosty.ClientManager

	gitAPI *gitapi.API

	previewBuffer *renderpreview.PreviewBuffer

	appQueues map[string]*appQueue

	simpleBackup *simplebackup.Manager

	telegramAuthManager *tgtd.AuthManager

	*turnstile.Client
	signinCounter *requestemailsignin.SignInCounter

	gqlServer   *handler.Server
	gqlExecutor graphql.GraphExecutor

	// replicaForwarder is non-nil only in read-only replica mode. It reverse-
	// proxies mutating requests to the leader (--leader-addr).
	replicaForwarder *readreplica.Forwarder
}

//nolint:funlen // boot sequence split into read-only (Block A) and writer (Block B) phases
func main() {
	if err := defaulttemplate.Init(); err != nil {
		panic(fmt.Errorf("failed to init default template i18n: %w", err))
	}

	config, err := appconfig.Get()
	if err != nil {
		panic(fmt.Errorf("failed to load configuration: %w", err))
	}

	log := zerologger.New(config.LogLevel, config.DevMode)

	// Deprecated: map legacy RESEND_API_KEY onto Resend SMTP (warns if it fires).
	applyLegacyEmailConfig(config, log)

	if config.SMTPHost == "" {
		log.Warn(
			"no email transport configured: outgoing email (including sign-in codes) will be skipped; codes are logged instead — set SMTP_HOST/SMTP_USER/SMTP_PASS or RESEND_API_KEY to enable delivery",
		)
	}

	// RESTORE PHASE (Pre-DB Init) - if simple backup enabled
	if config.SimpleBackup.Enabled {
		restoreBackup(log, config)
	}

	dbs := initDBs(config, log)
	conn, writeConn, queueConn := dbs.Read, dbs.Write, dbs.Queue

	tokenManager := usertoken.NewManager(config.UserToken)
	// use USER_TOKEN_INSECURE instead
	// tokenManager.SetInsecure(config.DevMode) // for k6

	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "read: no tx:")))
	writeQueries := db.NewWriteQueries(
		db.WithLogger(writeConn, logger.WithPrefix(log, "write: no tx:")).
			WithPoolStats(writeConn.Stats),
	)

	nowpaymentsClient, err := nowpayments.NewClient(config.NowpaymentsAPIKey, log)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileStorage, err := newStorage(ctx, config)
	if err != nil {
		panic(err)
	}

	log.Info("using storage prefix", "prefix", config.Storage.Prefix)

	a := &app{
		Queries:      queries,
		WriteQueries: writeQueries,

		Storage: fileStorage,
		Manager: initDataEncryptionManager(config),

		config: config,

		tokenManager: tokenManager,

		graphTxs: &graphTransactions{
			EnvMap: make(map[*app]*sql.Tx),
		},

		noteWriteMu:   &sync.Mutex{},
		assetsMu:      &sync.Mutex{},
		webhookTestMu: &sync.Mutex{},
		debugJobMu:    &sync.Mutex{},
		stopped:       &atomic.Bool{},
		ready:         &atomic.Bool{},
		appHandler:    &atomic.Pointer[fasthttp.RequestHandler]{},

		hotAuthTokenManager: hotauthtoken.NewManager(config.HotAuthToken),
		tgAuthTokenManager:  tgauthtoken.NewManager(config.TgAuthToken),

		purchaseTokenManager: purchasetoken.NewManager(config.PurchaseToken),

		log:     log,
		noteBus: notebus.New(log),
		queries: queries,
		conn:    conn,
		// mail:    mailyak.New(mailAddr, mailAuth),

		writeConn: writeConn,
		queueConn: queueConn,

		UserBans: userbans.New(queries),

		nowpaymentsClient: nowpaymentsClient,

		Client:        turnstile.New(config.Turnstile),
		signinCounter: &requestemailsignin.SignInCounter{},
	}

	if config.IsReadReplica() {
		a.replicaForwarder = readreplica.NewForwarder(config.LeaderAddr, config.UserToken.Secret)
	}

	a.ctx = ctx
	a.SiteConfigBuilder = configregistry.NewSiteConfigBuilder(a)
	a.pageCache = pagecache.New()
	a.sigChan = make(chan os.Signal, 1)
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)

	a.shutdownCtx, a.shutdown = context.WithCancel(context.Background())

	a.auditLogger = auditlogger.New(ctx, a, a.config.AuditLog)

	// ========================================================================
	// BLOCK A — read-only warmup. Everything here is safe to run WITHOUT the
	// SQLite writer slot: it only reads the DB (or touches no DB at all), so a
	// new instance can fully warm up (load notes, build in-memory indexes,
	// construct handlers) while the OLD instance keeps serving, including
	// writes. No writer subsystem (queues, cron, patreon/boosty refresh) is
	// started here. /readyz stays 503 until Block B finishes.
	// ========================================================================

	// No-DB construction halves (client managers, job handlers).
	a.constructPatreon()
	a.constructBoosty()

	a.globalQueue = a.createQueue(ctx, "global_jobs", QueueOpts{
		Limit:        10,
		PollInterval: a.config.GlobalQueuePollInterval,
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
			a.config.Features.VectorSearch.BaseURL,
		)
	}

	// Start the internal health server early so /livez answers 200 throughout
	// the read-only warmup (loadAllNotes). Traefik active-health-checks /livez
	// and would pull the backend out of rotation during ~12s warmup if this
	// server were started later. /readyz correctly stays 503 until a.ready flips.
	go a.startInternalServer()

	a.constructJobs()

	a.redirectManager, err = redirectmanager.New(ctx, a)
	if err != nil {
		panic(fmt.Errorf("failed to create redirect manager: %w", err))
	}

	// Read-only replica: skip the not-found tracker (it writes to the DB every
	// minute via runDumpTicker; the replica's connection is read-only).
	if !config.IsReadReplica() {
		// TODO: remove this tracker. it's extra work
		a.notFoundTracker, err = notfoundtracker.New(ctx, a)
		if err != nil {
			panic(fmt.Errorf("failed to create not found tracker: %w", err))
		}
	}

	a.liveNoteLoader = noteloader.New("live", makeLiveNoteLoaderWrapper(a), a.config.MDLoaderConfig)
	a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), a.config.MDLoaderConfig)
	a.ChartData = chartdata.New(a)
	a.liveNoteLoader.SetChartDataProvider(a)
	a.latestNoteLoader.SetChartDataProvider(a)
	a.frontmatterPatchLoader = frontmatterpatch.NewLoader(a)

	a.gitAPI, err = gitapi.New(ctx, a.config.GitAPI, a)
	if err != nil {
		panic(err)
	}

	a.previewBuffer = renderpreview.NewPreviewBuffer(a.config.RenderPreview)

	// TODO: remove notion at all
	a.notionClientManager = notion.NewClientManager(a, a.config.Notion)

	// Initialize simple backup manager if enabled
	if config.SimpleBackup.Enabled {
		a.simpleBackup = simplebackup.New(a, config.DatabaseFile)
		log.Info("simple backup manager initialized")
	}

	// loadAllNotes is read-only (noteloader.Load does zero DB writes): it builds
	// the in-memory NoteViews / bleve index / layouts / sitemap. Safe in Block A.
	err = a.loadAllNotes(ctx, noteloader.LoadOptions{})
	if err != nil {
		panic(err)
	}

	a.setupAssets()
	a.setTokenValidator()
	a.personalTokenResolver = personaltoken.NewResolver(a)
	a.setFileStorageExpiringCallback()

	// ========================================================================
	// WRITER SLOT — acquire before starting any writer subsystem. This is an
	// honest-but-minimal probe (BEGIN IMMEDIATE; COMMIT) that the SQLite write
	// lock is currently grabbable; the OLD instance releases it on SIGTERM
	// after stopping its own writers. It does NOT hold the lock open and does
	// NOT guarantee hard cross-process single-writer (deferred to Phase 2).
	// ========================================================================
	// Read-only replica: never acquire the writer slot or start any writer
	// subsystem. The replica serves reads off the replicated DB and forwards
	// every mutating request to the leader (--leader-addr). It becomes ready as
	// soon as the read-only warmup (Block A) is done.
	if config.IsReadReplica() {
		log.Info("read-only replica mode: skipping writer subsystems, forwarding writes to leader", "leader", config.LeaderAddr)
		a.replicaReload = replicareload.New(a, a.log, replicaNoteReloadInterval)
		go a.replicaReload.Run(a.shutdownCtx)
		a.ready.Store(true)
		a.startServer()
		return
	}

	if acquireErr := db.AcquireWriterSlot(ctx, config.DatabaseFile, config.WriterAcquireTimeout); acquireErr != nil {
		panic(fmt.Errorf("failed to acquire writer slot: %w", acquireErr))
	}

	log.Info("writer slot acquired, starting writer subsystems")

	// ========================================================================
	// BLOCK B — writer-only. Runs only after the writer slot is acquired.
	// Everything here writes to the DB or starts a background loop that does.
	// ========================================================================

	// createOwnerIfNotExists inserts the owner user+admin (WRITE).
	err = a.createOwnerIfNotExists(ctx)
	if err != nil {
		panic(err)
	}

	// cron jobs (writes UpsertCronJob/DeleteCronJobByName + starts cron).
	a.startJobWriters(ctx)

	// patreon/boosty: credential scan, webhook registration, refresh jobs (writes).
	a.startPatreonWriters(ctx)
	a.startBoostyWriters(ctx)

	// queue runners (poll + execute jobs that write).
	a.globalQueue.start()
	a.telegramTaskQueue.start()
	a.telegramBotAPIQueue.start()
	a.telegramAccountAPIQueue.start()
	a.telegramLongRunningQueue.start()

	// Fully ready: can serve reads AND writes. /readyz flips to 200.
	a.ready.Store(true)

	a.startServer()
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

	// A reload changes note content, layouts and telegram links, so flush the
	// whole anonymous page cache rather than reasoning about which keys moved.
	a.ClearPageCache()

	return a.latestNoteLoader.NoteViews(), nil
}

func (a *app) PrepareLiveNotes(ctx context.Context) (*model.NoteViews, error) {
	err := a.liveNoteLoader.Load(ctx, noteloader.LoadOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load live notes: %w", err)
	}

	a.ClearPageCache()

	return a.liveNoteLoader.NoteViews(), nil
}

// CachedPage returns the pre-gzipped bytes cached for key, if any. Part of the
// rendernotepage.Env contract for the anonymous page cache.
func (a *app) CachedPage(key pagecache.Key) ([]byte, bool) {
	return a.pageCache.Get(key)
}

// StoreCachedPage records pre-gzipped bytes for key.
func (a *app) StoreCachedPage(key pagecache.Key, gz []byte) {
	a.pageCache.Set(key, gz)
}

// ClearPageCache drops every cached page (on note reload).
func (a *app) ClearPageCache() {
	a.pageCache.Clear()
}

func (a *app) Layouts() *model.Layouts {
	return a.latestNoteLoader.Layouts()
}

func (a *app) LatestNoteViews() *model.NoteViews {
	return a.latestNoteLoader.NoteViews()
}

func (a *app) PreviousLatestNoteHTML(pathID int64) (htmltemplate.HTML, bool) {
	return a.latestNoteLoader.PreviousHTML(pathID)
}

func (a *app) LatestNoteChunks() []model.NoteChunk {
	return a.latestNoteLoader.NoteChunks()
}

func (a *app) LiveNoteChunks() []model.NoteChunk {
	return a.liveNoteLoader.NoteChunks()
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
