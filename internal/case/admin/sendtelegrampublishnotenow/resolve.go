package sendtelegrampublishnotenow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampublishnotenow_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	SendTelegramPublishPost(ctx context.Context, notePathID int64, instant bool) error
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

	err = env.SendTelegramPublishPost(ctx, input.ID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to send telegram publish post: %w", err)
	}

	payload := model.SendTelegramPublishNoteNowPayload{
		PublishNote: &publishNote,
	}

	return &payload, nil
}
