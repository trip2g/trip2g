package defaulttemplate

import (
	"embed"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed langs/*.toml
var langFiles embed.FS

var bundle *i18n.Bundle //nolint:gochecknoglobals // package-level i18n bundle initialized once at startup

// TODO: refactor to Init() error
func init() { //nolint:gochecknoinits // required for embedding and registering translation files at startup
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	entries, _ := langFiles.ReadDir("langs")
	for _, e := range entries {
		if !e.IsDir() {
			data, err := langFiles.ReadFile("langs/" + e.Name())
			if err == nil {
				_, _ = bundle.ParseMessageFileBytes(data, e.Name())
			}
		}
	}
}

// T returns the translation for the given message ID in the given language.
// Falls back to English if the language is not supported or the key is missing.
func T(lang, messageID string) string {
	loc := i18n.NewLocalizer(bundle, lang, "en")
	s, err := loc.Localize(&i18n.LocalizeConfig{MessageID: messageID})
	if err != nil || s == "" {
		return messageID
	}
	return s
}

// T returns the translation for the given message ID using the context's UILang.
func (ctx *Ctx) T(messageID string) string {
	return T(ctx.UILang, messageID)
}
