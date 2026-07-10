package appconfig

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"trip2g/internal/auditlogger"
	"trip2g/internal/boostyjobs"
	"trip2g/internal/case/admin/renderpreview"
	cleanupapikeylogs "trip2g/internal/case/cronjob/cleanupapikeylogs"
	"trip2g/internal/dataencryption"
	"trip2g/internal/datasize"
	"trip2g/internal/features"
	"trip2g/internal/gitapi"
	"trip2g/internal/hotauthtoken"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/miniostorage"
	"trip2g/internal/patreonjobs"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/tgauthtoken"
	"trip2g/internal/turnstile"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// ArrayFlags implements flag.Value interface for string slice flags.
type ArrayFlags []string

// String returns the string representation of ArrayFlags.
func (a *ArrayFlags) String() string {
	return fmt.Sprintf("%v", *a)
}

// Set appends a value to ArrayFlags.
func (a *ArrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// Config holds all application configuration.
type Config struct {
	// Server configuration
	ListenAddr         string
	DatabaseFile       string
	MaxRequestBodySize int // in MB

	LogQueries bool

	// Application settings
	DevMode              bool
	AdminJSURL           string
	LogLevel             string
	MaxActiveSignInCodes int

	ShutdownGracePeriod     time.Duration
	ShutdownTimeout         time.Duration
	WriterAcquireTimeout    time.Duration
	GlobalQueuePollInterval time.Duration
	InternalListenAddr      string
	// DebugEmbedding exposes the unauthenticated /debug/embedding endpoint on
	// the internal listener outside dev mode. It can drive arbitrary embedding
	// API/CPU load, so it is off by default in production.
	DebugEmbedding bool
	// LeaderAddr, when non-empty, puts this instance in read-only replica mode:
	// it serves safe (GET/HEAD/OPTIONS) requests locally off a replicated DB and
	// reverse-proxies every mutating request to the leader's internal address
	// (host:port — plain HTTP over the private network). Skips migrations and the
	// writer/queue connections; disables cron and queue workers.
	LeaderAddr                 string
	MCPFederationMaxDepth      int
	MCPFederationAllowPrivate  bool
	MCPFederatedGraphQLEnabled bool
	FederatedFanoutConcurrency int
	FederatedFanoutLimit       int
	FederatedFanoutTimeout     time.Duration

	CronExecuteWebhooksSchedule string
	CronTelegramPublishSchedule string

	// TLS/ACME configuration
	AcmeDomains ArrayFlags

	// StorageBackend selects the file-storage backend: "minio" (default,
	// MinIO/S3) or "local" (local filesystem, no MinIO required).
	StorageBackend string
	// StorageLocalDir is the base directory for the local storage backend.
	// Only used when StorageBackend == "local".
	StorageLocalDir string

	// MinIO storage configuration
	Storage miniostorage.Config

	// Additional application settings
	PublicURL  string
	OwnerEmail string

	NowpaymentsAPIKey string
	NowpaymentsIPNKey string

	// SMTP
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	SMTPStartTLS bool

	// Mail
	MailFrom string

	MDLoaderConfig mdloader.Config

	PatreonJobsConfig patreonjobs.Config
	BoostyJobsConfig  boostyjobs.Config

	AuditLog auditlogger.Config

	UserToken usertoken.Config

	PurchaseToken purchasetoken.Config

	HotAuthToken hotauthtoken.Config

	TgAuthToken tgauthtoken.Config

	GitAPI gitapi.Config

	RenderPreview renderpreview.Config

	APIKeyLogs cleanupapikeylogs.Config

	DataEncryption dataencryption.Config

	SimpleBackup SimpleBackupConfig
	// VacuumCron enables the periodic VACUUM/ANALYZE + wal_checkpoint(TRUNCATE)
	// maintenance job. Off by default: VACUUM is a heavy full-DB rewrite most
	// deployments don't need, and it is incompatible with an external WAL
	// replicator (Litestream owns WAL checkpointing).
	VacuumCron bool

	CronJobs CronJobsConfig

	// Storage limits (0 = no limit). Accept human-readable sizes: "1GB", "500MB".
	StorageDBLimit     datasize.Size
	StorageAssetsLimit datasize.Size

	Metrics MetricsConfig

	// Features configuration (parsed from JSON)
	FeaturesJSON string            // Raw JSON from flag/env
	Features     features.Features // Parsed features

	// Cloudflare Turnstile captcha
	Turnstile turnstile.Config

	// OIDC env-managed provider. When Issuer/ClientID/ClientSecret are all set,
	// a virtual OIDC provider built from these values takes precedence over any
	// DB-configured one. It is never persisted — remove the env vars and it is
	// gone on the next boot.
	OIDC OIDCConfig
}

// OIDCConfig configures an optional environment-managed OIDC login provider.
type OIDCConfig struct {
	Issuer             string
	ClientID           string
	ClientSecret       string
	Scopes             string
	AutoProvision      bool
	AllowedEmailDomain string
	RequiredGroup      string
}

// SimpleBackupConfig holds simple backup system configuration.
type SimpleBackupConfig struct {
	Enabled bool
	// BackupOnShutdown runs a final backup on graceful shutdown. Disable for
	// zero-downtime rolling deploys: a departing instance backing up is
	// redundant (a peer takes over and cron backups continue) and the dump can
	// race the new writer / delay the drain.
	BackupOnShutdown bool
}

// CronJobsConfig holds cron jobs admin configuration.
type CronJobsConfig struct {
	AllowEdit bool
	// RunOnStart runs every ExecuteAfterStart cron job once at boot,
	// asynchronously after the instance is ready. Off by default: instances
	// restart often and re-running these jobs on every boot is needless load.
	RunOnStart bool
}

// Default values for configuration.
const (
	DefaultListenAddr = ":8081"
	// Loopback by default: the internal listener serves unauthenticated
	// /metrics, pprof and debug endpoints and must not be world-exposed.
	DefaultInternalListenAddr   = "127.0.0.1:8082"
	DefaultDatabaseFile         = "data.sqlite3"
	DefaultAdminJSURL           = "/assets/ui/admin/-/web.js"
	DefaultLogLevel             = "info"
	DefaultDevMode              = false
	DefaultMaxActiveSignInCodes = 3

	// Storage backend selection.
	StorageBackendMinIO    = "minio"
	StorageBackendLocal    = "local"
	DefaultStorageBackend  = StorageBackendMinIO
	DefaultStorageLocalDir = "/data/storage"

	// MinIO defaults.
	DefaultMinIOEndpoint    = "localhost:9000"
	DefaultMinIOAccessKey   = "root"
	DefaultMinIOSecretKey   = "password"
	DefaultMinIOBucket      = "development"
	DefaultMinIORegion      = "us-east-1"
	DefaultMinIOUseSSL      = false
	DefaultMinIOInitTimeout = 5 * time.Second
	DefaultMinIOURLExpires  = 6 * 24 * time.Hour // Max 7 days for MinIO presigned URLs
)

// DefaultStorageConfig returns default MinIO storage configuration.
func DefaultStorageConfig() miniostorage.Config {
	return miniostorage.Config{
		Endpoint:     DefaultMinIOEndpoint,
		AccessKey:    DefaultMinIOAccessKey,
		SecretKey:    DefaultMinIOSecretKey,
		Bucket:       DefaultMinIOBucket,
		Region:       DefaultMinIORegion,
		UseSSL:       DefaultMinIOUseSSL,
		InitTimeout:  DefaultMinIOInitTimeout,
		URLExpiresIn: DefaultMinIOURLExpires,
	}
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:           DefaultListenAddr,
		InternalListenAddr:   DefaultInternalListenAddr,
		DatabaseFile:         DefaultDatabaseFile,
		DevMode:              DefaultDevMode,
		AdminJSURL:           DefaultAdminJSURL,
		LogLevel:             DefaultLogLevel,
		MaxActiveSignInCodes: DefaultMaxActiveSignInCodes,
		AcmeDomains:          ArrayFlags{},
		StorageBackend:       DefaultStorageBackend,
		StorageLocalDir:      DefaultStorageLocalDir,
		Storage:              DefaultStorageConfig(),
		Metrics:              DefaultMetricsConfig(),
		RenderPreview:        renderpreview.DefaultConfig(),
		APIKeyLogs:           cleanupapikeylogs.DefaultConfig(),
	}
}

