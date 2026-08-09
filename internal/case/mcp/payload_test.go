package mcp_test

import (
	"encoding/json"
	"testing"

	"trip2g/internal/case/mcp"

	"github.com/stretchr/testify/require"
)

// decodePayload decodes a tool result's structured payload. Results travel
// through the MCP transport as JSON, so structuredContent arrives as raw bytes
// rather than the in-memory value the handler built — the same shape a real
// client sees.
func decodePayload[T any](t *testing.T, result mcp.CallToolResult) T {
	t.Helper()

	var payload T
	raw, ok := result.StructuredContent.(json.RawMessage)
	require.True(t, ok, "expected raw structured content, got %T", result.StructuredContent)
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}
