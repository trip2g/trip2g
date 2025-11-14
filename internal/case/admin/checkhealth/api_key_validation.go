package checkhealth

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
)

type APIKeyValidationChecker struct{}

func (c *APIKeyValidationChecker) ID() string {
	return "api_key_validation"
}

func (c *APIKeyValidationChecker) checkMissingAPIKey(ctx context.Context, graphqlURL, apiQuery string) *model.HealchCheck {
	// Intentionally NOT setting X-API-Key header
	resp, err := makeGraphQLRequest(ctx, graphqlURL, apiQuery, nil)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request without API key: %v", err),
		}
	}

	// Check for "missing X-API-Key" error
	if !hasErrorContaining(resp, "missing X-API-Key in request header") {
		if resp.Data != nil {
			return &model.HealchCheck{
				ID:          c.ID(),
				Status:      model.HealthCheckStatusCritical,
				Description: "API key validation is NOT enforced - requests without X-API-Key can access data!",
			}
		}
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: fmt.Sprintf("Expected 'missing X-API-Key' error, got: %s", getFirstErrorMessage(resp)),
		}
	}

	return nil // success
}

func (c *APIKeyValidationChecker) checkInvalidAPIKey(ctx context.Context, graphqlURL, apiQuery string) *model.HealchCheck {
	// Set invalid API key
	headers := map[string]string{
		"X-API-Key": "test",
	}

	resp, err := makeGraphQLRequest(ctx, graphqlURL, apiQuery, headers)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request with invalid API key: %v", err),
		}
	}

	// Check for "invalid API key" error
	if !hasErrorContaining(resp, "invalid API key") {
		if resp.Data != nil {
			return &model.HealchCheck{
				ID:          c.ID(),
				Status:      model.HealthCheckStatusCritical,
				Description: "Invalid API keys are accepted - requests with fake API keys can access data!",
			}
		}
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: fmt.Sprintf("Expected 'invalid API key' error, got: %s", getFirstErrorMessage(resp)),
		}
	}

	return nil // success
}

func (c *APIKeyValidationChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.GetPublicURLForRequest(ctx)
	if publicURL == "" {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: "Unable to determine public URL for request",
		}
	}

	graphqlURL := publicURL + "/graphql"

	// Query that requires API key
	apiQuery := "query { notePaths { value } }"

	// First check: request without X-API-Key header
	if result := c.checkMissingAPIKey(ctx, graphqlURL, apiQuery); result != nil {
		return *result
	}

	// Second check: request with invalid API key "test"
	if result := c.checkInvalidAPIKey(ctx, graphqlURL, apiQuery); result != nil {
		return *result
	}

	// Both checks passed
	return model.HealchCheck{
		ID:          c.ID(),
		Status:      model.HealthCheckStatusOk,
		Description: "API key validation is properly enforced - missing and invalid keys are rejected",
	}
}