// Get loads configuration from environment variables and command-line flags.
// Environment variables take precedence over defaults, and command-line flags
// take precedence over environment variables.
func Get() (*Config, error) {
	return GetWithLogger(nil)
}

// GetWithLogger loads configuration with a specific logger for the envflag processing.
func GetWithLogger(log logger.Logger) (*Config, error) {
	ctx := context.Background()
	cfg := DefaultConfig()

	// Get env file path from ENV_FILE environment variable, default to .env
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}

	// Load .env file (or custom file if specified)
	err := LoadDotEnvFromPath(envFile)
	if err != nil {
		if log != nil {
			log.Debug("failed to load env file", "file", envFile, "error", err)
		}
		// Don't fail if .env file doesn't exist or has issues
	}

	// Define all flags
	cfg.defineFlags()

	// Set up envflag logger if provided
	if log != nil {
		SetLogger(log)
	}

	// Parse with environment variable support using package-level function
	err = Parse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	cfg.Prepare()

	// Validate configuration
	err = cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// defineFlags sets up all command-line flags.
func (c *Config) defineFlags() {
	// Server flags
	c.defineServerFlags()

	// MinIO storage flags
	c.defineMinioFlags()

	// ACME domains (multiple values allowed)
	flag.Var(&c.AcmeDomains, "acme-domain", "ACME domains (multiple values allowed)")

	// Additional application settings
	flag.StringVar(&c.PublicURL, "public-url", c.PublicURL, "Public URL for the application")
	flag.StringVar(&c.OwnerEmail, "owner-email", c.OwnerEmail, "Owner email for the application")

	flag.StringVar(&c.NowpaymentsAPIKey, "nowpayments-api-key", c.NowpaymentsAPIKey, "Nowpayments API key")
	flag.StringVar(&c.NowpaymentsIPNKey, "nowpayments-ipn-key", c.NowpaymentsIPNKey, "Nowpayments IPN key")

	// SMTP
	flag.StringVar(&c.SMTPHost, "smtp-host", "", "SMTP server host (empty = log only)")
	flag.IntVar(&c.SMTPPort, "smtp-port", 587, "SMTP port")
	flag.StringVar(&c.SMTPUser, "smtp-user", "", "SMTP username")
	flag.StringVar(&c.SMTPPass, "smtp-pass", "", "SMTP password")
	flag.BoolVar(&c.SMTPStartTLS, "smtp-starttls", true, "Use STARTTLS")

	// Mail
	flag.StringVar(&c.MailFrom, "mail-from", "no-reply@example.com", "Email address to use as sender")

	// Patreon jobs configuration
	patreonJobsDefaults := patreonjobs.DefaultConfig()

	flag.DurationVar(
		&c.PatreonJobsConfig.RefreshInterval,
		"patreon-jobs-refresh-interval",
		patreonJobsDefaults.RefreshInterval,
		"Patreon jobs refresh interval",
	)

	flag.DurationVar(
		&c.PatreonJobsConfig.ImmediatelyGap,
		"patreon-jobs-immediately-gap",
		patreonJobsDefaults.ImmediatelyGap,
		"how old synced_at must be to trigger immediate refresh",
	)

	// Boosty jobs configuration
	boostyJobsDefaults := boostyjobs.DefaultConfig()

	flag.DurationVar(
		&c.BoostyJobsConfig.RefreshInterval,
		"boosty-jobs-refresh-interval",
		boostyJobsDefaults.RefreshInterval,
		"Boosty jobs refresh interval",
	)

	flag.DurationVar(
		&c.BoostyJobsConfig.ImmediatelyGap,
		"boosty-jobs-immediately-gap",
		boostyJobsDefaults.ImmediatelyGap,
		"how old synced_at must be to trigger immediate refresh",
	)

	// Audit log configuration
	flag.StringVar(&c.AuditLog.LogLevel, "audit-log-level", "info", "Audit log level")

	// User Token
	userTokenDefaults := usertoken.DefaultConfig()

	flag.StringVar(
		&c.UserToken.CookieName,
		"user-token-cookie-name",
		userTokenDefaults.CookieName,
		"user token cookie name",
	)

	// secret for all jwt tokens
	flag.StringVar(
		&c.UserToken.Secret,
		"jwt-secret",
		userTokenDefaults.Secret,
		"jwt secret for user tokens",
	)

	flag.BoolVar(
		&c.UserToken.Insecure,
		"user-token-insecure",
		userTokenDefaults.Insecure,
		"user token insecure flag (development or http only)",
	)

	flag.DurationVar(
		&c.UserToken.ExpiresIn,
		"user-token-expires-in",
		userTokenDefaults.ExpiresIn,
		"user token expiration duration",
	)

	// Purchase Token
	purchaseTokenDefaults := purchasetoken.DefaultConfig()

	flag.StringVar(
		&c.PurchaseToken.CookieName,
		"purchase-token-cookie-name",
		purchaseTokenDefaults.CookieName,
		"purchase token cookie name",
	)

	flag.DurationVar(
		&c.PurchaseToken.ExpiresIn,
		"purchase-token-expires-in",
		purchaseTokenDefaults.ExpiresIn,
		"purchase token expiration duration",
	)

	// Hot Auth Token
	hotAuthTokenDefaults := hotauthtoken.DefaultConfig()

	flag.DurationVar(
		&c.HotAuthToken.ExpiresIn,
		"hot-auth-token-expires-in",
		hotAuthTokenDefaults.ExpiresIn,
		"hot auth token expiration duration",
	)

	// Telegram Auth Token
	tgAuthTokenDefaults := tgauthtoken.DefaultConfig()

	flag.DurationVar(
		&c.TgAuthToken.ExpiresIn,
		"tg-auth-token-expires-in",
		tgAuthTokenDefaults.ExpiresIn,
		"telegram auth token expiration duration",
	)

	// Git API
	gitAPIDefaults := gitapi.DefaultConfig()

	flag.StringVar(&c.GitAPI.BasePath, "git-api-base-path", gitAPIDefaults.BasePath, "base url path for git API")
	flag.StringVar(&c.GitAPI.RepoPath, "git-api-repo-path", gitAPIDefaults.RepoPath, "path to the git repository")
	flag.StringVar(&c.GitAPI.MasterBranch, "git-api-master-branch", gitAPIDefaults.MasterBranch, "name of the master branch")

	// Data Encryption
	dataEncryptionDefaults := dataencryption.DefaultConfig()

	flag.StringVar(
		&c.DataEncryption.Key,
		"data-encryption-key",
		dataEncryptionDefaults.Key,
		"32-byte key for encrypting sensitive data (AES-256)",
	)

	// Cloudflare Turnstile captcha
	flag.StringVar(&c.Turnstile.SiteKey, "turnstile-site-key", "", "Cloudflare Turnstile site key")
	flag.StringVar(&c.Turnstile.SecretKey, "turnstile-secret-key", "", "Cloudflare Turnstile secret key")

	// OIDC env-managed provider (OIDC_ISSUER / OIDC_CLIENT_ID / OIDC_CLIENT_SECRET ...)
	flag.StringVar(&c.OIDC.Issuer, "oidc-issuer", "", "OIDC issuer URL for the env-managed provider")
	flag.StringVar(&c.OIDC.ClientID, "oidc-client-id", "", "OIDC client id for the env-managed provider")
	flag.StringVar(&c.OIDC.ClientSecret, "oidc-client-secret", "", "OIDC client secret for the env-managed provider")
	flag.StringVar(&c.OIDC.Scopes, "oidc-scopes", "openid email profile", "OIDC scopes for the env-managed provider")
	flag.BoolVar(&c.OIDC.AutoProvision, "oidc-auto-provision", false, "auto-create users on first OIDC login (env-managed provider)")
	flag.StringVar(&c.OIDC.AllowedEmailDomain, "oidc-allowed-email-domain", "", "restrict OIDC logins to this email domain (env-managed provider)")
	flag.StringVar(&c.OIDC.RequiredGroup, "oidc-required-group", "", "restrict OIDC logins to members of this group (env-managed provider)")

	// Metrics
	c.defineMetricsFlags()
}

