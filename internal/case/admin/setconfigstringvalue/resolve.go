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
	InsertConfigSiteTitleTemplate(ctx context.Context, arg db.InsertConfigSiteTitleTemplateParams) (db.ConfigSiteTitleTemplate, error)
	GetLatestConfigSiteTitleTemplate(ctx context.Context) (db.ConfigSiteTitleTemplate, error)
	ListConfigSiteTitleTemplateHistory(ctx context.Context) ([]db.ConfigSiteTitleTemplate, error)

	InsertConfigTimezone(ctx context.Context, arg db.InsertConfigTimezoneParams) (db.ConfigTimezone, error)
	GetLatestConfigTimezone(ctx context.Context) (db.ConfigTimezone, error)

	InsertConfigDefaultLayout(ctx context.Context, arg db.InsertConfigDefaultLayoutParams) (db.ConfigDefaultLayout, error)
	GetLatestConfigDefaultLayout(ctx context.Context) (db.ConfigDefaultLayout, error)

	InsertConfigRobotsTxt(ctx context.Context, arg db.InsertConfigRobotsTxtParams) (db.ConfigRobotsTxt, error)
	GetLatestConfigRobotsTxt(ctx context.Context) (db.ConfigRobotsTxt, error)

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

	switch input.ID {
	case configregistry.ConfigSiteTitleTemplate:
		return insertSiteTitleTemplate(ctx, env, token, input.Value)
	case configregistry.ConfigTimezone:
		return insertTimezone(ctx, env, token, input.Value)
	case configregistry.ConfigDefaultLayout:
		return insertDefaultLayout(ctx, env, token, input.Value)
	case configregistry.ConfigRobotsTxt:
		return insertRobotsTxt(ctx, env, token, input.Value)
	default:
		return &model.ErrorPayload{Message: fmt.Sprintf("config %s is not yet implemented", input.ID)}, nil
	}
}

func insertSiteTitleTemplate(ctx context.Context, env Env, token *usertoken.Data, value string) (Payload, error) {
	params := db.InsertConfigSiteTitleTemplateParams{
		CreatedBy: int64(token.ID),
		Value:     value,
	}

	entry, err := env.InsertConfigSiteTitleTemplate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config site title template: %w", err)
	}

	// Build the config value response.
	meta := configregistry.Registry[configregistry.ConfigSiteTitleTemplate]

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
	// Note: UpdatedBy and History are resolved by field resolvers.
	_ = user // Will be resolved by field resolver.

	return &model.SetConfigStringValueSuccess{
		ConfigValue: configValue,
	}, nil
}

func insertTimezone(ctx context.Context, env Env, token *usertoken.Data, value string) (Payload, error) {
	params := db.InsertConfigTimezoneParams{
		CreatedBy: int64(token.ID),
		Value:     value,
	}

	entry, err := env.InsertConfigTimezone(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config timezone: %w", err)
	}

	meta := configregistry.Registry[configregistry.ConfigTimezone]

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

func insertDefaultLayout(ctx context.Context, env Env, token *usertoken.Data, value string) (Payload, error) {
	params := db.InsertConfigDefaultLayoutParams{
		CreatedBy: int64(token.ID),
		Value:     value,
	}

	entry, err := env.InsertConfigDefaultLayout(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config default layout: %w", err)
	}

	meta := configregistry.Registry[configregistry.ConfigDefaultLayout]

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

func insertRobotsTxt(ctx context.Context, env Env, token *usertoken.Data, value string) (Payload, error) {
	params := db.InsertConfigRobotsTxtParams{
		CreatedBy: int64(token.ID),
		Value:     value,
	}

	entry, err := env.InsertConfigRobotsTxt(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config robots txt: %w", err)
	}

	meta := configregistry.Registry[configregistry.ConfigRobotsTxt]

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
