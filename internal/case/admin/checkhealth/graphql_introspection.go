package checkhealth

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/graph/model"
)

type GraphQLIntrospectionChecker struct{}

func (c *GraphQLIntrospectionChecker) ID() string {
	return "graphql_introspection"
}

func (c *GraphQLIntrospectionChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.GetPublicURLForRequest(ctx)
	if publicURL == "" {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: "Unable to determine public URL for request",
		}
	}

	graphqlURL := publicURL + "/graphql"

	// Introspection query to test if introspection is disabled
	introspectionQuery := "query IntrospectionQuery { __schema { queryType { name } } }"

	resp, err := makeGraphQLRequest(ctx, graphqlURL, introspectionQuery, nil)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to make request: %v", err),
		}
	}

	// Check if introspection is disabled
	// We expect an error message about introspection being disabled
	if len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
			if strings.Contains(strings.ToLower(e.Message), "introspection") {
				return model.HealchCheck{
					ID:          c.ID(),
					Status:      model.HealthCheckStatusOk,
					Description: "GraphQL introspection is properly disabled for unauthenticated requests",
				}
			}
		}

		// Got errors but not about introspection
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: fmt.Sprintf("Unexpected error: %s", getFirstErrorMessage(resp)),
		}
	}

	// No errors means introspection is enabled - this is bad in production
	if resp.Data != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: "GraphQL introspection is ENABLED for unauthenticated requests - security risk!",
		}
	}

	// Unknown state
	return model.HealchCheck{
		ID:          c.ID(),
		Status:      model.HealthCheckStatusWarning,
		Description: "Unable to determine introspection status",
	}
}
