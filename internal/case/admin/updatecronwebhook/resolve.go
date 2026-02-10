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

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Validate cron schedule if provided.
	if input.CronSchedule != nil {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		_, parseErr := parser.Parse(*input.CronSchedule)
		if parseErr != nil {
			return &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "cronSchedule", Value: "invalid cron expression: " + parseErr.Error()},
				},
			}, nil
		}
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