func (c *Config) defineServerFlags() {
	flag.StringVar(&c.ListenAddr, "listen-addr", c.ListenAddr, "Listen address")
	flag.StringVar(&c.DatabaseFile, "db-file", c.DatabaseFile, "Database file")
	flag.IntVar(&c.MaxRequestBodySize, "max-request-body-size", 10, "Max request body size in MB")
	flag.BoolVar(&c.LogQueries, "log-queries", c.LogQueries, "Log database queries")
	flag.StringVar(&c.AdminJSURL, "admin-js-url", c.AdminJSURL, "Admin JS URL")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log level")
	flag.BoolVar(&c.DevMode, "dev", c.DevMode, "Development mode")
	flag.DurationVar(&c.ShutdownGracePeriod, "shutdown-grace-period", 200*time.Millisecond, "Shutdown grace period")
	flag.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", 1*time.Second, "Shutdown timeout")
	flag.DurationVar(&c.WriterAcquireTimeout, "writer-acquire-timeout", 20*time.Second,
		"Max time to wait for the SQLite writer slot before starting writer subsystems")
	flag.DurationVar(&c.GlobalQueuePollInterval, "global-queue-poll-interval", 3*time.Second, "Poll interval for the global background job queue")
	flag.StringVar(
		&c.CronExecuteWebhooksSchedule,
		"cron-execute-webhooks-schedule",
		"0 * * * * *",
		"Cron schedule (6-field, with seconds) for the execute_cron_webhooks job. Default every minute; the public cloud sets it hourly.",
	)
	flag.StringVar(
		&c.CronTelegramPublishSchedule,
		"cron-telegram-publish-schedule",
		"0 * * * * *",
		"Cron schedule (6-field, with seconds) for the send_scheduled_telegram_publishposts job. Default every minute; the public cloud sets it hourly.",
	)
	flag.StringVar(&c.InternalListenAddr, "internal-listen-addr", c.InternalListenAddr, "Internal listen address (for health checks etc.)")
	flag.BoolVar(&c.DebugEmbedding, "debug-embedding", false,
		"Expose the unauthenticated /debug/embedding endpoint on the internal listener (always on in --dev)")
	flag.StringVar(
		&c.LeaderAddr,
		"leader-addr",
		"",
		"Leader internal address as host:port (the leader's --internal-listen-addr, plain HTTP over the private network). When set, this instance runs as a read-only replica: serves reads locally and forwards writes to the leader.",
	)
	flag.IntVar(&c.MCPFederationMaxDepth, "mcp-federation-max-depth", 3, "Max MCP federation fan-out depth")
	flag.BoolVar(
		&c.MCPFederationAllowPrivate,
		"mcp-federation-allow-private",
		false,
		"Disable SSRF protection for federation calls (allow private/internal addresses).",
	)
	flag.BoolVar(&c.MCPFederatedGraphQLEnabled, "mcp-federated-graphql", false,
		"Enable the federated_graphql_request MCP tool (query-only, subgraph-scoped). Off by default.")
	flag.IntVar(&c.FederatedFanoutConcurrency, "federated-fanout-concurrency", 5,
		"Max peers queried in parallel by a federated_search fan-out (0 = unlimited)")
	flag.IntVar(&c.FederatedFanoutLimit, "federated-fanout-limit", 7,
		"Max peers a blind (no kb_id) federated_search fan-out touches; the rest are reported as skipped (0 = unlimited)")
	flag.DurationVar(&c.FederatedFanoutTimeout, "federated-fanout-timeout", 5*time.Second,
		"Per-peer timeout for federated fan-out requests (0 = no extra timeout)")
	flag.BoolVar(&c.SimpleBackup.Enabled, "simple-backup", false, "Enable simple backup system (hourly backups to S3-compatible storage)")
	flag.BoolVar(
		&c.SimpleBackup.BackupOnShutdown,
		"simple-backup-on-shutdown",
		true,
		"Run a final backup on graceful shutdown. Set false for zero-downtime rolling deploys where a peer takes over.",
	)
	flag.BoolVar(
		&c.VacuumCron,
		"vacuum-cron",
		false,
		"Enable the periodic VACUUM/ANALYZE database maintenance cron. Off by default (heavy full-DB rewrite; incompatible with Litestream WAL replication).",
	)
	flag.BoolVar(&c.CronJobs.AllowEdit, "cronjobs-allow-edit", false, "Allow admin to edit cron job schedule and enabled state")
	flag.BoolVar(
		&c.CronJobs.RunOnStart,
		"cronjobs-run-on-start",
		false,
		"Run ExecuteAfterStart cron jobs once at boot, async after ready (env CRONJOBS_RUN_ON_START). Off by default.",
	)

	// Storage limits.
	datasize.FlagVar(
		flag.CommandLine, &c.StorageDBLimit, "storage-db-limit",
		100*datasize.Megabyte, "SQLite database size limit, e.g. 1GB, 500MB (0 = no limit)",
	)
	datasize.FlagVar(
		flag.CommandLine, &c.StorageAssetsLimit, "storage-assets-limit",
		datasize.Gigabyte, "Total note assets size limit, e.g. 2GB (0 = no limit)",
	)

	// Features configuration
	flag.StringVar(&c.FeaturesJSON, "features", "{}", "Features configuration as JSON")

	flag.IntVar(&c.MaxActiveSignInCodes, "max-active-signin-codes", DefaultMaxActiveSignInCodes,
		"Max active (unconsumed, unexpired) sign-in codes per user before requestEmailSignInCode is rejected")
}

