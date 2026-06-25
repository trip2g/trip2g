package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
	"trip2g/internal/appconfig"
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
	"trip2g/internal/cronjobs"
	"trip2g/internal/dataencryption"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/noteloader"
	"trip2g/internal/simplebackup"
	"trip2g/internal/webhookutil"
)

// DBSet holds the three pooled connections the app uses. In a read replica all
// three point to the same strict-read-only pool: there is no local writer, and
// the query_only guardrail turns any stray write into an error rather than
// silent corruption — every mutation is forwarded to the leader instead.
type DBSet struct {
	Read  *sql.DB // application reads + maintenance (checkpoint/vacuum/analyze)
	Write *sql.DB // write transactions
	Queue *sql.DB // goqite queue, isolated so polling never blocks app writes
}

func initDBs(config *appconfig.Config, log logger.Logger) DBSet {
	dbConfig := db.SetupConfig{
		DatabaseFile: config.DatabaseFile,
		Logger:       log,
		LogQueries:   config.LogQueries,
		ReadOnly:     true,
		SkipDump:     true,
		DevMode:      config.DevMode,
	}

	// Read-only replica: skip migrations (the DB is owned by the replicator and
	// already migrated by the leader) and open strict read-only. There is no
	// local writer — reuse the read-only pool for all three handles; the
	// query_only guardrail turns any stray local write into an error instead of
	// silent corruption. All mutating requests are forwarded to the leader.
	if config.IsReadReplica() {
		dbConfig.SkipMigrations = true
		dbConfig.StrictReadOnly = true
	}

	conn, err := db.Setup(dbConfig)
	if err != nil {
		panic(fmt.Errorf("failed to setup database: %w", err))
	}

	if config.IsReadReplica() {
		// One strict-read-only pool serves all three roles; see DBSet doc.
		return DBSet{Read: conn, Write: conn, Queue: conn}
	}

	dbConfig.ReadOnly = false
	dbConfig.CheckStatus = true

	writeConn, err := db.Setup(dbConfig)
	if err != nil {
		panic(fmt.Errorf("failed to setup database: %w", err))
	}

	// Separate connection for goqite queue so polling goroutines never compete
	// with application writes for the single writeConn connection slot.
	dbConfig.CheckStatus = false
	dbConfig.SkipDump = true
	queueConn, err := db.Setup(dbConfig)
	if err != nil {
		panic(fmt.Errorf("failed to setup queue database: %w", err))
	}

	return DBSet{Read: conn, Write: writeConn, Queue: queueConn}
}

func initDataEncryptionManager(config *appconfig.Config) *dataencryption.Manager {
	manager, err := dataencryption.NewManager(config.DataEncryption)
	if err != nil {
		panic(fmt.Errorf("failed to create data encryption manager: %w", err))
	}
	return manager
}

func restoreBackup(log logger.Logger, config *appconfig.Config) {
	log.Info("simple backup enabled, checking for restore")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create temporary storage client for restore
	restoreStorage, restoreErr := newStorage(ctx, config)
	if restoreErr != nil {
		log.Error("FATAL: failed to init storage for restore", "error", restoreErr)
		panic(fmt.Errorf("failed to init storage for restore: %w", restoreErr))
	}

	// Create restore environment adapter
	restoreEnv := &restoreEnvAdapter{
		Storage: restoreStorage,
		log:     log,
	}

	restoreMgr := simplebackup.New(restoreEnv, config.DatabaseFile)

	startupErr := restoreMgr.RestoreOnStartup(ctx)
	if startupErr != nil {
		log.Error("FATAL: failed to restore database", "error", startupErr)
		panic(fmt.Errorf("failed to restore database: %w", startupErr))
	}
}

