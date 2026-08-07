package tgrich

// Limits are the server-side rich-message limits.
//
// Telegram advertises every one of them through help.getAppConfig, so they are
// read, not hardcoded. trip2g already stores that response per account in
// telegram_accounts.app_config and already resolves a limit out of it — see
// (*app).TelegramCaptionLengthLimit in cmd/server/telegram.go, which unmarshals
// account.AppConfig into a map[string]interface{} and falls back to a documented
// default on any error. A future app-layer method follows the same shape and
// feeds LimitsFromAppConfig; DefaultLimits plays the role of that fallback.
type Limits struct {
	// TextLength is the total visible text budget in UTF-16 code units.
	TextLength int
	// MaxBlocks counts top-level blocks only; list items do not count.
	MaxBlocks int
	// MaxMedia counts photo and video blocks anywhere in the tree.
	MaxMedia int
	// MaxDepth is the maximum block nesting depth.
	MaxDepth int
	// MaxTableCols is the maximum number of table columns.
	MaxTableCols int
}

// App config keys advertising the limits above.
const (
	KeyTextLength   = "rich_message_length_limit"
	KeyMaxBlocks    = "rich_message_max_blocks"
	KeyMaxMedia     = "rich_message_max_media"
	KeyMaxDepth     = "rich_message_max_depth"
	KeyMaxTableCols = "rich_message_max_table_cols"
)

// DefaultLimits returns the values help.getAppConfig advertised in July 2026.
// The first three were confirmed by probing (N passes, N+1 fails); the last two
// were never probed, which is exactly why the server is the better source.
func DefaultLimits() Limits {
	return Limits{
		TextLength:   32768,
		MaxBlocks:    500,
		MaxMedia:     50,
		MaxDepth:     16,
		MaxTableCols: 20,
	}
}

// LimitsFromAppConfig overlays an unmarshalled help.getAppConfig document onto
// DefaultLimits. Missing or non-numeric entries keep their default, so a config
// that predates rich messages degrades to the documented values rather than to
// zero.
func LimitsFromAppConfig(config map[string]interface{}) Limits {
	limits := DefaultLimits()

	for key, target := range map[string]*int{
		KeyTextLength:   &limits.TextLength,
		KeyMaxBlocks:    &limits.MaxBlocks,
		KeyMaxMedia:     &limits.MaxMedia,
		KeyMaxDepth:     &limits.MaxDepth,
		KeyMaxTableCols: &limits.MaxTableCols,
	} {
		// JSON numbers decode into float64 through map[string]interface{}.
		if value, ok := config[key].(float64); ok && value > 0 {
			*target = int(value)
		}
	}

	return limits
}
