package createwebhook

import (
	"context"
	"encoding/json"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	InsertWebhook(ctx context.Context, params db.InsertWebhookParams) (db.ChangeWebhook, error)
}

type Input = model.ChangeWebhookCreateInput
type Payload = model.ChangeWebhookCreateOrErrorPayload

func validateInput(i *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(i,
		ozzo.Field(&i.URL, ozzo.Required, is.URL),
		ozzo.Field(&i.IncludePatterns, ozzo.Required, ozzo.Length(1, 0)),
	))
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
	includeJSON, err := json.Marshal(input.IncludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal include_patterns: %w", err)
	}

	excludePatterns := input.ExcludePatterns
	if excludePatterns == nil {
		excludePatterns = []string{}
	}
	excludeJSON, err := json.Marshal(excludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal exclude_patterns: %w", err)
	}

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
	maxDepth := int64(1)
	if input.MaxDepth != nil {
		maxDepth = *input.MaxDepth
	}
	passAPIKey := false
	if input.PassAPIKey != nil {
		passAPIKey = *input.PassAPIKey
	}
	includeContent := true
	if input.IncludeContent != nil {
		includeContent = *input.IncludeContent
	}
	timeoutSeconds := int64(60)
	if input.TimeoutSeconds != nil {
		timeoutSeconds = *input.TimeoutSeconds
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
	onCreate := true
	if input.OnCreate != nil {
		onCreate = *input.OnCreate
	}
	onUpdate := true
	if input.OnUpdate != nil {
		onUpdate = *input.OnUpdate
	}
	onRemove := true
	if input.OnRemove != nil {
		onRemove = *input.OnRemove
	}

	boundsErr := validateBounds(maxDepth, timeoutSeconds, maxRetries)
	if boundsErr != nil {
		return boundsErr, nil
	}

	params := db.InsertWebhookParams{
		Url:             input.URL,
		IncludePatterns: string(includeJSON),
		ExcludePatterns: string(excludeJSON),
		Instruction:     instruction,
		Secret:          secret,
		MaxDepth:        maxDepth,
		PassApiKey:      passAPIKey,
		IncludeContent:  includeContent,
		TimeoutSeconds:  timeoutSeconds,
		MaxRetries:      maxRetries,
		Description:     description,
		OnCreate:        onCreate,
		OnUpdate:        onUpdate,
		OnRemove:        onRemove,
		ReadPatterns:    string(readJSON),
		WritePatterns:   string(writeJSON),
		CreatedBy:       int64(token.ID),
	}

	webhook, err := env.InsertWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert webhook: %w", err)
	}

	return &model.ChangeWebhookCreatePayload{
		Webhook: &webhook,
		Secret:  secret,
	}, nil
}
