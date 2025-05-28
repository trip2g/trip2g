package appconfig

import (
	"context"
	"flag"
	"fmt"
	"time"

	"trip2g/internal/logger"
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
	AdminJSURL string
	LogLevel   string

	// TLS/ACME configuration
	AcmeDomains ArrayFlags

	// MinIO storage configuration
	Storage miniostorage.Config

	// Additional application settings
	PublicURL         string
	NowpaymentsAPIKey string
	NowpaymentsIPNKey string
}

// Default values for configuration
const (
	DefaultListenAddr   = ":8081"
	DefaultDatabaseFile = "data.sqlite3"
	DefaultAdminJSURL   = "/assets/ui/admin/-/web.js"
	DefaultLogLevel     = "info"
	DefaultDevMode      = false

	// MinIO defaults
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

	// Define all flags
	if err := cfg.defineFlags(); err != nil {
		return nil, fmt.Errorf("failed to define flags: %w", err)
	}

	// Set up envflag logger if provided
	if log != nil {
		SetLogger(log)
	}

	// Parse with environment variable support using package-level function
	if err := Parse(ctx); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// defineFlags sets up all command-line flags.
func (c *Config) defineFlags() error {
	// Server flags
	flag.StringVar(&c.ListenAddr, "listen-addr", c.ListenAddr, "Listen address")
	flag.StringVar(&c.DatabaseFile, "db-file", c.DatabaseFile, "Database file")
	flag.StringVar(&c.AdminJSURL, "admin-js-url", c.AdminJSURL, "Admin JS URL")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log level")
	flag.BoolVar(&c.DevMode, "dev", c.DevMode, "Development mode")

	// MinIO storage flags
	flag.StringVar(&c.Storage.Endpoint, "minio-endpoint", c.Storage.Endpoint, "MinIO endpoint")
	flag.StringVar(&c.Storage.AccessKey, "minio-access-key-id", c.Storage.AccessKey, "MinIO access key ID")
	flag.StringVar(&c.Storage.SecretKey, "minio-secret-key", c.Storage.SecretKey, "MinIO secret key")
	flag.StringVar(&c.Storage.Bucket, "minio-bucket", c.Storage.Bucket, "MinIO bucket name")
	flag.StringVar(&c.Storage.Region, "minio-region", c.Storage.Region, "MinIO region")
	flag.BoolVar(&c.Storage.UseSSL, "minio-use-ssl", c.Storage.UseSSL, "Use SSL for MinIO")
	flag.DurationVar(&c.Storage.InitTimeout, "minio-init-timeout", c.Storage.InitTimeout, "MinIO init timeout (check and make bucket)")
	flag.DurationVar(&c.Storage.URLExpiresIn, "minio-url-expires-in", c.Storage.URLExpiresIn, "MinIO presigned URL expiration time")

	// ACME domains (multiple values allowed)
	flag.Var(&c.AcmeDomains, "acme-domain", "ACME domains (multiple values allowed)")

	// Additional application settings
	flag.StringVar(&c.PublicURL, "public-url", c.PublicURL, "Public URL for the application")
	flag.StringVar(&c.NowpaymentsAPIKey, "nowpayments-api-key", c.NowpaymentsAPIKey, "Nowpayments API key")
	flag.StringVar(&c.NowpaymentsIPNKey, "nowpayments-ipn-key", c.NowpaymentsIPNKey, "Nowpayments IPN key")

	return nil
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
				return fmt.Errorf("invalid storage config type")
			}
			return storage.ValidateConfig()
		})),
	)
}
