package mcp_test

import (
	"context"
	"html/template"
	"testing"

	"trip2g/internal/case/mcp"
	"trip2g/internal/embedguard"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

const embedSecret = "zzsecretzz"

func embedVault() (*appmodel.NoteView, *appmodel.NoteView) {
	closed := &appmodel.NoteView{
		Path:      "closed.md",
		PathID:    2,
		Title:     "Closed",
		Permalink: "/closed",
		HTML:      template.HTML("<p>" + embedSecret + "</p>"),
	}

	parent := &appmodel.NoteView{
		Path:      "parent.md",
		PathID:    1,
		Title:     "Parent",
		Permalink: "/parent",
		Free:      true,
		HTML:      template.HTML("<p>intro</p>" + embedguard.Wrap(2, string(closed.HTML)) + "<p>outro</p>"),
	}

	return parent, closed
}

func embedEnvFor(parent, closed *appmodel.NoteView, readable func(*appmodel.NoteView) bool) *EnvMock {
	env := noteHTMLEnv(parent)

	env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
		views := appmodel.NewNoteViews()
		views.RegisterNote(parent)
		views.RegisterNote(closed)

		return views
	}
	env.CanReadNoteFunc = func(_ context.Context, note *appmodel.NoteView) (bool, error) {
		return readable(note), nil
	}

	return env
}

func TestNoteHTMLToolStripsAnEmbedTheCallerCannotRead(t *testing.T) {
	parent, closed := embedVault()
	env := embedEnvFor(parent, closed, func(note *appmodel.NoteView) bool { return note.Free })

	resp := callTool(t, env, "note_html", `{"path":"parent.md"}`)

	require.Nil(t, resp.Error)
	text := resp.Result.(mcp.CallToolResult).Content[0].Text
	require.Contains(t, text, "intro")
	require.NotContains(t, text, embedSecret)
	require.Contains(t, text, `<a href="/closed">Closed</a>`)
}

func TestNoteHTMLToolKeepsAnEmbedTheCallerCanRead(t *testing.T) {
	parent, closed := embedVault()
	env := embedEnvFor(parent, closed, func(*appmodel.NoteView) bool { return true })

	resp := callTool(t, env, "note_html", `{"path":"parent.md"}`)

	require.Nil(t, resp.Error)
	text := resp.Result.(mcp.CallToolResult).Content[0].Text
	require.Contains(t, text, embedSecret)
	require.NotContains(t, text, "<!--t2e:")
}

func TestNoteHTMLToolDoesNotChangeTheSharedNote(t *testing.T) {
	parent, closed := embedVault()
	before := string(parent.HTML)
	env := embedEnvFor(parent, closed, func(note *appmodel.NoteView) bool { return note.Free })

	callTool(t, env, "note_html", `{"path":"parent.md"}`)

	require.Equal(t, before, string(parent.HTML))
}
