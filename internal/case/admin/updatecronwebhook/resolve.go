package updatecronwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/jsonneteval"
	appmodel "trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UpdateCronWebhook(ctx context.Context, params db.UpdateCronWebhookParams) (db.CronWebhook, error)
	UpdateCronWebhookNextRunAt(ctx context.Context, params db.UpdateCronWebhookNextRunAtParams) error
	CronWebhookByID(ctx context.Context, id int64) (db.CronWebhook, error)
	GetSecretValues(ctx context.Context, like string) (map[string]string, error)
}

type Input = model.UpdateCronWebhookInput
type Payload = model.UpdateCronWebhookOrErrorPayload

// validateCronSchedule validates a cron expression if provided.
func validateCronSchedule(schedule *string) *model.ErrorPayload {
	if schedule == nil {
		return nil
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(*schedule)
	if err != nil {
		return &model.ErrorPayload{
			ByFields: []model.FieldMessage{
				{Name: "cronSchedule", Value: "invalid cron expression: " + err.Error()},
			},
		}
	}
	return nil
}

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

func validateConcurrencyMode(mode string) *model.ErrorPayload {
	switch mode {
	case "allow_overlap", "skip", "queue_one":
		return nil
	default:
		return &model.ErrorPayload{ByFields: []model.FieldMessage{
			{Name: "concurrencyMode", Value: "must be one of allow_overlap, skip, queue_one"},
		}}
	}
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Load existing cron webhook to compute effective state for cross-field guards.
	// Update inputs are partial (patch semantics): a nil field means "keep existing".
	existing, err := env.CronWebhookByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cron webhook %d: %w", input.ID, err)
	}

	scheduleErr := validateCronSchedule(input.CronSchedule)
	if scheduleErr != nil {
		return scheduleErr, nil
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
	prefix, _ := appmodel.CronWebhookSecretPrefix(input.ID)
	secretValues, _ := env.GetSecretValues(ctx, prefix.Like())
	hasSecrets := len(secretValues) > 0

	// F9(a): require HTTPS whenever sensitive data (api_token or decrypted secrets)
	// would travel in the delivery body. Use effective merged state so partial updates
	// (URL-only or pass_api_key-only) are caught even when only one field is provided.
	if effectivePassAPIKey || hasSecrets {
		if msg := webhookutil.RequireHTTPS(effectiveURL); msg != "" {
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
		if cErr := validateConcurrencyMode(*input.ConcurrencyMode); cErr != nil {
			return cErr, nil
		}
	}

	params := db.UpdateCronWebhookParams{
		ID:               input.ID,
		Url:              input.URL,
		CronSchedule:     input.CronSchedule,
		Instruction:      input.Instruction,
		PassApiKey:       input.PassAPIKey,
		TimeoutSeconds:   input.TimeoutSeconds,
		MaxDepth:         input.MaxDepth,
		MaxRetries:       input.MaxRetries,
		Enabled:          input.Enabled,
		Description:      input.Description,
		TransformJsonnet: input.TransformJsonnet,
		ConcurrencyMode:  input.ConcurrencyMode,
	}

	// Marshal JSON arrays only if provided.
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

	webhook, err := env.UpdateCronWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update cron webhook: %w", err)
	}

	// Recalculate next_run_at if cron_schedule changed.
	if input.CronSchedule != nil {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, _ := parser.Parse(webhook.CronSchedule)
		nextRun := schedule.Next(time.Now())
		updateErr := env.UpdateCronWebhookNextRunAt(ctx, db.UpdateCronWebhookNextRunAtParams{
			ID:        webhook.ID,
			NextRunAt: ptr.To(nextRun),
		})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update next_run_at: %w", updateErr)
		}
		webhook.NextRunAt = ptr.To(nextRun)
	}

	return &model.UpdateCronWebhookPayload{
		CronWebhook: &webhook,
	}, nil
}
