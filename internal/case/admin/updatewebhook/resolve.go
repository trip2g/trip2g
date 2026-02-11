package updatewebhook

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UpdateWebhook(ctx context.Context, params db.UpdateWebhookParams) (db.ChangeWebhook, error)
}

type Input = model.UpdateWebhookInput
type Payload = model.UpdateWebhookOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Validate bounds for optional fields.
	var fieldErrs []model.FieldMessage
	if input.MaxDepth != nil && (*input.MaxDepth < 0 || *input.MaxDepth > 999) {
		fieldErrs = append(fieldErrs, model.FieldMessage{Name: "maxDepth", Value: "must be between 0 and 999"})
	}
	if input.TimeoutSeconds != nil && (*input.TimeoutSeconds < 1 || *input.TimeoutSeconds > 3600) {
		fieldErrs = append(fieldErrs, model.FieldMessage{Name: "timeoutSeconds", Value: "must be between 1 and 3600"})
	}
	if input.MaxRetries != nil && (*input.MaxRetries < 0 || *input.MaxRetries > 100) {
		fieldErrs = append(fieldErrs, model.FieldMessage{Name: "maxRetries", Value: "must be between 0 and 100"})
	}
	if len(fieldErrs) > 0 {
		return &model.ErrorPayload{ByFields: fieldErrs}, nil
	}

	params := db.UpdateWebhookParams{
		ID:             input.ID,
		Url:            input.URL,
		Instruction:    input.Instruction,
		MaxDepth:       input.MaxDepth,
		PassApiKey:     input.PassAPIKey,
		IncludeContent: input.IncludeContent,
		TimeoutSeconds: input.TimeoutSeconds,
		MaxRetries:     input.MaxRetries,
		Enabled:        input.Enabled,
		Description:    input.Description,
		OnCreate:       input.OnCreate,
		OnUpdate:       input.OnUpdate,
		OnRemove:       input.OnRemove,
	}

	// Marshal JSON arrays only if provided.
	if input.IncludePatterns != nil {
		j, jsonErr := json.Marshal(input.IncludePatterns)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal include_patterns: %w", jsonErr)
		}
		params.IncludePatterns = ptr.To(string(j))
	}
	if input.ExcludePatterns != nil {
		j, jsonErr := json.Marshal(input.ExcludePatterns)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal exclude_patterns: %w", jsonErr)
		}
		params.ExcludePatterns = ptr.To(string(j))
	}
	if input.ReadPatterns != nil {
		j, jsonErr := json.Marshal(input.ReadPatterns)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal read_patterns: %w", jsonErr)
		}
		params.ReadPatterns = ptr.To(string(j))
	}
	if input.WritePatterns != nil {
		j, jsonErr := json.Marshal(input.WritePatterns)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal write_patterns: %w", jsonErr)
		}
		params.WritePatterns = ptr.To(string(j))
	}

	webhook, err := env.UpdateWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return &model.UpdateWebhookPayload{
		Webhook: &webhook,
	}, nil
}
