package codellm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/webhookutil"
)

func TestChangeToToolCall(t *testing.T) {
	tests := []struct {
		name     string
		change   webhookutil.AgentChange
		wantName string
		wantArgs string
	}{
		{
			name:     "legacy change without a kind is a write",
			change:   webhookutil.AgentChange{Path: "a.md", Content: "x"},
			wantName: "write_note",
			wantArgs: `{"content":"x","path":"a.md"}`,
		},
		{
			name:     "write",
			change:   webhookutil.AgentChange{Path: "a.md", Content: "x", Kind: webhookutil.AgentChangeKindWrite},
			wantName: "write_note",
			wantArgs: `{"content":"x","path":"a.md"}`,
		},
		{
			name:     "patch",
			change:   webhookutil.AgentChange{Path: "a.md", Find: "a", Replace: "b", Kind: webhookutil.AgentChangeKindPatch},
			wantName: "patch_note",
			wantArgs: `{"find":"a","path":"a.md","replace":"b"}`,
		},
		{
			name: "meta",
			change: webhookutil.AgentChange{
				Path:  "a.md",
				Set:   map[string]any{"status": "done"},
				Unset: []string{"draft"},
				Kind:  webhookutil.AgentChangeKindMeta,
			},
			wantName: "set_note_meta",
			wantArgs: `{"path":"a.md","set":{"status":"done"},"unset":["draft"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args, err := changeToToolCall(tt.change)
			require.NoError(t, err)
			require.Equal(t, tt.wantName, name)
			require.JSONEq(t, tt.wantArgs, args)
		})
	}
}

func TestChangeToToolCallRefusesAnUnknownKind(t *testing.T) {
	change := webhookutil.AgentChange{Path: "a.md", Kind: "rename"}

	_, _, err := changeToToolCall(change)
	require.ErrorContains(t, err, `unknown kind "rename"`)

	_, err = buildResponse("", []webhookutil.AgentChange{change}, "done")
	require.Error(t, err)
}
