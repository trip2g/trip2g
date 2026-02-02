package model

// SiteConfig contains all site-wide configuration values with defaults.
type SiteConfig struct {
	SiteTitleTemplate string
	Timezone          string
	DefaultLayout     string
	RobotsTxt         string
	ShowDraftVersions bool
	EnableRSS         bool
}
