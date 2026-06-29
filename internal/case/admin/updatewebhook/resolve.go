package updatewebhook

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/jsonneteval"
	appmodel "trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	IsDevMode() bool
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UpdateWebhook(ctx context.Context, params db.UpdateWebhookParams) (db.ChangeWebhook, error)
	WebhookByID(ctx context.Context, id int64) (db.ChangeWebhook, error)
	GetSecretValues(ctx context.Context, like string) (map[string]string, error)
}

type Input = model.ChangeWebhookUpdateInput
type Payload = model.ChangeWebhookUpdateOrErrorPayload

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

// validateTransformJsonnet rejects a transform that cannot even evaluate.
func validateTransformJsonnet(src *string) *model.ErrorPayload {
	if src == nil || *src == "" {
		return nil
	}
	if err := jsonneteval.Validate(*src, map[string]string{
		"change":         "[]",
		"attached_notes": "[]",
		"meta":           "{}",
	}); err != nil {
		return &model.ErrorPayload{ByFields: []model.FieldMessage{
			{Name: "transformJsonnet", Value: "invalid jsonnet: " + err.Error()},
		}}
	}
	return nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Load existing webhook to compute effective state for cross-field guards.
	// Update inputs are partial (patch semantics): a nil field means "keep existing".
	existing, err := env.WebhookByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load webhook %d: %w", input.ID, err)
	}

	boundsErr := validateBounds(input)
	if boundsErr != nil {
		return boundsErr, nil
	}

	if ep := validateTransformJsonnet(input.TransformJsonnet); ep != nil {
		return ep, nil
	}

	// Compute effective state: merge input (patches) onto existing persisted values.
	effectiveURL := existing.Url
	if input.URL != nil {
		effectiveURL = *input.URL
	}
	effectivePassAPIKey := existing.PassApiKey
	if input.PassAPIKey != nil {
		effectivePassAPIKey = *input.PassAPIKey
	}
	effectiveTransform := existing.TransformJsonnet
	if input.TransformJsonnet != nil {
		effectiveTransform = *input.TransformJsonnet
	}

	// Check for attached secrets (exist independently of pass_api_key).
	prefix, _ := appmodel.ChangeWebhookSecretPrefix(input.ID)
	secretValues, _ := env.GetSecretValues(ctx, prefix.Like())
	hasSecrets := len(secretValues) > 0

	// F9(a): require HTTPS whenever sensitive data (api_token or decrypted secrets)
	// would travel in the delivery body. Use effective merged state so partial updates
	// (URL-only or pass_api_key-only) are caught even when only one field is provided.
	// Dev mode exempts Docker-internal URLs (e.g. compose service names).
	if effectivePassAPIKey || hasSecrets {
		if msg := webhookutil.RequireHTTPSUnlessDevMode(effectiveURL, env.IsDevMode()); msg != "" {
			return &model.ErrorPayload{ByFields: []model.FieldMessage{{Name: "url", Value: msg}}}, nil
		}
	}

	// F9(b): transform_jsonnet output replaces the entire body, silently dropping
	// api_token and secrets. Reject the combination on the effective merged state so
	// enabling transform alone on a webhook that already has pass_api_key or secrets
	// is caught even when only transform_jsonnet is provided in this request.
	if effectiveTransform != "" && (effectivePassAPIKey || hasSecrets) {
		return &model.ErrorPayload{ByFields: []model.FieldMessage{
			{Name: "transformJsonnet", Value: "transform_jsonnet cannot be combined with pass_api_key or attached secrets"},
		}}, nil
	}

	if input.ConcurrencyMode != nil {
		if modeErr := webhookutil.ValidateConcurrencyMode(*input.ConcurrencyMode); modeErr != nil {
			//nolint:nilerr // returning user-facing validation error
			return &model.ErrorPayload{ByFields: []model.FieldMessage{
				{Name: "concurrencyMode", Value: modeErr.Error()},
			}}, nil
		}
	}

	params := db.UpdateWebhookParams{
		ID:               input.ID,
		Url:              input.URL,
		Instruction:      input.Instruction,
		MaxDepth:         input.MaxDepth,
		PassApiKey:       input.PassAPIKey,
		IncludeContent:   input.IncludeContent,
		TimeoutSeconds:   input.TimeoutSeconds,
		MaxRetries:       input.MaxRetries,
		Enabled:          input.Enabled,
		Description:      input.Description,
		OnCreate:         input.OnCreate,
		OnUpdate:         input.OnUpdate,
		OnRemove:         input.OnRemove,
		TransformJsonnet: input.TransformJsonnet,
		ConcurrencyMode:  input.ConcurrencyMode,
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
	params.AttachNotes, err = marshalOptionalJSON(input.AttachNotes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attach_notes: %w", err)
	}

	webhook, err := env.UpdateWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return &model.ChangeWebhookUpdatePayload{
		Webhook: &webhook,
	}, nil
}
