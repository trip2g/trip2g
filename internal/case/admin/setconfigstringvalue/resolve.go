//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg setconfigstringvalue_test . Env

package setconfigstringvalue

import (
	"context"
	"fmt"

	"trip2g/internal/configregistry"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertConfigChange(ctx context.Context, arg db.InsertConfigChangeParams) (db.ConfigChange, error)
	InsertConfigStringValue(ctx context.Context, arg db.InsertConfigStringValueParams) error
	GetLatestConfigString(ctx context.Context, valueID string) (db.GetLatestConfigStringRow, error)
	ListConfigStringHistory(ctx context.Context, valueID string) ([]db.ListConfigStringHistoryRow, error)

	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
}

type Input = model.SetConfigStringValueInput
type Payload = model.SetConfigStringValuePayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	meta, ok := configregistry.Get(input.ID)
	if !ok {
		return &model.ErrorPayload{Message: fmt.Sprintf("unknown config: %s", input.ID)}, nil
	}

	if meta.Type != configregistry.ConfigTypeString {
		return &model.ErrorPayload{Message: fmt.Sprintf("config %s is not a string config", input.ID)}, nil
	}

	// Validate value.
	if meta.Validate != nil {
		validationErr := meta.Validate(input.Value)
		if validationErr != nil {
			//nolint:nilerr // validation error returned as payload, not as error.
			return &model.ErrorPayload{Message: validationErr.Error()}, nil
		}
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	return insertConfigString(ctx, env, token, input.ID, input.Value)
}

func insertConfigString(ctx context.Context, env Env, token *usertoken.Data, valueID, value string) (Payload, error) {
	// 1. Insert config change to get the change ID.
	changeParams := db.InsertConfigChangeParams{
		ValueID:   valueID,
		CreatedBy: int64(token.ID),
	}

	change, err := env.InsertConfigChange(ctx, changeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config change: %w", err)
	}

	// 2. Insert string value with the change ID.
	valueParams := db.InsertConfigStringValueParams{
		ChangeID: change.ID,
		Value:    value,
	}

	err = env.InsertConfigStringValue(ctx, valueParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config string value: %w", err)
	}

	// 3. Build the response using the new unified query.
	entry, err := env.GetLatestConfigString(ctx, valueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest config string: %w", err)
	}

	meta, ok := configregistry.Get(valueID)
	if !ok {
		return nil, fmt.Errorf("config metadata not found: %s", valueID)
	}

	user, err := env.UserByID(ctx, entry.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	configValue := &model.AdminConfigStringValue{
		ID:          meta.ID,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       entry.Value,
	}
	_ = user // Will be resolved by field resolver.

	return &model.SetConfigStringValueSuccess{
		ConfigValue: configValue,
	}, nil
}
