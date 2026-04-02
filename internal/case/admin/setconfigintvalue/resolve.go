package setconfigintvalue

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
	InsertConfigIntValue(ctx context.Context, arg db.InsertConfigIntValueParams) error
	GetLatestConfigInt(ctx context.Context, valueID string) (db.GetLatestConfigIntRow, error)

	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
}

type Input = model.SetConfigIntValueInput
type Payload = model.SetConfigIntValuePayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	meta, ok := configregistry.Get(input.ID)
	if !ok {
		return &model.ErrorPayload{Message: fmt.Sprintf("unknown config: %s", input.ID)}, nil
	}

	if meta.Type != configregistry.ConfigTypeInt {
		return &model.ErrorPayload{Message: fmt.Sprintf("config %s is not an int config", input.ID)}, nil
	}

	if meta.Validate != nil {
		if err := meta.Validate(int(input.Value)); err != nil {
			return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr // business error returned as payload
		}
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	changeParams := db.InsertConfigChangeParams{
		ValueID:   input.ID,
		CreatedBy: int64(token.ID),
	}

	change, err := env.InsertConfigChange(ctx, changeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config change: %w", err)
	}

	valueParams := db.InsertConfigIntValueParams{
		ChangeID: change.ID,
		Value:    int64(input.Value),
	}

	err = env.InsertConfigIntValue(ctx, valueParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config int value: %w", err)
	}

	entry, err := env.GetLatestConfigInt(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest config int: %w", err)
	}

	configValue := &model.AdminConfigIntValue{
		ID:          meta.ID,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       int32(entry.Value),
	}

	return &model.SetConfigIntValueSuccess{
		ConfigValue: configValue,
	}, nil
}
