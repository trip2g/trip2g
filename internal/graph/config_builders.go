package graph

import (
	"context"
	"fmt"
	"trip2g/internal/configregistry"
	"trip2g/internal/graph/model"
)

func (r *adminQueryResolver) buildConfigValue(ctx context.Context, id string) (model.AdminConfigValue, error) {
	meta, ok := configregistry.Get(id)
	if !ok {
		return nil, fmt.Errorf("unknown config: %s", id)
	}

	switch meta.Type {
	case configregistry.ConfigTypeString:
		return r.buildStringConfigValue(ctx, id, meta), nil
	case configregistry.ConfigTypeBool:
		return r.buildBoolConfigValue(ctx, id, meta), nil
	default:
		return nil, fmt.Errorf("unknown config type: %s", meta.Type)
	}
}

func (r *adminQueryResolver) buildStringConfigValue(ctx context.Context, id string, meta configregistry.ConfigMeta) *model.AdminConfigStringValue {
	defaultValue, _ := meta.Default.(string)

	entry, err := r.env(ctx).GetLatestConfigString(ctx, id)
	if err != nil {
		return &model.AdminConfigStringValue{
			ID:          id,
			Description: &meta.Description,
			Value:       defaultValue,
		}
	}

	return &model.AdminConfigStringValue{
		ID:          id,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       entry.Value,
	}
}

func (r *adminQueryResolver) buildBoolConfigValue(ctx context.Context, id string, meta configregistry.ConfigMeta) *model.AdminConfigBoolValue {
	defaultValue, _ := meta.Default.(bool)

	entry, err := r.env(ctx).GetLatestConfigBool(ctx, id)
	if err != nil {
		return &model.AdminConfigBoolValue{
			ID:          id,
			Description: &meta.Description,
			Value:       defaultValue,
		}
	}

	return &model.AdminConfigBoolValue{
		ID:          id,
		Description: &meta.Description,
		UpdatedAt:   &entry.CreatedAt,
		Value:       entry.Value,
	}
}
