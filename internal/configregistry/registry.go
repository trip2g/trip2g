// Package configregistry provides metadata for atomic config values.
package configregistry

import "fmt"

// ConfigType represents the type of a config value.
type ConfigType string

const (
	ConfigTypeString ConfigType = "string"
	ConfigTypeBool   ConfigType = "bool"
	ConfigTypeInt    ConfigType = "int"
)

// ConfigMeta contains metadata for a config value.
type ConfigMeta struct {
	ID          string
	Description string
	Type        ConfigType
	Default     interface{}
	Validate    func(value interface{}) error
}

// String config IDs.
const (
	ConfigSiteTitleTemplate = "site_title_template"
	ConfigTimezone          = "timezone"
	ConfigDefaultLayout     = "default_layout"
	ConfigRobotsTxt         = "robots_txt"
)

// Bool config IDs.
const (
	ConfigShowDraftVersions = "show_draft_versions"
	ConfigEnableRSS         = "enable_rss"
)

// Int config IDs.
const (
	ConfigVectorMinSimilarity    = "vector_min_similarity"
	ConfigCaptchaSigninThreshold = "captcha_signin_threshold"
)

// Registry contains all config metadata.
//
//nolint:gochecknoglobals // intentional global registry for config metadata.
var Registry = map[string]ConfigMeta{
	ConfigSiteTitleTemplate: {
		ID:          ConfigSiteTitleTemplate,
		Description: "Формат заголовка страницы. %s заменяется на название страницы.",
		Type:        ConfigTypeString,
		Default:     "%s",
		Validate:    validateSiteTitleTemplate,
	},
	ConfigTimezone: {
		ID:          ConfigTimezone,
		Description: "Часовой пояс для отображения дат.",
		Type:        ConfigTypeString,
		Default:     "UTC",
		Validate:    validateTimezone,
	},
	ConfigDefaultLayout: {
		ID:          ConfigDefaultLayout,
		Description: "Layout по умолчанию для страниц.",
		Type:        ConfigTypeString,
		Default:     "",
		Validate:    nil,
	},
	ConfigRobotsTxt: {
		ID:          ConfigRobotsTxt,
		Description: "Содержимое robots.txt. Значения: opened, closed или произвольный текст.",
		Type:        ConfigTypeString,
		Default:     "opened",
		Validate:    nil,
	},
	ConfigShowDraftVersions: {
		ID:          ConfigShowDraftVersions,
		Description: "Показывать черновики админам.",
		Type:        ConfigTypeBool,
		Default:     true,
		Validate:    nil,
	},
	ConfigEnableRSS: {
		ID:          ConfigEnableRSS,
		Description: "Включить RSS-ленту для заметок.",
		Type:        ConfigTypeBool,
		Default:     true,
		Validate:    nil,
	},
	ConfigVectorMinSimilarity: {
		ID:          ConfigVectorMinSimilarity,
		Description: "Минимальная похожесть для векторного поиска (1–1000, делится на 1000).",
		Type:        ConfigTypeInt,
		Default:     820,
		Validate:    validateIntRange(1, 1000),
	},
	ConfigCaptchaSigninThreshold: {
		ID:          ConfigCaptchaSigninThreshold,
		Description: "Maximum sign-in code requests per hour before captcha is required.",
		Type:        ConfigTypeInt,
		Default:     5,
		Validate:    validateIntRange(1, 10000),
	},
}

// StringConfigs returns all string config IDs.
func StringConfigs() []string {
	var result []string
	for id, meta := range Registry {
		if meta.Type == ConfigTypeString {
			result = append(result, id)
		}
	}
	return result
}

// BoolConfigs returns all bool config IDs.
func BoolConfigs() []string {
	var result []string
	for id, meta := range Registry {
		if meta.Type == ConfigTypeBool {
			result = append(result, id)
		}
	}
	return result
}

// IntConfigs returns all int config IDs.
func IntConfigs() []string {
	var result []string
	for id, meta := range Registry {
		if meta.Type == ConfigTypeInt {
			result = append(result, id)
		}
	}
	return result
}

// Get returns config metadata by ID.
func Get(id string) (ConfigMeta, bool) {
	meta, ok := Registry[id]
	return meta, ok
}

func validateIntRange(minVal, maxVal int) func(interface{}) error {
	return func(value interface{}) error {
		v, ok := value.(int)
		if !ok {
			return fmt.Errorf("expected int, got %T", value)
		}
		if v < minVal || v > maxVal {
			return fmt.Errorf("value must be between %d and %d", minVal, maxVal)
		}
		return nil
	}
}
