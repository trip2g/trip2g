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
	"trip2g/internal/dataencryption"
	"trip2g/internal/features"
	"trip2g/internal/gitapi"
	"trip2g/internal/hotauthtoken"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/miniostorage"
	"trip2g/internal/notion"
	"trip2g/internal/patreonjobs"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/tgauthtoken"
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
	DevMode    bool
	LatestLive bool // use the latest release as live version
	AdminJSURL string
	LogLevel   string

	ShutdownGracePeriod time.Duration
	ShutdownTimeout     time.Duration
	InternalListenAddr  string

	// TLS/ACME configuration
	AcmeDomains ArrayFlags

	// MinIO storage configuration
	Storage miniostorage.Config

	// Additional application settings
	PublicURL  string
	OwnerEmail string

	NowpaymentsAPIKey string
	NowpaymentsIPNKey string

	// SMTP
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string

	// Resend
	ResendAPIKey string

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

	Notion notion.Config

	DataEncryption dataencryption.Config

	SimpleBackup SimpleBackupConfig

	// Features configuration (parsed from JSON)
	FeaturesJSON string           // Raw JSON from flag/env
	Features     features.Features // Parsed features
}

// SimpleBackupConfig holds simple backup system configuration.
type SimpleBackupConfig struct {
	Enabled bool
}

// Default values for configuration.
const (
	DefaultListenAddr   = ":8081"
	DefaultDatabaseFile = "data.sqlite3"
	DefaultAdminJSURL   = "/assets/ui/admin/-/web.js"
	DefaultLogLevel     = "info"
	DefaultDevMode      = false

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
		ListenAddr:   DefaultListenAddr,
		DatabaseFile: DefaultDatabaseFile,
		DevMode:      DefaultDevMode,
		AdminJSURL:   DefaultAdminJSURL,
		LogLevel:     DefaultLogLevel,
		AcmeDomains:  ArrayFlags{},
		Storage:      DefaultStorageConfig(),
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
	flag.StringVar(&c.SMTPHost, "smtp-host", "", "SMTP host")
	flag.IntVar(&c.SMTPPort, "smtp-port", 587, "SMTP port")
	flag.StringVar(&c.SMTPUser, "smtp-user", "", "SMTP username")
	flag.StringVar(&c.SMTPPass, "smtp-pass", "", "SMTP password")

	// Resend
	flag.StringVar(&c.ResendAPIKey, "resend-api-key", "", "Resend API key")

	// Mail
	flag.StringVar(&c.MailFrom, "mail-from", "no-reply@resend.trip2g.com", "Email address to use as sender")

	// MD Loader
	flag.BoolVar(&c.MDLoaderConfig.AutoLowerWikilinks, "md-loader-auto-lower-wikilinks", false, "Automatically lower-case wikilinks")

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

	// Notion
	notionDefaults := notion.DefaultConfig()

	flag.DurationVar(&c.Notion.RequestTimeout, "notion-request-timeout", notionDefaults.RequestTimeout, "Notion API request timeout")

	// Data Encryption
	dataEncryptionDefaults := dataencryption.DefaultConfig()

	flag.StringVar(
		&c.DataEncryption.Key,
		"data-encryption-key",
		dataEncryptionDefaults.Key,
		"32-byte key for encrypting sensitive data (AES-256)",
	)
}

func (c *Config) defineServerFlags() {
	flag.StringVar(&c.ListenAddr, "listen-addr", c.ListenAddr, "Listen address")
	flag.StringVar(&c.DatabaseFile, "db-file", c.DatabaseFile, "Database file")
	flag.IntVar(&c.MaxRequestBodySize, "max-request-body-size", 10, "Max request body size in MB")
	flag.BoolVar(&c.LogQueries, "log-queries", c.LogQueries, "Log database queries")
	flag.StringVar(&c.AdminJSURL, "admin-js-url", c.AdminJSURL, "Admin JS URL")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log level")
	flag.BoolVar(&c.DevMode, "dev", c.DevMode, "Development mode")
	flag.BoolVar(&c.LatestLive, "latest-live", c.LatestLive, "Use latest release as live version")
	flag.DurationVar(&c.ShutdownGracePeriod, "shutdown-grace-period", 50*time.Millisecond, "Shutdown grace period")
	flag.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", 1*time.Second, "Shutdown timeout")
	flag.StringVar(&c.InternalListenAddr, "internal-listen-addr", ":8082", "Internal listen address (for health checks etc.)")
	flag.BoolVar(&c.SimpleBackup.Enabled, "simple-backup", false, "Enable simple backup system (hourly backups to S3-compatible storage)")

	// Features configuration
	flag.StringVar(&c.FeaturesJSON, "features", "{}", "Features configuration as JSON")
}

func (c *Config) defineMinioFlags() {
	flag.StringVar(&c.Storage.Endpoint, "minio-endpoint", c.Storage.Endpoint, "MinIO endpoint")
	flag.StringVar(&c.Storage.AccessKey, "minio-access-key-id", c.Storage.AccessKey, "MinIO access key ID")
	flag.StringVar(&c.Storage.SecretKey, "minio-secret-key", c.Storage.SecretKey, "MinIO secret key")
	flag.StringVar(&c.Storage.Bucket, "minio-bucket", c.Storage.Bucket, "MinIO bucket name")
	flag.StringVar(&c.Storage.Region, "minio-region", c.Storage.Region, "MinIO region")
	flag.StringVar(&c.Storage.Prefix, "minio-prefix", c.Storage.Prefix, "MinIO object key prefix")
	flag.BoolVar(&c.Storage.UseSSL, "minio-use-ssl", c.Storage.UseSSL, "Use SSL for MinIO")
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
}

func (c *Config) Prepare() {
	c.PurchaseToken.Secret = c.UserToken.Secret
	c.HotAuthToken.Secret = c.UserToken.Secret
	c.TgAuthToken.Secret = c.UserToken.Secret

	// Parse and validate features (panics on error)
	c.Features = features.Parse(c.FeaturesJSON)
}

// validate checks if the configuration is valid using ozzo validation.
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

		// Storage configuration (delegated to storage's own validation)
		ozzo.Field(&c.Storage, ozzo.By(func(value interface{}) error {
			storage, ok := value.(miniostorage.Config)
			if !ok {
				return errors.New("invalid storage config type")
			}
			return storage.ValidateConfig()
		})),
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