func (c *Config) defineMinioFlags() {
	flag.StringVar(
		&c.StorageBackend,
		"storage-backend",
		c.StorageBackend,
		"File storage backend: minio (S3/MinIO) or local (local filesystem, no MinIO required)",
	)
	flag.StringVar(
		&c.StorageLocalDir,
		"storage-local-dir",
		c.StorageLocalDir,
		"Base directory for the local storage backend (used when storage-backend=local)",
	)
	flag.StringVar(&c.Storage.Endpoint, "minio-endpoint", c.Storage.Endpoint, "MinIO endpoint")
	flag.StringVar(&c.Storage.AccessKey, "minio-access-key-id", c.Storage.AccessKey, "MinIO access key ID")
	flag.StringVar(&c.Storage.SecretKey, "minio-secret-key", c.Storage.SecretKey, "MinIO secret key")
	flag.StringVar(&c.Storage.Bucket, "minio-bucket", c.Storage.Bucket, "MinIO bucket name")
	flag.StringVar(&c.Storage.Region, "minio-region", c.Storage.Region, "MinIO region")
	flag.StringVar(&c.Storage.Prefix, "minio-prefix", c.Storage.Prefix, "MinIO object key prefix")
	flag.BoolVar(&c.Storage.UseSSL, "minio-use-ssl", c.Storage.UseSSL, "Use SSL for MinIO")
	flag.StringVar(
		&c.Storage.PublicURL,
		"minio-public-url",
		c.Storage.PublicURL,
		"Override scheme and host in presigned URLs (e.g. https://storage.example.com)",
	)
	flag.DurationVar(
		&c.Storage.InitTimeout,
		"minio-init-timeout",
		c.Storage.InitTimeout,
		"MinIO init timeout (check and make bucket)",
	)
	flag.DurationVar(
		&c.Storage.URLExpiresIn,
		"minio-url-expires-in",
		c.Storage.URLExpiresIn,
		"MinIO presigned URL expiration time",
	)
	flag.DurationVar(
		&c.APIKeyLogs.Retention,
		"api-key-logs-retention",
		c.APIKeyLogs.Retention,
		"How long to keep API key access logs (default 90d)",
	)
}

