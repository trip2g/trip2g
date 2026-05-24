package handletgcanvasupdate

import "context"

// renderTransition handles the transition between messages when navigating.
// If previous message had media and new one doesn't (or vice versa), we must
// delete the old message and send a new one (Telegram can't edit photo→text).
// Returns the new message ID.
func renderTransition(ctx context.Context, env Env, chatID int64, bcID string, oldMsgID int, oldMedia, newMedia, text, markup string) int {
	hadMedia := oldMedia != ""
	hasMedia := newMedia != ""

	// Same type: can edit in place
	if oldMsgID != 0 && hadMedia == hasMedia && !hasMedia {
		// text→text: edit text
		err := env.EditMessageText(ctx, chatID, oldMsgID, bcID, text, markup)
		if err == nil {
			return oldMsgID
		}
		// Fall through to delete+send on error
	}

	// Different type or photo→photo (Telegram doesn't support editMessageMedia
	// with URL-based photos reliably): delete old and send new
	if oldMsgID != 0 {
		_ = env.DeleteMessage(ctx, chatID, oldMsgID, bcID)
	}

	var newMsgID int
	if hasMedia {
		newMsgID, _ = env.SendPhoto(ctx, chatID, bcID, newMedia, text, markup)
	} else {
		newMsgID, _ = env.SendMessage(ctx, chatID, bcID, text, markup)
	}
	return newMsgID
}