// constructJobs wires up all job handlers (no DB writes). The handler *.New(a)
// constructors only build in-memory structs; they do not touch the database.
// initDebugJobs only registers a queue handler (no DB write). The DB-writing,
// cron-starting part lives in startJobWriters (Block B).
func (a *app) constructJobs() {
	a.SendTelegramMessageJob = sendtelegrammessage.New(a)
	a.UpdateTelegramMessageJob = updatetelegrammessage.New(a)
	a.SendTelegramAccountMessageJob = sendtelegramaccountmessage.New(a)
	a.UpdateTelegramAccountMessageJob = updatetelegramaccountmessage.New(a)
	a.SendTelegramAccountPostJob = sendtelegramaccountpost.New(a)
	a.UpdateTelegramAccountPostJob = updatetelegramaccountpost.New(a)
	a.ImportTelegramChannelJob = importtelegramchannel.New(a)

	a.SendSignInCodeJob = sendsignincode.New(a)
	a.SendFormSubmitEmailJob = sendformsubmit.New(a)
	a.refreshChartDataJob = refreshchartdata.New(a)
	a.SendTelegramPostJob = sendtelegrampost.New(a)
	a.UpdateTelegramPostJob = updatetelegrampost.New(a)
	a.ExtractNotionPagesJob = extractnotionpages.New(a)
	a.UpdateAllChatTelegramPublishPostsJob = updateallchattelegrampublishposts.New(a)
	a.UpdateAllAccountTelegramPublishPostsJob = updateallaccounttelegrampublishposts.New(a)
	a.GenerateNoteVersionEmbeddingJob = generatenoteversionembedding.New(a)
	a.DeliverChangeWebhookJob = deliverchangewebhook.New(a)
	a.DeliverCronWebhookJob = delivercronwebhook.New(a)
	a.webhookHTTPClient = webhookutil.NewClient(a.config.DevMode)
	a.fedHTTPClient = webhookutil.NewClient(a.config.DevMode || a.config.MCPFederationAllowPrivate)

	a.initDebugJobs()
}

// startJobWriters creates the cron jobs (writes UpsertCronJob/DeleteCronJobByName
// and starts the cron scheduler). Writer-only — runs in Block B after the
// writer slot is acquired.
func (a *app) startJobWriters(ctx context.Context) {
	var err error

	a.CronJobs, err = cronjobs.New(ctx, a, getCronJobConfigs(a))
	if err != nil {
		panic(fmt.Errorf("failed to create cron jobs: %w", err))
	}
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

func (a *app) loadAllNotes(ctx context.Context, options noteloader.LoadOptions) error {
	// Whole warmup (live + latest note loaders) shares one budget. Large vaults
	// on a resource-starved box can need well over 10s; too tight a budget makes
	// the startup warmup fail mid-DB-read and panic into a restart crash-loop.
	startCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
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

// systemdListener returns the socket-activated listener passed by systemd via
// the LISTEN_FDS protocol, or nil when not running under socket activation.
// Inheriting the listening socket lets it survive a restart: the kernel keeps
// accepting and queuing connections in the backlog while the new process warms
// up, instead of refusing them — the basis for zero-downtime single-server
// deploys without a load balancer (pair with a trip2g.socket unit).
func (a *app) systemdListener() net.Listener {
	if pid, _ := strconv.Atoi(os.Getenv("LISTEN_PID")); pid != os.Getpid() {
		return nil
	}
	if n, _ := strconv.Atoi(os.Getenv("LISTEN_FDS")); n < 1 {
		return nil
	}

	const sdListenFdsStart = 3 // systemd passes the first socket on fd 3
	f := os.NewFile(sdListenFdsStart, "systemd-listen-fd")
	ln, err := net.FileListener(f)
	_ = f.Close() // net.FileListener duplicates the fd
	if err != nil {
		a.log.Error("socket activation: failed to inherit listener fd", "error", err)
		return nil
	}

	a.log.Info("using systemd socket-activated listener", "fd", sdListenFdsStart)
	return ln
}

// restoreEnvAdapter adapts dependencies for the restore phase (before DB init).
type restoreEnvAdapter struct {
	Storage
	log logger.Logger
}

func (r *restoreEnvAdapter) Logger() logger.Logger {
	return r.log
}

func (r *restoreEnvAdapter) DB() *sql.DB {
	return nil // Not needed for restore
}
