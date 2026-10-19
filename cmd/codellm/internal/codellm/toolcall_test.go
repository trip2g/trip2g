package codellm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/coderun"
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

func TestChangeToToolCallRefusesAKindItDoesNotKnow(t *testing.T) {
	for _, kind := range []string{"", "upsert", "rename"} {
		change := webhookutil.AgentChange{Path: "a.md", Content: "x", Kind: kind}

		_, _, err := changeToToolCall(change)
		require.ErrorContains(t, err, fmt.Sprintf("unknown kind %q", kind))

		_, err = buildResponse("", []webhookutil.AgentChange{change}, "done")
		require.Error(t, err)
	}
}

func TestBuildResponseFromCodeOutput(t *testing.T) {
	stdout := `{"changes":[` +
		`{"path":"a.md","content":"# A\n"},` +
		`{"path":"b.md","find":"x","replace":"y"},` +
		`{"path":"c.md","set":{"status":"done","n":9007199254740993},"unset":["draft"]}` +
		`],"answer":"done"}`
	changes, answer, err := coderun.ParseCodeOutput(stdout)
	require.NoError(t, err)

	resp, err := buildResponse("", changes, answer)
	require.NoError(t, err)

	type call struct{ Name, Args string }
	var got []call
	for _, tc := range resp.Choices[0].Message.ToolCalls {
		got = append(got, call{tc.Function.Name, tc.Function.Arguments})
	}
	require.Len(t, got, 4)
	require.Equal(t, "write_note", got[0].Name)
	require.JSONEq(t, `{"path":"a.md","content":"# A\n"}`, got[0].Args)
	require.Equal(t, "patch_note", got[1].Name)
	require.JSONEq(t, `{"path":"b.md","find":"x","replace":"y"}`, got[1].Args)
	require.Equal(t, "set_note_meta", got[2].Name)
	require.JSONEq(t, `{"path":"c.md","set":{"status":"done","n":9007199254740993},"unset":["draft"]}`, got[2].Args)
	require.Contains(t, got[2].Args, "9007199254740993")
	require.Equal(t, "finish", got[3].Name)
	require.JSONEq(t, `{"answer":"done"}`, got[3].Args)
}
