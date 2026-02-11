package webhookutil

import (
	"encoding/json"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// AgentResponse is the expected format of a webhook agent's response body.
type AgentResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Changes []AgentChange `json:"changes"`
}

// AgentChange represents a single file change from an agent.
type AgentChange struct {
	Path         string  `json:"path"`
	Content      string  `json:"content"`
	ExpectedHash *string `json:"expected_hash,omitempty"`
}

// Validate validates required fields of an AgentChange.
func (c AgentChange) Validate() error {
	return ozzo.ValidateStruct(&c,
		ozzo.Field(&c.Path, ozzo.Required),
		ozzo.Field(&c.Content, ozzo.Required),
	)
}

// ParseAgentResponse parses a webhook response body into AgentResponse.
// Returns nil, nil if body is empty or not valid JSON (non-fatal).
func ParseAgentResponse(body []byte) (*AgentResponse, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var resp AgentResponse

	err := json.Unmarshal(body, &resp)
	if err != nil {
		// Not valid JSON — acceptable, agent may return non-JSON responses.
		return nil, nil //nolint:nilerr // invalid JSON is acceptable from agents.
	}

	for i, change := range resp.Changes {
		err = change.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid change at index %d: %w", i, err)
		}
	}

	return &resp, nil
}
