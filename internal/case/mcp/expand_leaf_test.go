package mcp_test

import (
	"html/template"
	"testing"

	"trip2g/internal/case/mcp"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// The recorded walk that motivates this: an agent descended a note with expand,
// reached "Step 1. Boot the memory server", was told it is a leaf and to call
// note_html with the same arguments, and burned three such round trips before
// running out of budget without having read a line.

func sectionedNote() *appmodel.NoteView {
	return &appmodel.NoteView{
		Path:      "guide.md",
		PathID:    42,
		Title:     "Guide",
		Permalink: "/guide",
		// Titles long enough that expand adds no preview of the section text,
		// so a body word in the summary can only come from the body itself.
		Headings: appmodel.NoteViewHeadings{
			{Text: "Setup and installation", Level: 1, ID: "setup"},
			{Text: "Install the binary", Level: 2, ID: "install"},
			{Text: "Events and hooks", Level: 1, ID: "events"},
		},
		HTML: template.HTML(`<div data-header="Setup and installation" data-level="1"><h1>Setup and installation</h1><p>setup intro</p>` +
			`<div data-header="Install the binary" data-level="2"><h2>Install the binary</h2><p>run the installer</p></div></div>` +
			`<div data-header="Events and hooks" data-level="1"><h1>Events and hooks</h1><p>events body</p></div>`),
	}
}

func TestExpandLeafReturnsTheSection(t *testing.T) {
	tests := []struct {
		name         string
		pointer      bool
		tocPath      string
		wantErr      string
		wantChildren []string
		wantBody     string
	}{
		{
			name:     "leaf equals note_html on the same section",
			tocPath:  `["Events and hooks"]`,
			wantBody: "events body",
		},
		{
			name:     "nested leaf",
			tocPath:  `["Setup and installation","Install the binary"]`,
			wantBody: "run the installer",
		},
		{
			name:     "leaf of a pointer note keeps the federation line",
			pointer:  true,
			tocPath:  `["Events and hooks"]`,
			wantBody: "events body",
		},
		{
			name:         "section with children lists them, not the body",
			tocPath:      `["Setup and installation"]`,
			wantChildren: []string{"Install the binary"},
		},
		{
			name:         "top level is unchanged",
			tocPath:      `[]`,
			wantChildren: []string{"Setup and installation", "Events and hooks"},
		},
		{
			name:    "unknown toc_path fails loud",
			tocPath: `["Nope"]`,
			wantErr: "section not found for toc_path [Nope]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := sectionedNote()
			if tt.pointer {
				note.MCPFederationKBURL = "https://guide.example/_system/mcp"
				note.MCPFederationKBID = "guide"
			}
			env := noteHTMLEnv(note)
			args := `{"path":"guide.md","toc_path":` + tt.tocPath + `}`

			resp := callTool(t, env, "expand", args)

			if tt.wantErr != "" {
				require.NotNil(t, resp.Error)
				require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
				require.Contains(t, resp.Error.Message, tt.wantErr)
				require.Contains(t, resp.Error.Message, "top-level sections: Setup and installation, Events and hooks")
				return
			}
			require.Nil(t, resp.Error)
			result := resp.Result.(mcp.CallToolResult)
			payload := decodePayload[mcp.ExpandPayload](t, result)
			require.Equal(t, int64(42), payload.NoteID)
			require.Equal(t, "guide.md", payload.NotePath)

			if tt.wantBody == "" {
				titles := make([]string, 0, len(payload.Children))
				for _, c := range payload.Children {
					titles = append(titles, c.Title)
				}
				require.Equal(t, tt.wantChildren, titles)
				require.Empty(t, payload.SectionHTML)
				require.NotContains(t, result.Content[0].Text, "setup intro")
				require.NotContains(t, result.Content[0].Text, "run the installer")
				return
			}

			noteHTML := callTool(t, env, "note_html", args)
			require.Nil(t, noteHTML.Error)
			want := noteHTML.Result.(mcp.CallToolResult).Content[0].Text
			require.Contains(t, want, tt.wantBody)
			require.Equal(t, want, result.Content[0].Text)
			require.Empty(t, payload.Children)
			require.Contains(t, payload.SectionHTML, tt.wantBody)
			require.NotContains(t, payload.SectionHTML, "federation pointer")
			if tt.pointer {
				require.Contains(t, result.Content[0].Text, `federation pointer · kb_id: guide`)
			}
		})
	}
}

// A sectionless note has nothing to expand at any level; the top-level answer
// stays a summary that points at a whole-note read.
func TestExpandSectionlessNoteTopLevel(t *testing.T) {
	note := &appmodel.NoteView{Path: "flat.md", PathID: 7, Title: "Flat", HTML: "<p>only a paragraph</p>"}
	env := noteHTMLEnv(note)

	resp := callTool(t, env, "expand", `{"path":"flat.md"}`)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	payload := decodePayload[mcp.ExpandPayload](t, result)
	require.Empty(t, payload.Children)
	require.Empty(t, payload.SectionHTML)
	require.Contains(t, result.Content[0].Text, "no sections")
	require.Contains(t, result.Content[0].Text, "note_html without toc_path")
}
