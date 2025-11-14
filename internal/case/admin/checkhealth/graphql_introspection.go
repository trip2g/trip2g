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

type GraphQLIntrospectionChecker struct{}

func (c *GraphQLIntrospectionChecker) ID() string {
	return "graphql_introspection"
}

func (c *GraphQLIntrospectionChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.PublicURL()
	graphqlURL := publicURL + "/graphql"

	// Introspection query to test if introspection is disabled
	introspectionQuery := `{
		"query": "query IntrospectionQuery { __schema { queryType { name } } }"
	}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewBufferString(introspectionQuery))
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")

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

	// Check if introspection is disabled
	// We expect an error message about introspection being disabled
	if len(graphqlResp.Errors) > 0 {
		for _, e := range graphqlResp.Errors {
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
			Description: fmt.Sprintf("Unexpected error: %s", graphqlResp.Errors[0].Message),
		}
	}

	// No errors means introspection is enabled - this is bad in production
	if graphqlResp.Data != nil {
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
