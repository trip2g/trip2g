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
	InsertConfigShowDraftVersions(ctx context.Context, arg db.InsertConfigShowDraftVersionsParams) (db.ConfigShowDraftVersion, error)
	GetLatestConfigShowDraftVersions(ctx context.Context) (db.ConfigShowDraftVersion, error)
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

	switch input.ID {
	case configregistry.ConfigShowDraftVersions:
		return insertShowDraftVersions(ctx, env, token, input.Value)
	default:
		return &model.ErrorPayload{Message: fmt.Sprintf("config %s is not yet implemented", input.ID)}, nil
	}
}

func insertShowDraftVersions(ctx context.Context, env Env, token *usertoken.Data, value bool) (Payload, error) {
	params := db.InsertConfigShowDraftVersionsParams{
		CreatedBy: int64(token.ID),
		Value:     value,
	}

	entry, err := env.InsertConfigShowDraftVersions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config show draft versions: %w", err)
	}

	meta := configregistry.Registry[configregistry.ConfigShowDraftVersions]

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
