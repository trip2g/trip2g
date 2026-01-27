// Package configregistry provides metadata for atomic config values.
package configregistry

// ConfigType represents the type of a config value.
type ConfigType string

const (
	ConfigTypeString ConfigType = "string"
	ConfigTypeBool   ConfigType = "bool"
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
		Description: "Содержимое robots.txt. Значения: open, closed или произвольный текст.",
		Type:        ConfigTypeString,
		Default:     "open",
		Validate:    nil,
	},
	ConfigShowDraftVersions: {
		ID:          ConfigShowDraftVersions,
		Description: "Показывать черновики админам.",
		Type:        ConfigTypeBool,
		Default:     true,
		Validate:    nil,
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

// Get returns config metadata by ID.
func Get(id string) (ConfigMeta, bool) {
	meta, ok := Registry[id]
	return meta, ok
}
