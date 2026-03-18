package defaulttemplate

import (
	"embed"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed langs/*.toml
var langFiles embed.FS

//nolint:gochecknoglobals // package-level bundle initialized via Init()
var bundle *i18n.Bundle

// Init loads all translation files. Must be called once at startup before T is used.
func Init() error {
	b := i18n.NewBundle(language.English)
	b.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	entries, err := langFiles.ReadDir("langs")
	if err != nil {
		return fmt.Errorf("failed to read lang files: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			data, readErr := langFiles.ReadFile("langs/" + e.Name())
			if readErr != nil {
				return fmt.Errorf("failed to read lang file %s: %w", e.Name(), readErr)
			}
			if _, parseErr := b.ParseMessageFileBytes(data, e.Name()); parseErr != nil {
				return fmt.Errorf("failed to parse lang file %s: %w", e.Name(), parseErr)
			}
		}
	}
	bundle = b
	return nil
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
