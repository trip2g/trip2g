package toggleuserfavoritenote

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg toggleuserfavoritenote_test . Env

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	InsertUserFavoriteNote(ctx context.Context, arg db.InsertUserFavoriteNoteParams) error
	DeleteUserFavoriteNote(ctx context.Context, arg db.DeleteUserFavoriteNoteParams) error
	LiveNoteViews() *appmodel.NoteViews
}

type Input = model.ToggleFavoriteNoteInput
type Payload = model.ToggleFavoriteNoteOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.PathID, validation.Required, validation.Min(int64(1))),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Get current user
	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	if userToken == nil {
		return &model.ErrorPayload{Message: "no auth"}, nil
	}

	userID := int64(userToken.ID)

	note := env.LiveNoteViews().GetByPathID(input.PathID)
	if note == nil {
		return &model.ErrorPayload{Message: "note not found"}, nil
	}

	if !input.Value {
		// Remove from favorites
		deleteParams := db.DeleteUserFavoriteNoteParams{
			UserID:        userID,
			NoteVersionID: note.VersionID,
		}
		err = env.DeleteUserFavoriteNote(ctx, deleteParams)
		if err != nil {
			return nil, fmt.Errorf("failed to delete favorite: %w", err)
		}
	} else {
		// Add to favorites
		insertParams := db.InsertUserFavoriteNoteParams{
			UserID:        userID,
			NoteVersionID: note.VersionID,
		}
		err = env.InsertUserFavoriteNote(ctx, insertParams)
		if err != nil {
			return nil, fmt.Errorf("failed to insert favorite: %w", err)
		}
	}

	payload := model.ToggleFavoriteNotePayload{
		Success: true,
		UserID:  userID,
	}

	return &payload, nil
}
