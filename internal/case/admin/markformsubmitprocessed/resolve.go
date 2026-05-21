package markformsubmitprocessed

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg markformsubmitprocessed_test . Env

type Env interface {
	MarkFormSubmitProcessed(ctx context.Context, params db.MarkFormSubmitProcessedParams) (db.FormSubmit, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.MarkFormSubmitProcessedInput
type Payload = model.MarkFormSubmitProcessedOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current user token: %w", err)
	}

	var comment string
	if input.Comment != nil {
		comment = *input.Comment
	}

	submit, err := env.MarkFormSubmitProcessed(ctx, db.MarkFormSubmitProcessedParams{
		ID:          input.SubmitID,
		ProcessedBy: ptr.To(int64(token.ID)),
		Comment:     comment,
	})
	if err != nil {
		return nil, fmt.Errorf("mark form submit processed: %w", err)
	}

	return &model.MarkFormSubmitProcessedPayload{Submit: &submit}, nil
}
