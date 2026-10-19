package webhookutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/frontmatteredit"
	"trip2g/internal/model"
)

func TestParseAgentResponse_MetaChange(t *testing.T) {
	body := []byte(`{"changes":[{"path":"a.md","kind":"meta","set":{"status":"done","n":3},"unset":["draft"]}]}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.Len(t, resp.Changes, 1)
	require.True(t, resp.Changes[0].IsMeta())
	require.Equal(t, map[string]any{"status": "done", "n": float64(3)}, resp.Changes[0].Set)
	require.Equal(t, []string{"draft"}, resp.Changes[0].Unset)
}

func TestParseAgentResponse_InvalidMetaChange(t *testing.T) {
	tests := []struct {
		name    string
		change  string
		wantErr error
	}{
		{
			name:    "nothing to change",
			change:  `{"path":"a.md","kind":"meta"}`,
			wantErr: frontmatteredit.ErrNothingToChange,
		},
		{
			name:    "empty set and unset",
			change:  `{"path":"a.md","kind":"meta","set":{},"unset":[]}`,
			wantErr: frontmatteredit.ErrNothingToChange,
		},
		{
			name:    "key both set and unset",
			change:  `{"path":"a.md","kind":"meta","set":{"status":"done"},"unset":["status"]}`,
			wantErr: frontmatteredit.ErrSetAndUnset,
		},
		{
			name:    "blank set key",
			change:  `{"path":"a.md","kind":"meta","set":{" ":"x"}}`,
			wantErr: frontmatteredit.ErrEmptyKey,
		},
		{
			name:    "blank unset key",
			change:  `{"path":"a.md","kind":"meta","unset":[""]}`,
			wantErr: frontmatteredit.ErrEmptyKey,
		},
		{
			name:    "key with a newline",
			change:  `{"path":"a.md","kind":"meta","set":{"a\nb":"x"}}`,
			wantErr: frontmatteredit.ErrControlInKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAgentResponse([]byte(`{"changes":[` + tt.change + `]}`))
			require.ErrorContains(t, err, "invalid change at index 0")
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestResolveAgentChanges_Meta(t *testing.T) {
	nvs := model.NewNoteViews()
	nvs.RegisterNote(&model.NoteView{
		PathID:    1,
		Path:      "a.md",
		Permalink: "a.md",
		Content:   []byte("---\ntitle: A\ndraft: true\n---\nBody\n"),
	})

	tests := []struct {
		name    string
		change  AgentChange
		want    []model.RawNote
		wantErr string
	}{
		{
			name:   "sets and unsets keys of the stored note",
			change: AgentChange{Path: "a.md", Kind: AgentChangeKindMeta, Set: map[string]any{"status": "done", "n": float64(3)}, Unset: []string{"draft"}},
			want:   []model.RawNote{{Path: "a.md", Content: "---\ntitle: A\nn: 3\nstatus: done\n---\nBody\n"}},
		},
		{
			name:    "missing note",
			change:  AgentChange{Path: "b.md", Kind: AgentChangeKindMeta, Unset: []string{"draft"}},
			wantErr: "note not found for meta: b.md",
		},
		{
			name:    "outside write patterns",
			change:  AgentChange{Path: "secret/a.md", Kind: AgentChangeKindMeta, Unset: []string{"draft"}},
			wantErr: "not allowed by write_patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAgentChanges([]AgentChange{tt.change}, nvs, []string{"*.md"})
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
