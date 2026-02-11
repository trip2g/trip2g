package createcronwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/robfig/cron/v3"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	InsertCronWebhook(ctx context.Context, params db.InsertCronWebhookParams) (db.CronWebhook, error)
}

type Input = model.CreateCronWebhookInput
type Payload = model.CreateCronWebhookOrErrorPayload

func validateInput(i *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(i,
		ozzo.Field(&i.URL, ozzo.Required, is.URL),
		ozzo.Field(&i.CronSchedule, ozzo.Required),
	))
}

// parseCronSchedule validates and computes next run time.
func parseCronSchedule(expression string) (time.Time, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expression)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(time.Now()), nil
}

func validateBounds(maxDepth, timeoutSeconds, maxRetries int64) *model.ErrorPayload {
	var errs []model.FieldMessage

	if maxDepth < 0 || maxDepth > 999 {
		errs = append(errs, model.FieldMessage{Name: "maxDepth", Value: "must be between 0 and 999"})
	}
	if timeoutSeconds < 1 || timeoutSeconds > 3600 {
		errs = append(errs, model.FieldMessage{Name: "timeoutSeconds", Value: "must be between 1 and 3600"})
	}
	if maxRetries < 0 || maxRetries > 100 {
		errs = append(errs, model.FieldMessage{Name: "maxRetries", Value: "must be between 0 and 100"})
	}

	if len(errs) > 0 {
		return &model.ErrorPayload{ByFields: errs}
	}
	return nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateInput(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Validate cron expression.
	nextRunAt, err := parseCronSchedule(input.CronSchedule)
	if err != nil {
		//nolint:nilerr // returning user-facing validation error.
		return &model.ErrorPayload{
			ByFields: []model.FieldMessage{
				{Name: "cronSchedule", Value: "invalid cron expression: " + err.Error()},
			},
		}, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Generate secret if not provided.
	var secret string
	if input.Secret != nil && *input.Secret != "" {
		secret = *input.Secret
	} else {
		secret, err = webhookutil.GenerateSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret: %w", err)
		}
	}

	// Marshal JSON arrays.
	readPatterns := input.ReadPatterns
	if readPatterns == nil {
		readPatterns = []string{"*"}
	}
	readJSON, err := json.Marshal(readPatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal read_patterns: %w", err)
	}

	writePatterns := input.WritePatterns
	if writePatterns == nil {
		writePatterns = []string{}
	}
	writeJSON, err := json.Marshal(writePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal write_patterns: %w", err)
	}

	// Set defaults for optional fields.
	passAPIKey := false
	if input.PassAPIKey != nil {
		passAPIKey = *input.PassAPIKey
	}
	timeoutSeconds := int64(60)
	if input.TimeoutSeconds != nil {
		timeoutSeconds = *input.TimeoutSeconds
	}
	maxDepth := int64(1)
	if input.MaxDepth != nil {
		maxDepth = *input.MaxDepth
	}
	maxRetries := int64(0)
	if input.MaxRetries != nil {
		maxRetries = *input.MaxRetries
	}
	description := ""
	if input.Description != nil {
		description = *input.Description
	}
	instruction := ""
	if input.Instruction != nil {
		instruction = *input.Instruction
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	boundsErr := validateBounds(maxDepth, timeoutSeconds, maxRetries)
	if boundsErr != nil {
		return boundsErr, nil
	}

	// Compute nextRunAt only if enabled.
	var nextRunAtPtr *time.Time
	if enabled {
		nextRunAtPtr = ptr.To(nextRunAt)
	}

	params := db.InsertCronWebhookParams{
		Url:            input.URL,
		CronSchedule:   input.CronSchedule,
		Instruction:    instruction,
		Secret:         secret,
		PassApiKey:     passAPIKey,
		TimeoutSeconds: timeoutSeconds,
		MaxDepth:       maxDepth,
		MaxRetries:     maxRetries,
		NextRunAt:      nextRunAtPtr,
		ReadPatterns:   string(readJSON),
		WritePatterns:  string(writeJSON),
		Description:    description,
		CreatedBy:      int64(token.ID),
	}

	webhook, err := env.InsertCronWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert cron webhook: %w", err)
	}

	return &model.CreateCronWebhookPayload{
		CronWebhook: &webhook,
		Secret:      secret,
	}, nil
}
