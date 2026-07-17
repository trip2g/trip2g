package coderun

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInterpreterNamesAreFenceLabels pins the wire invariant the exec tool
// relies on: every interpreter name is also one of its own fence labels, so a
// fenced block labeled with the canonical program name (```python, ```bash, …)
// always resolves back to that interpreter. Fleet's exec tool wraps exec code
// in exactly such a block before sending it to codellm.
func TestInterpreterNamesAreFenceLabels(t *testing.T) {
	var payload struct {
		Interpreters []interpreter `json:"interpreters"`
	}
	require.NoError(t, json.Unmarshal(defaultInterpretersJSON, &payload))
	require.NotEmpty(t, payload.Interpreters)

	for _, interp := range payload.Interpreters {
		require.Contains(t, interp.CodeBlockLabels, interp.Name,
			"interpreter %q must list its own name among code_block_labels", interp.Name)
	}
}
