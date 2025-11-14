package checkhealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"trip2g/internal/graph/model"
)

type APIKeyValidationChecker struct{}

func (c *APIKeyValidationChecker) ID() string {
	return "api_key_validation"
}

type graphqlResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data interface{} `json:"data"`
}

func (c *APIKeyValidationChecker) checkMissingAPIKey(ctx context.Context, graphqlURL, apiQuery string) *model.HealchCheck {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewBufferString(apiQuery))
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to create request: %v", err),
		}
	}
	req.Header.Set("Content-Type", "application/json")
	// Intentionally NOT setting X-API-Key header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request without API key: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to read response: %v", err),
		}
	}

	var graphqlResp graphqlResponse
	err = json.Unmarshal(body, &graphqlResp)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to parse response: %v", err),
		}
	}

	// Check for "missing X-API-Key" error
	foundMissingKeyError := false
	if len(graphqlResp.Errors) > 0 {
		for _, e := range graphqlResp.Errors {
			if strings.Contains(e.Message, "missing X-API-Key in request header") {
				foundMissingKeyError = true
				break
			}
		}
	}

	if !foundMissingKeyError {
		if graphqlResp.Data != nil {
			return &model.HealchCheck{
				ID:          c.ID(),
				Status:      model.HealthCheckStatusCritical,
				Description: "API key validation is NOT enforced - requests without X-API-Key can access data!",
			}
		}
		errorMsg := "no error"
		if len(graphqlResp.Errors) > 0 {
			errorMsg = graphqlResp.Errors[0].Message
		}
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: fmt.Sprintf("Expected 'missing X-API-Key' error, got: %s", errorMsg),
		}
	}

	return nil // success
}

func (c *APIKeyValidationChecker) checkInvalidAPIKey(ctx context.Context, graphqlURL, apiQuery string) *model.HealchCheck {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewBufferString(apiQuery))
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to create second request: %v", err),
		}
	}
	req.Header.Set("Content-Type", "application/json")
	//nolint:canonicalheader // X-API-Key is the custom header name used by the API
	req.Header.Set("X-API-Key", "test")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request with invalid API key: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to read second response: %v", err),
		}
	}

	var graphqlResp graphqlResponse
	err = json.Unmarshal(body, &graphqlResp)
	if err != nil {
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to parse second response: %v", err),
		}
	}

	// Check for "invalid API key" error
	foundInvalidKeyError := false
	if len(graphqlResp.Errors) > 0 {
		for _, e := range graphqlResp.Errors {
			if strings.Contains(e.Message, "invalid API key") {
				foundInvalidKeyError = true
				break
			}
		}
	}

	if !foundInvalidKeyError {
		if graphqlResp.Data != nil {
			return &model.HealchCheck{
				ID:          c.ID(),
				Status:      model.HealthCheckStatusCritical,
				Description: "Invalid API keys are accepted - requests with fake API keys can access data!",
			}
		}
		errorMsg := "no error"
		if len(graphqlResp.Errors) > 0 {
			errorMsg = graphqlResp.Errors[0].Message
		}
		return &model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: fmt.Sprintf("Expected 'invalid API key' error, got: %s", errorMsg),
		}
	}

	return nil // success
}

func (c *APIKeyValidationChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.PublicURL()
	graphqlURL := publicURL + "/graphql"

	// Query that requires API key
	apiQuery := `{
		"query": "query { notePaths { value } }"
	}`

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
