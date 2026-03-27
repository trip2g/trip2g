package model

// SiteConfig contains all site-wide configuration values with defaults.
type SiteConfig struct {
	SiteTitleTemplate    string
	Timezone             string
	DefaultLayout        string
	RobotsTxt            string
	ShowDraftVersions    bool
	EnableRSS            bool
	VectorMinSimilarity      int // 1–1000, divide by 1000 to get float threshold
	URLNormalizationMethod   URLNormalizationMethod
}
