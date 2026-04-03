package graph

import (
	"context"
	"fmt"
	"trip2g/internal/configregistry"
	"trip2g/internal/graph/model"
)

func (r *adminQueryResolver) buildConfigValue(ctx context.Context, id string) (model.AdminConfigValue, error) {
	if meta, ok := configregistry.GetString(id); ok {
		return r.buildStringConfigValue(ctx, id, meta), nil
	}

	if meta, ok := configregistry.GetBool(id); ok {
		return r.buildBoolConfigValue(ctx, id, meta), nil
	}

	if meta, ok := configregistry.GetInt(id); ok {
		return r.buildIntConfigValue(ctx, id, meta), nil
	}

	return nil, fmt.Errorf("unknown config: %s", id)
}

func (r *adminQueryResolver) buildStringConfigValue(ctx context.Context, id string, meta configregistry.StringConfigMeta) *model.AdminConfigStringValue {
	entry, err := r.env(ctx).GetLatestConfigString(ctx, id)
	if err != nil {
		return &model.AdminConfigStringValue{
			ID:          id,
			Description: &meta.Description,
			Value:       meta.Default,
		}
	}

	return &model.AdminConfigStringValue{
		ID:          id,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       entry.Value,
	}
}

func (r *adminQueryResolver) buildBoolConfigValue(ctx context.Context, id string, meta configregistry.BoolConfigMeta) *model.AdminConfigBoolValue {
	entry, err := r.env(ctx).GetLatestConfigBool(ctx, id)
	if err != nil {
		return &model.AdminConfigBoolValue{
			ID:          id,
			Description: &meta.Description,
			Value:       meta.Default,
		}
	}

	return &model.AdminConfigBoolValue{
		ID:          id,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       entry.Value,
	}
}

func (r *adminQueryResolver) buildIntConfigValue(ctx context.Context, id string, meta configregistry.IntConfigMeta) *model.AdminConfigIntValue {
	entry, err := r.env(ctx).GetLatestConfigInt(ctx, id)
	if err != nil {
		return &model.AdminConfigIntValue{
			ID:          id,
			Description: &meta.Description,
			Value:       int32(meta.Default),
		}
	}

	return &model.AdminConfigIntValue{
		ID:          id,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       int32(entry.Value),
	}
}
