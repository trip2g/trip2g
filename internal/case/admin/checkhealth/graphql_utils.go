package checkhealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data interface{} `json:"data"`
}

// makeGraphQLRequest makes a POST request to the GraphQL endpoint with the given query.
func makeGraphQLRequest(ctx context.Context, graphqlURL, query string, headers map[string]string) (*graphQLResponse, error) {
	reqBody := graphQLRequest{Query: query}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var graphqlResp graphQLResponse
	err = json.Unmarshal(body, &graphqlResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &graphqlResp, nil
}

// hasErrorContaining checks if the response has an error message containing the given substring.
func hasErrorContaining(resp *graphQLResponse, substring string) bool {
	if len(resp.Errors) == 0 {
		return false
	}

	for _, e := range resp.Errors {
		if contains(e.Message, substring) {
			return true
		}
	}

	return false
}

// contains is a case-sensitive substring check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getFirstErrorMessage returns the first error message from the response, or "no error".
func getFirstErrorMessage(resp *graphQLResponse) string {
	if len(resp.Errors) > 0 {
		return resp.Errors[0].Message
	}
	return "no error"
}
