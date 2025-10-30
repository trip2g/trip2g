package createapikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	GenerateAPIKey() string
	InsertAPIKey(ctx context.Context, params db.InsertAPIKeyParams) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.CreateAPIKeyInput
type Payload = model.CreateAPIKeyOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	apiKey := env.GenerateAPIKey()

	// Hash the API key before storing (like git tokens)
	hash := sha256.Sum256([]byte(apiKey))
	hashedValue := hex.EncodeToString(hash[:])

	params := db.InsertAPIKeyParams{
		Value:       hashedValue,
		CreatedBy:   int64(token.ID),
		Description: input.Description,
	}

	createdKey, err := env.InsertAPIKey(ctx, params)
	if err != nil {
		return nil, err
	}

	response := model.CreateAPIKeyPayload{
		Value:  apiKey, // Return the plain text key to user (only time they see it)
		APIKey: &createdKey,
	}

	return &response, nil
}
