package model

import (
	"fmt"
	"strings"
)

// TelegramRichMode is the value of the `telegram_rich` frontmatter key.
type TelegramRichMode string

const (
	TelegramRichAuto TelegramRichMode = "auto"
	TelegramRichOn   TelegramRichMode = "on"
	TelegramRichOff  TelegramRichMode = "off"
)

// UseRich reports whether this mode selects the rich converter.
//
// In V1 only an explicit `on` does. `auto` is reserved for a future predicate
// built on the rich converter's typed loss set; it must not read the classic
// converter's warnings, which mix conversion losses with length and policy
// warnings and would flip a note's format for unrelated reasons.
func (m TelegramRichMode) UseRich() bool {
	return m == TelegramRichOn
}

// ExtractTelegramRichMode reads `telegram_rich` from the frontmatter. Missing
// means `auto`. YAML booleans are accepted alongside the three spellings
// because frontmatter lands in an untyped RawMeta and `telegram_rich: true`
// arrives as a bool. An unrecognised value returns `auto` plus an error rather
// than degrading to `off`, so a typo is visible instead of silently changing
// the published format.
func (note *NoteView) ExtractTelegramRichMode() (TelegramRichMode, error) {
	raw, ok := note.RawMeta["telegram_rich"]
	if !ok {
		return TelegramRichAuto, nil
	}

	switch value := raw.(type) {
	case bool:
		if value {
			return TelegramRichOn, nil
		}
		return TelegramRichOff, nil

	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "auto":
			return TelegramRichAuto, nil
		case "on", boolTrueStr:
			return TelegramRichOn, nil
		case "off", "false":
			return TelegramRichOff, nil
		}

		return TelegramRichAuto, fmt.Errorf(
			"invalid telegram_rich value %q, expected auto, on or off", value)
	}

	return TelegramRichAuto, fmt.Errorf(
		"invalid telegram_rich format %T, expected auto, on or off", raw)
}
