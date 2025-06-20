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

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/miniostorage"

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
	ListenAddr   string
	DatabaseFile string

	// Application settings
	DevMode    bool
	LatestLive bool // use the latest release as live version
	AdminJSURL string
	LogLevel   string

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
	DefaultMinIOURLExpires  = 10 * time.Minute
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

	// Load .env file if it exists
	err := loadDotEnv()
	if err != nil {
		if log != nil {
			log.Debug("failed to load .env file", "error", err)
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
	flag.StringVar(&c.ListenAddr, "listen-addr", c.ListenAddr, "Listen address")
	flag.StringVar(&c.DatabaseFile, "db-file", c.DatabaseFile, "Database file")
	flag.StringVar(&c.AdminJSURL, "admin-js-url", c.AdminJSURL, "Admin JS URL")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log level")
	flag.BoolVar(&c.DevMode, "dev", c.DevMode, "Development mode")
	flag.BoolVar(&c.LatestLive, "latest-live", c.LatestLive, "Use latest release as live version")

	// MinIO storage flags
	flag.StringVar(&c.Storage.Endpoint, "minio-endpoint", c.Storage.Endpoint, "MinIO endpoint")
	flag.StringVar(&c.Storage.AccessKey, "minio-access-key-id", c.Storage.AccessKey, "MinIO access key ID")
	flag.StringVar(&c.Storage.SecretKey, "minio-secret-key", c.Storage.SecretKey, "MinIO secret key")
	flag.StringVar(&c.Storage.Bucket, "minio-bucket", c.Storage.Bucket, "MinIO bucket name")
	flag.StringVar(&c.Storage.Region, "minio-region", c.Storage.Region, "MinIO region")
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
}

// validate checks if the configuration is valid using ozzo validation.
func (c *Config) validate() error {
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

// loadDotEnv loads environment variables from .env file in the current directory.
// It doesn't override existing environment variables.
func loadDotEnv() error {
	return loadDotEnvFromPath(".env")
}

// loadDotEnvFromPath loads environment variables from a specific .env file path.
// It doesn't override existing environment variables.
func loadDotEnvFromPath(path string) error {
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
