package handletgcanvasupdate

import (
	"context"
	"encoding/json"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

// canvasState holds the in-memory navigation state for a user.
type canvasState struct {
	BotID      int64
	BCID       string
	UserID     int64
	CanvasPath string
	Current    string
	Stack      []string
	LastMedia  string
	MessageID  int64
}

func serializeStack(stack []string) string {
	if len(stack) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(stack)
	return string(b)
}

func deserializeStack(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var stack []string
	if err := json.Unmarshal([]byte(raw), &stack); err != nil {
		return nil
	}
	return stack
}

func saveState(ctx context.Context, env Env, log logger.Logger, s canvasState) {
	err := env.UpsertTgUserCanvasState(ctx, db.UpsertTgUserCanvasStateParams{
		BotID:                s.BotID,
		BusinessConnectionID: s.BCID,
		UserID:               s.UserID,
		CanvasPath:           s.CanvasPath,
		CurrentNode:          s.Current,
		Stack:                serializeStack(s.Stack),
		LastMedia:            s.LastMedia,
		MessageID:            s.MessageID,
	})
	if err != nil {
		log.Error("canvas: save state failed", "error", err)
	}
}