// IsReadReplica reports whether this instance runs in read-only replica mode
// (a leader URL was configured). In this mode migrations and the writer/queue
// connections are skipped and mutating requests are forwarded to the leader.
func (c *Config) IsReadReplica() bool {
	return c.LeaderAddr != ""
}

func (c *Config) Prepare() {
	c.PurchaseToken.Secret = c.UserToken.Secret
	c.HotAuthToken.Secret = c.UserToken.Secret
	c.TgAuthToken.Secret = c.UserToken.Secret

	// Parse and validate features (panics on error)
	c.Features = features.Parse(c.FeaturesJSON)
}

// CronScheduleOverride returns a cron schedule override for the named job from
// the environment — <JOB_NAME>_SCHEDULE (or TRIP2G_<JOB_NAME>_SCHEDULE) — or ""
// if unset. Centralizes env access so callers don't read os.Getenv directly.
func (c *Config) CronScheduleOverride(jobName string) string {
	key := strings.ToUpper(jobName) + "_SCHEDULE"
	if v := os.Getenv("TRIP2G_" + key); v != "" {
		return v
	}
	return os.Getenv(key)
}

func (c *Config) validate() error {
	if !c.DevMode {
		if c.UserToken.Secret == usertoken.DefaultConfig().Secret {
			panic("in production, user token secret must be changed from default")
		}

		if c.DataEncryption.Key == dataencryption.DefaultConfig().Key {
			panic("in production, data encryption key must be changed from default")
		}
	}

	return ozzo.ValidateStruct(c,
		// Server configuration
		ozzo.Field(&c.ListenAddr, ozzo.Required),
		ozzo.Field(&c.DatabaseFile, ozzo.Required),
		ozzo.Field(&c.LogLevel, ozzo.Required, ozzo.In("debug", "info", "warn", "error")),
		ozzo.Field(&c.AdminJSURL, ozzo.Required),

		// URLs should be valid if provided
		ozzo.Field(&c.PublicURL, ozzo.When(c.PublicURL != "", is.URL)),
		ozzo.Field(&c.LeaderAddr, ozzo.When(c.LeaderAddr != "", is.DialString)),

		// Storage backend selection.
		ozzo.Field(&c.StorageBackend, ozzo.Required, ozzo.In(StorageBackendMinIO, StorageBackendLocal)),

		// Local storage requires a base directory.
		ozzo.Field(&c.StorageLocalDir, ozzo.When(c.StorageBackend == StorageBackendLocal, ozzo.Required)),

		// MinIO storage configuration (delegated to storage's own validation).
		// Only validated for the MinIO backend; the local backend needs no MinIO config.
		ozzo.Field(&c.Storage, ozzo.When(c.StorageBackend == StorageBackendMinIO, ozzo.By(func(value interface{}) error {
			storage, ok := value.(miniostorage.Config)
			if !ok {
				return errors.New("invalid storage config type")
			}
			return storage.ValidateConfig()
		}))),
	)
}

// LoadDotEnvFromPath loads environment variables from a specific .env file path.
// It doesn't override existing environment variables.
func LoadDotEnvFromPath(path string) error {
	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf(".env file not found at %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format in .env file at line %d: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Only set if environment variable doesn't already exist
		_, exists := os.LookupEnv(key)
		if !exists {
			err = os.Setenv(key, value)
			if err != nil {
				return fmt.Errorf("failed to set environment variable %s: %w", key, err)
			}
		}
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("error reading .env file: %w", err)
	}

	return nil
}
