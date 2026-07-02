package model

import "time"

// SiteConfig contains all site-wide configuration values.
// Defaults and DB→field mapping are defined in configregistry via SetupFunc.
type SiteConfig struct {
	SiteTitleTemplate      string
	Timezone               string
	DefaultLayout          string
	ShowDraftVersions      bool
	EnableRSS              bool
	VectorMinSimilarity    int // 1–1000, divide by 1000 to get float threshold
	URLNormalizationMethod URLNormalizationMethod
	KBID                   string         // federation KB identifier; falls back to the public URL host when empty
	TimeLocation           *time.Location // computed from Timezone, not stored in DB
}
