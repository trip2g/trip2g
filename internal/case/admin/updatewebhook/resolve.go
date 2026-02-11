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

// validateBounds checks optional numeric fields are within allowed ranges.
func validateBounds(input Input) *model.ErrorPayload {
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
		return &model.ErrorPayload{ByFields: fieldErrs}
	}
	return nil
}

// marshalOptionalJSON marshals a string slice to JSON if non-nil.
func marshalOptionalJSON(patterns []string) (*string, error) {
	if patterns == nil {
		return nil, nil
	}
	j, err := json.Marshal(patterns)
	if err != nil {
		return nil, err
	}
	return ptr.To(string(j)), nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	boundsErr := validateBounds(input)
	if boundsErr != nil {
		return boundsErr, nil
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
	params.IncludePatterns, err = marshalOptionalJSON(input.IncludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal include_patterns: %w", err)
	}
	params.ExcludePatterns, err = marshalOptionalJSON(input.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal exclude_patterns: %w", err)
	}
	params.ReadPatterns, err = marshalOptionalJSON(input.ReadPatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal read_patterns: %w", err)
	}
	params.WritePatterns, err = marshalOptionalJSON(input.WritePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal write_patterns: %w", err)
	}

	webhook, err := env.UpdateWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return &model.UpdateWebhookPayload{
		Webhook: &webhook,
	}, nil
}
