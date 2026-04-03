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

	InvalidateSiteConfig()
}

type Input = model.SetConfigStringValueInput
type Payload = model.SetConfigStringValuePayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	meta, ok := configregistry.GetString(input.ID)
	if !ok {
		return &model.ErrorPayload{Message: fmt.Sprintf("unknown string config: %s", input.ID)}, nil
	}

	if meta.Validate != nil {
		if err := meta.Validate(input.Value); err != nil {
			//nolint:nilerr // validation error returned as payload, not as error.
			return &model.ErrorPayload{Message: err.Error()}, nil
		}
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	result, err := insertConfigString(ctx, env, token, input.ID, input.Value)
	if err == nil {
		env.InvalidateSiteConfig()
	}

	return result, err
}

func insertConfigString(ctx context.Context, env Env, token *usertoken.Data, valueID, value string) (Payload, error) {
	changeParams := db.InsertConfigChangeParams{
		ValueID:   valueID,
		CreatedBy: int64(token.ID),
	}

	change, err := env.InsertConfigChange(ctx, changeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config change: %w", err)
	}

	valueParams := db.InsertConfigStringValueParams{
		ChangeID: change.ID,
		Value:    value,
	}

	err = env.InsertConfigStringValue(ctx, valueParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config string value: %w", err)
	}

	entry, err := env.GetLatestConfigString(ctx, valueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest config string: %w", err)
	}

	meta, ok := configregistry.GetString(valueID)
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
