// Package configregistry provides metadata for atomic config values.
package configregistry

import (
	"fmt"

	"trip2g/internal/model"
)

// ConfigMeta contains base metadata shared by all config types.
type ConfigMeta struct {
	ID          string
	Description string
}

// StringConfigMeta is metadata for a string config value.
type StringConfigMeta struct {
	ConfigMeta
	Default   string
	Validate  func(value string) error
	SetupFunc func(cfg *model.SiteConfig, value string)
}

// BoolConfigMeta is metadata for a bool config value.
type BoolConfigMeta struct {
	ConfigMeta
	Default   bool
	Validate  func(value bool) error
	SetupFunc func(cfg *model.SiteConfig, value bool)
}

// IntConfigMeta is metadata for an int config value.
type IntConfigMeta struct {
	ConfigMeta
	Default   int
	Validate  func(value int) error
	SetupFunc func(cfg *model.SiteConfig, value int)
}

// String config IDs.
const (
	ConfigSiteTitleTemplate      = "site_title_template"
	ConfigTimezone               = "timezone"
	ConfigDefaultLayout          = "default_layout"
	ConfigURLNormalizationMethod = "url_normalization_method"
	ConfigWikilinkResolution     = "wikilink_resolution"
	ConfigKBID                   = "kb_id"
)

// Bool config IDs.
const (
	ConfigShowDraftVersions      = "show_draft_versions"
	ConfigEnableRSS              = "enable_rss"
	ConfigEnableNotFoundTracking = "enable_not_found_tracking"
)

// Int config IDs.
const (
	ConfigVectorMinSimilarity    = "vector_min_similarity"
	ConfigCaptchaSigninThreshold = "captcha_signin_threshold"
)

// Typed registries.
//
//nolint:gochecknoglobals // intentional global registries for config metadata.
var (
	StringRegistry = map[string]StringConfigMeta{
		ConfigSiteTitleTemplate: {
			ConfigMeta: ConfigMeta{ID: ConfigSiteTitleTemplate, Description: "Page title format. %s is replaced with the page name."},
			Default:    "%s",
			Validate:   validateSiteTitleTemplate,
			SetupFunc:  func(cfg *model.SiteConfig, v string) { cfg.SiteTitleTemplate = v },
		},
		ConfigTimezone: {
			ConfigMeta: ConfigMeta{ID: ConfigTimezone, Description: "Timezone for displaying dates."},
			Default:    "UTC",
			Validate:   validateTimezone,
			SetupFunc:  func(cfg *model.SiteConfig, v string) { cfg.Timezone = v },
		},
		ConfigDefaultLayout: {
			ConfigMeta: ConfigMeta{ID: ConfigDefaultLayout, Description: "Default layout for pages."},
			Default:    "",
			SetupFunc:  func(cfg *model.SiteConfig, v string) { cfg.DefaultLayout = v },
		},
		ConfigURLNormalizationMethod: {
			ConfigMeta: ConfigMeta{ID: ConfigURLNormalizationMethod, Description: "URL normalization method for note permalinks."},
			Default:    string(model.DefaultURLNormalizationMethod),
			Validate:   validateURLNormalizationMethod,
			SetupFunc: func(cfg *model.SiteConfig, v string) {
				cfg.URLNormalizationMethod = model.URLNormalizationMethod(v)
			},
		},
		ConfigWikilinkResolution: {
			ConfigMeta: ConfigMeta{
				ID:          ConfigWikilinkResolution,
				Description: "Bare wikilink resolution: global (default, Obsidian-compatible shallowest-only) or scoped (same folder → same language → shallowest).",
			},
			Default:  string(model.DefaultWikilinkResolution),
			Validate: validateWikilinkResolution,
			SetupFunc: func(cfg *model.SiteConfig, v string) {
				cfg.WikilinkResolution = model.WikilinkResolution(v)
			},
		},
		ConfigKBID: {
			ConfigMeta: ConfigMeta{ID: ConfigKBID, Description: "Federation KB identifier. Falls back to the public URL host when empty."},
			Default:    "",
			SetupFunc:  func(cfg *model.SiteConfig, v string) { cfg.KBID = v },
		},
	}

	BoolRegistry = map[string]BoolConfigMeta{
		ConfigShowDraftVersions: {
			ConfigMeta: ConfigMeta{ID: ConfigShowDraftVersions, Description: "Show the latest unreleased versions to everyone (bypass releases)."},
			Default:    true,
			SetupFunc:  func(cfg *model.SiteConfig, v bool) { cfg.ShowDraftVersions = v },
		},
		ConfigEnableRSS: {
			ConfigMeta: ConfigMeta{ID: ConfigEnableRSS, Description: "Enable RSS feed for notes."},
			Default:    true,
			SetupFunc:  func(cfg *model.SiteConfig, v bool) { cfg.EnableRSS = v },
		},
		ConfigEnableNotFoundTracking: {
			ConfigMeta: ConfigMeta{
				ID:          ConfigEnableNotFoundTracking,
				Description: "Track 404s (not_found_paths, not_found_ip_hits) for review in the admin panel.",
			},
			Default:   false,
			SetupFunc: func(cfg *model.SiteConfig, v bool) { cfg.EnableNotFoundTracking = v },
		},
	}

	IntRegistry = map[string]IntConfigMeta{
		ConfigVectorMinSimilarity: {
			ConfigMeta: ConfigMeta{ID: ConfigVectorMinSimilarity, Description: "Minimum similarity for vector search (1–1000, divided by 1000)."},
			Default:    820,
			Validate:   validateIntRange(1, 1000),
			SetupFunc:  func(cfg *model.SiteConfig, v int) { cfg.VectorMinSimilarity = v },
		},
		ConfigCaptchaSigninThreshold: {
			ConfigMeta: ConfigMeta{ID: ConfigCaptchaSigninThreshold, Description: "Maximum sign-in code requests per hour before captcha is required."},
			Default:    5,
			Validate:   validateIntRange(1, 10000),
		},
	}
)

// Get returns base config metadata by ID (looks up all registries).
func Get(id string) (ConfigMeta, bool) {
	if m, ok := StringRegistry[id]; ok {
		return m.ConfigMeta, true
	}

	if m, ok := BoolRegistry[id]; ok {
		return m.ConfigMeta, true
	}

	if m, ok := IntRegistry[id]; ok {
		return m.ConfigMeta, true
	}

	return ConfigMeta{}, false
}

// GetString returns string config metadata by ID.
func GetString(id string) (StringConfigMeta, bool) {
	m, ok := StringRegistry[id]
	return m, ok
}

// GetBool returns bool config metadata by ID.
func GetBool(id string) (BoolConfigMeta, bool) {
	m, ok := BoolRegistry[id]
	return m, ok
}

// GetInt returns int config metadata by ID.
func GetInt(id string) (IntConfigMeta, bool) {
	m, ok := IntRegistry[id]
	return m, ok
}

func validateIntRange(minVal, maxVal int) func(int) error {
	return func(value int) error {
		if value < minVal || value > maxVal {
			return fmt.Errorf("value must be between %d and %d", minVal, maxVal)
		}
		return nil
	}
}
