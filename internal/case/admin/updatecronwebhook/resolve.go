package updatecronwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UpdateCronWebhook(ctx context.Context, params db.UpdateCronWebhookParams) (db.CronWebhook, error)
	UpdateCronWebhookNextRunAt(ctx context.Context, params db.UpdateCronWebhookNextRunAtParams) error
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

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	scheduleErr := validateCronSchedule(input.CronSchedule)
	if scheduleErr != nil {
		return scheduleErr, nil
	}

	boundsErr := validateBounds(input)
	if boundsErr != nil {
		return boundsErr, nil
	}

	params := db.UpdateCronWebhookParams{
		ID:             input.ID,
		Url:            input.URL,
		CronSchedule:   input.CronSchedule,
		Instruction:    input.Instruction,
		PassApiKey:     input.PassAPIKey,
		TimeoutSeconds: input.TimeoutSeconds,
		MaxDepth:       input.MaxDepth,
		MaxRetries:     input.MaxRetries,
		Enabled:        input.Enabled,
		Description:    input.Description,
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
