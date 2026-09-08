package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Wall wins: a note is teasable only when it belongs to at least one subgraph
// and every subgraph it belongs to carries the teaser flag.
func TestNoteView_IsTeasable(t *testing.T) {
	teaser := &NoteSubgraph{Name: "shop", Teaser: true}
	closed := &NoteSubgraph{Name: "vault"}

	tests := []struct {
		name      string
		subgraphs map[string]*NoteSubgraph
		expected  bool
	}{
		{
			name:      "no subgraph at all",
			subgraphs: nil,
			expected:  false,
		},
		{
			name:      "single teaser subgraph",
			subgraphs: map[string]*NoteSubgraph{"shop": teaser},
			expected:  true,
		},
		{
			name:      "single non-teaser subgraph",
			subgraphs: map[string]*NoteSubgraph{"vault": closed},
			expected:  false,
		},
		{
			name:      "every subgraph is a teaser",
			subgraphs: map[string]*NoteSubgraph{"shop": teaser, "shop2": {Name: "shop2", Teaser: true}},
			expected:  true,
		},
		{
			name:      "one closed subgraph walls the note",
			subgraphs: map[string]*NoteSubgraph{"shop": teaser, "vault": closed},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &NoteView{Path: "n.md", Subgraphs: tt.subgraphs}
			require.Equal(t, tt.expected, note.IsTeasable())
		})
	}
}

func TestNoteView_IsTeasable_NilNote(t *testing.T) {
	var note *NoteView
	require.False(t, note.IsTeasable())
}
