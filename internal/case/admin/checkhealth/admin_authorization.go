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

type AdminAuthorizationChecker struct{}

func (c *AdminAuthorizationChecker) ID() string {
	return "admin_authorization"
}

func (c *AdminAuthorizationChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.PublicURL()
	graphqlURL := publicURL + "/graphql"

	// Query that requires admin authorization
	adminQuery := `{
		"query": "query { admin { latestConfig { timezone } } }"
	}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewBufferString(adminQuery))
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	// Intentionally NOT setting authorization header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request to %s: %v", graphqlURL, err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to read response: %v", err),
		}
	}

	// Parse response
	var graphqlResp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Data interface{} `json:"data"`
	}

	err = json.Unmarshal(body, &graphqlResp)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to parse response: %v", err),
		}
	}

	// Check that we got an "unauthorized" error
	if len(graphqlResp.Errors) > 0 {
		for _, e := range graphqlResp.Errors {
			lowerMsg := strings.ToLower(e.Message)
			if strings.Contains(lowerMsg, "unauthorized") {
				return model.HealchCheck{
					ID:          c.ID(),
					Status:      model.HealthCheckStatusOk,
					Description: "Admin authorization is properly enforced - unauthorized requests are blocked",
				}
			}
		}

		// Got errors but not "unauthorized"
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: fmt.Sprintf("Unexpected error response: %s", graphqlResp.Errors[0].Message),
		}
	}

	// No errors means authorization was not enforced - this is bad!
	if graphqlResp.Data != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: "Admin authorization is NOT enforced - unauthenticated requests can access admin data!",
		}
	}

	// Unknown state
	return model.HealchCheck{
		ID:          c.ID(),
		Status:      model.HealthCheckStatusWarning,
		Description: "Unable to determine admin authorization status",
	}
}
