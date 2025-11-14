package checkhealth

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/graph/model"
)

type AdminAuthorizationChecker struct{}

func (c *AdminAuthorizationChecker) ID() string {
	return "admin_authorization"
}

func (c *AdminAuthorizationChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.GetPublicURLForRequest(ctx)
	if publicURL == "" {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: "Unable to determine public URL for request",
		}
	}

	graphqlURL := publicURL + "/graphql"

	// Query that requires admin authorization
	adminQuery := "query { admin { latestConfig { timezone } } }"

	// Intentionally NOT setting authorization header
	resp, err := makeGraphQLRequest(ctx, graphqlURL, adminQuery, nil)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request: %v", err),
		}
	}

	// Check that we got an "unauthorized" error
	if len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
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
			Description: fmt.Sprintf("Unexpected error response: %s", getFirstErrorMessage(resp)),
		}
	}

	// No errors means authorization was not enforced - this is bad!
	if resp.Data != nil {
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
