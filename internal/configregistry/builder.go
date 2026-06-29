package configregistry

import (
	"context"
	"sync/atomic"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// BuilderEnv provides DB access and logging for the SiteConfigBuilder.
type BuilderEnv interface {
	AllLatestConfigStrings(ctx context.Context) ([]db.AllLatestConfigStringsRow, error)
	AllLatestConfigBools(ctx context.Context) ([]db.AllLatestConfigBoolsRow, error)
	AllLatestConfigInts(ctx context.Context) ([]db.AllLatestConfigIntsRow, error)
	Logger() logger.Logger
}

// SiteConfigBuilder builds and caches SiteConfig from the registry and DB.
type SiteConfigBuilder struct {
	env   BuilderEnv
	log   logger.Logger
	cache atomic.Pointer[model.SiteConfig]

	// configEpoch is a monotonic counter bumped on every InvalidateSiteConfig.
	// The anonymous page cache embeds it in its key so any site-config change
	// (which can affect site name, RSS, default layout, paywall, etc.) makes
	// previously cached pages unreachable without an explicit flush.
	configEpoch atomic.Uint64
}

// NewSiteConfigBuilder creates a new SiteConfigBuilder.
func NewSiteConfigBuilder(env BuilderEnv) *SiteConfigBuilder {
	return &SiteConfigBuilder{
		env: env,
		log: logger.WithPrefix(env.Logger(), "configregistry:"),
	}
}

// SiteConfig returns the cached SiteConfig, building it on first call or after invalidation.
func (b *SiteConfigBuilder) SiteConfig(ctx context.Context) model.SiteConfig {
	if cfg := b.cache.Load(); cfg != nil {
		return *cfg
	}

	cfg := b.buildSiteConfig(ctx)
	b.cache.Store(&cfg)

	return cfg
}

// InvalidateSiteConfig resets the cached SiteConfig so it is rebuilt on next
// access and bumps the config epoch so anonymous page-cache entries built under
// the previous config are no longer served.
func (b *SiteConfigBuilder) InvalidateSiteConfig() {
	b.cache.Store(nil)
	b.configEpoch.Add(1)
}

// ConfigEpoch returns the current site-config epoch, bumped on every
// InvalidateSiteConfig. Used as part of the anonymous page-cache key.
func (b *SiteConfigBuilder) ConfigEpoch() uint64 {
	return b.configEpoch.Load()
}

// TimeLocation returns the parsed timezone from the cached SiteConfig.
// Kept for backward compatibility with callers that don't have a context.
func (b *SiteConfigBuilder) TimeLocation() *time.Location {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return b.SiteConfig(ctx).TimeLocation
}

// buildSiteConfig constructs SiteConfig from registry defaults + DB values.
func (b *SiteConfigBuilder) buildSiteConfig(ctx context.Context) model.SiteConfig {
	cfg := model.SiteConfig{}

	// Apply defaults from registries.
	for _, meta := range StringRegistry {
		if meta.SetupFunc != nil {
			meta.SetupFunc(&cfg, meta.Default)
		}
	}

	for _, meta := range BoolRegistry {
		if meta.SetupFunc != nil {
			meta.SetupFunc(&cfg, meta.Default)
		}
	}

	for _, meta := range IntRegistry {
		if meta.SetupFunc != nil {
			meta.SetupFunc(&cfg, meta.Default)
		}
	}

	// Apply DB overrides.
	b.applyStringValues(ctx, &cfg)
	b.applyBoolValues(ctx, &cfg)
	b.applyIntValues(ctx, &cfg)

	// Compute TimeLocation from Timezone.
	cfg.TimeLocation = b.parseTimeLocation(cfg.Timezone)

	return cfg
}

func (b *SiteConfigBuilder) applyStringValues(ctx context.Context, cfg *model.SiteConfig) {
	rows, err := b.env.AllLatestConfigStrings(ctx)
	if err != nil {
		b.log.Error("failed to load config strings", "error", err)
		return
	}

	for _, row := range rows {
		meta, ok := StringRegistry[row.ValueID]
		if !ok {
			b.log.Warn("unknown string config key in DB", "key", row.ValueID)
			continue
		}

		if meta.SetupFunc == nil {
			continue
		}

		if meta.Validate != nil {
			if validErr := meta.Validate(row.Value); validErr != nil {
				b.log.Warn("invalid config value, keeping default", "key", row.ValueID, "error", validErr)
				continue
			}
		}

		meta.SetupFunc(cfg, row.Value)
	}
}

func (b *SiteConfigBuilder) applyBoolValues(ctx context.Context, cfg *model.SiteConfig) {
	rows, err := b.env.AllLatestConfigBools(ctx)
	if err != nil {
		b.log.Error("failed to load config bools", "error", err)
		return
	}

	for _, row := range rows {
		meta, ok := BoolRegistry[row.ValueID]
		if !ok {
			b.log.Warn("unknown bool config key in DB", "key", row.ValueID)
			continue
		}

		if meta.SetupFunc == nil {
			continue
		}

		if meta.Validate != nil {
			if validErr := meta.Validate(row.Value); validErr != nil {
				b.log.Warn("invalid config value, keeping default", "key", row.ValueID, "error", validErr)
				continue
			}
		}

		meta.SetupFunc(cfg, row.Value)
	}
}

func (b *SiteConfigBuilder) applyIntValues(ctx context.Context, cfg *model.SiteConfig) {
	rows, err := b.env.AllLatestConfigInts(ctx)
	if err != nil {
		b.log.Error("failed to load config ints", "error", err)
		return
	}

	for _, row := range rows {
		meta, ok := IntRegistry[row.ValueID]
		if !ok {
			b.log.Warn("unknown int config key in DB", "key", row.ValueID)
			continue
		}

		if meta.SetupFunc == nil {
			continue
		}

		intVal := int(row.Value)

		if meta.Validate != nil {
			if validErr := meta.Validate(intVal); validErr != nil {
				b.log.Warn("invalid config value, keeping default", "key", row.ValueID, "error", validErr)
				continue
			}
		}

		meta.SetupFunc(cfg, intVal)
	}
}

func (b *SiteConfigBuilder) parseTimeLocation(timezone string) *time.Location {
	if timezone == "" {
		return time.UTC
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		b.log.Error("failed to load timezone location, using UTC", "timezone", timezone, "error", err)
		return time.UTC
	}

	return loc
}
