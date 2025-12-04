package sendtelegrampublishnotenow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// isNoChatsError checks if the error is "no chats found" error
func isNoChatsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no chat IDs found") ||
		strings.Contains(err.Error(), "no account chats found")
}

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampublishnotenow_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	SendTelegramPublishPost(ctx context.Context, params appmodel.SendTelegramPublishPostParams) error
	SendTelegramAccountPublishPost(ctx context.Context, params appmodel.SendTelegramPublishPostParams) error
	GetTelegramPublishNoteByNotePathID(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error)
}

type Input = model.SendTelegramPublishNoteNowInput
type Payload = model.SendTelegramPublishNoteNowOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return nil // No validation needed for simple int64 ID
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Check if the publish note exists
	publishNote, err := env.GetTelegramPublishNoteByNotePathID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Telegram publish note not found"}, nil
		}
		return nil, fmt.Errorf("failed to get telegram publish note: %w", err)
	}

	params := appmodel.SendTelegramPublishPostParams{
		NotePathID:        input.ID,
		Instant:           false,
		UpdateLinkedPosts: true,
	}

	// Send via bot (ignore "no chats" error - account might have chats)
	botErr := env.SendTelegramPublishPost(ctx, params)
	if botErr != nil && !isNoChatsError(botErr) {
		return nil, fmt.Errorf("failed to send telegram publish post via bot: %w", botErr)
	}

	// Send via account (ignore "no chats" error - bot might have chats)
	accountErr := env.SendTelegramAccountPublishPost(ctx, params)
	if accountErr != nil && !isNoChatsError(accountErr) {
		return nil, fmt.Errorf("failed to send telegram publish post via account: %w", accountErr)
	}

	// If both have no chats, return error
	if isNoChatsError(botErr) && isNoChatsError(accountErr) {
		return &model.ErrorPayload{Message: "No chats configured for this note"}, nil
	}

	payload := model.SendTelegramPublishNoteNowPayload{
		PublishNote: &publishNote,
	}

	return &payload, nil
}
