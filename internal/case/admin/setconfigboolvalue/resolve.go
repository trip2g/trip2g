//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg setconfigboolvalue_test . Env

package setconfigboolvalue

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
	InsertConfigBoolValue(ctx context.Context, arg db.InsertConfigBoolValueParams) error
	GetLatestConfigBool(ctx context.Context, valueID string) (db.GetLatestConfigBoolRow, error)
	ListConfigBoolHistory(ctx context.Context, valueID string) ([]db.ListConfigBoolHistoryRow, error)

	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
}

type Input = model.SetConfigBoolValueInput
type Payload = model.SetConfigBoolValuePayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	meta, ok := configregistry.Get(input.ID)
	if !ok {
		return &model.ErrorPayload{Message: fmt.Sprintf("unknown config: %s", input.ID)}, nil
	}

	if meta.Type != configregistry.ConfigTypeBool {
		return &model.ErrorPayload{Message: fmt.Sprintf("config %s is not a bool config", input.ID)}, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	return insertConfigBool(ctx, env, token, input.ID, input.Value)
}

func insertConfigBool(ctx context.Context, env Env, token *usertoken.Data, valueID string, value bool) (Payload, error) {
	// 1. Insert config change to get the change ID.
	changeParams := db.InsertConfigChangeParams{
		ValueID:   valueID,
		CreatedBy: int64(token.ID),
	}

	change, err := env.InsertConfigChange(ctx, changeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config change: %w", err)
	}

	// 2. Insert bool value with the change ID.
	valueParams := db.InsertConfigBoolValueParams{
		ChangeID: change.ID,
		Value:    value,
	}

	err = env.InsertConfigBoolValue(ctx, valueParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config bool value: %w", err)
	}

	// 3. Build the response using the new unified query.
	entry, err := env.GetLatestConfigBool(ctx, valueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest config bool: %w", err)
	}

	meta, ok := configregistry.Get(valueID)
	if !ok {
		return nil, fmt.Errorf("config metadata not found: %s", valueID)
	}

	user, err := env.UserByID(ctx, entry.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	configValue := &model.AdminConfigBoolValue{
		ID:          meta.ID,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       entry.Value,
	}
	_ = user // Will be resolved by field resolver.

	return &model.SetConfigBoolValueSuccess{
		ConfigValue: configValue,
	}, nil
}
