package createfrontmatterpatch

import (
	"context"
	"encoding/json"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertFrontmatterPatch(ctx context.Context, arg db.InsertFrontmatterPatchParams) (db.NoteFrontmatterPatch, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
}

// Input is an alias for the GraphQL input type.
type Input = model.CreateFrontmatterPatchInput

// Payload is an alias for the GraphQL payload type.
type Payload = model.CreateFrontmatterPatchOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.IncludePatterns, validation.Required, validation.Length(1, 0)),
		validation.Field(&r.Jsonnet, validation.Required),
		// Priority validation removed - 0 and negative values are valid
		validation.Field(&r.Description, validation.Required),
	))
}

// Resolve creates a new frontmatter patch.
func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Check admin authorization.
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Always validate input first.
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Validate patterns.
	err = frontmatterpatch.ValidatePatterns(input.IncludePatterns)
	if err != nil {
		return model.NewFieldError("includePatterns", "invalid glob pattern"), nil //nolint:nilerr // validation error → field error
	}

	if len(input.ExcludePatterns) > 0 {
		err = frontmatterpatch.ValidatePatterns(input.ExcludePatterns)
		if err != nil {
			return model.NewFieldError("excludePatterns", "invalid glob pattern"), nil //nolint:nilerr // validation error → field error
		}
	}

	// Validate jsonnet.
	err = frontmatterpatch.ValidateJsonnet(input.Jsonnet)
	if err != nil {
		return model.NewFieldError("jsonnet", "invalid jsonnet: "+err.Error()), nil //nolint:nilerr // validation error → field error
	}

	// Marshal patterns to JSON.
	includePatternsJSON, err := json.Marshal(input.IncludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal include patterns: %w", err)
	}

	excludePatternsJSON, err := json.Marshal(input.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal exclude patterns: %w", err)
	}

	// Define params as separate variable for cleaner code.
	params := db.InsertFrontmatterPatchParams{
		IncludePatterns: string(includePatternsJSON),
		ExcludePatterns: string(excludePatternsJSON),
		Jsonnet:         input.Jsonnet,
		Priority:        int64(input.Priority),
		Description:     input.Description,
		Enabled:         input.Enabled,
		CreatedBy:       int64(token.ID),
	}

	// Execute database operation.
	patch, err := env.InsertFrontmatterPatch(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert frontmatter patch: %w", err)
	}

	// Reload notes to apply the new patch.
	_, err = env.PrepareLatestNotes(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to reload notes after patch creation: %w", err)
	}

	// Define payload as separate variable.
	payload := model.CreateFrontmatterPatchPayload{
		FrontmatterPatch: &patch,
	}

	return &payload, nil
}
