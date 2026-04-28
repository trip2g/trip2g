package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractMCPFederationFields(t *testing.T) {
	tests := []struct {
		name      string
		rawMeta   map[string]interface{}
		wantURL   string
		wantID    string
		wantDepth int
	}{
		{
			name: "url only",
			rawMeta: map[string]interface{}{
				"mcp_federation_kb_url": "https://bob.team.io/_system/mcp",
			},
			wantURL: "https://bob.team.io/_system/mcp",
		},
		{
			name: "id overrides hostname and depth parses",
			rawMeta: map[string]interface{}{
				"mcp_federation_kb_url":       "https://bob.team.io/_system/mcp",
				"mcp_federation_kb_id":        "bob",
				"mcp_federation_kb_max_depth": 2,
			},
			wantURL:   "https://bob.team.io/_system/mcp",
			wantID:    "bob",
			wantDepth: 2,
		},
		{
			name: "depth parses from string",
			rawMeta: map[string]interface{}{
				"mcp_federation_kb_url":       "https://science.example/_system/mcp",
				"mcp_federation_kb_max_depth": "3",
			},
			wantURL:   "https://science.example/_system/mcp",
			wantDepth: 3,
		},
		{
			name: "non-integer depth defaults to zero",
			rawMeta: map[string]interface{}{
				"mcp_federation_kb_url":       "https://science.example/_system/mcp",
				"mcp_federation_kb_max_depth": "many",
			},
			wantURL: "https://science.example/_system/mcp",
		},
		{
			name:    "no url leaves fields empty",
			rawMeta: map[string]interface{}{"mcp_federation_kb_id": "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &NoteView{RawMeta: tt.rawMeta}

			err := note.ExtractMetaData()

			require.NoError(t, err)
			require.Equal(t, tt.wantURL, note.MCPFederationKBURL)
			require.Equal(t, tt.wantID, note.MCPFederationKBID)
			require.Equal(t, tt.wantDepth, note.MCPFederationKBMaxDepth)
		})
	}
}

func TestNewMCPFederationNote(t *testing.T) {
	t.Run("hostname id fallback", func(t *testing.T) {
		note := &NoteView{
			MCPFederationKBURL: "https://bob.team.io/_system/mcp",
		}

		kb := newMCPFederationNote(note)

		require.NotNil(t, kb)
		require.Equal(t, note, kb.Note)
		require.Equal(t, "https://bob.team.io/_system/mcp", kb.URL)
		require.Equal(t, "bob.team.io", kb.ID)
		require.Zero(t, kb.MaxDepth)
	})

	t.Run("explicit id and max depth", func(t *testing.T) {
		note := &NoteView{
			MCPFederationKBURL:      "https://bob.team.io/_system/mcp",
			MCPFederationKBID:       "bob",
			MCPFederationKBMaxDepth: 2,
		}

		kb := newMCPFederationNote(note)

		require.NotNil(t, kb)
		require.Equal(t, "bob", kb.ID)
		require.Equal(t, 2, kb.MaxDepth)
	})

	t.Run("no url is not a federation note", func(t *testing.T) {
		require.Nil(t, newMCPFederationNote(&NoteView{}))
	})

	t.Run("malformed url has empty fallback id", func(t *testing.T) {
		kb := newMCPFederationNote(&NoteView{
			MCPFederationKBURL: "://bad",
		})

		require.NotNil(t, kb)
		require.Empty(t, kb.ID)
	})
}

func TestHostnameFromURL(t *testing.T) {
	require.Equal(t, "bob.team.io", hostnameFromURL("https://bob.team.io/_system/mcp"))
	require.Equal(t, "bob.team.io", hostnameFromURL(" https://BOB.Team.IO/_system/mcp "))
	require.Empty(t, hostnameFromURL("://bad"))
}

func TestNoteViewsExtractMCPFederationNotes(t *testing.T) {
	nvs := NewNoteViews()
	kbNote := &NoteView{
		PathID:                  1,
		Permalink:               "/kb/bob",
		MCPFederationKBURL:      "https://bob.team.io/_system/mcp",
		MCPFederationKBID:       "bob",
		MCPFederationKBMaxDepth: 1,
	}
	regularNote := &NoteView{
		PathID:    2,
		Permalink: "/regular",
	}
	nvs.Map[kbNote.Permalink] = kbNote
	nvs.Map[regularNote.Permalink] = regularNote

	nvs.ExtractMCPFederationNotes()

	require.Len(t, nvs.MCPFederationNotes, 1)
	require.Equal(t, kbNote, nvs.MCPFederationNotes[0].Note)
	require.Equal(t, "bob", nvs.MCPFederationNotes[0].ID)
	require.Equal(t, "https://bob.team.io/_system/mcp", nvs.MCPFederationNotes[0].URL)
	require.Equal(t, 1, nvs.MCPFederationNotes[0].MaxDepth)
}
