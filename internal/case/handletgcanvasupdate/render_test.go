package handletgcanvasupdate

import (
	"testing"
	"trip2g/internal/model"
	"trip2g/internal/obsidiancanvas"

	"github.com/stretchr/testify/require"
)

func TestRenderNode_TextNode(t *testing.T) {
	raw := `{
		"nodes":[{"id":"n1","type":"text","text":"Hello **world**"}],
		"edges":[]
	}`
	canvas, err := obsidiancanvas.Parse([]byte(raw))
	require.NoError(t, err)

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return model.NewNoteViews() },
		BotIDFunc:           func() int64 { return 1 },
	}

	text, media, markup := renderNode(env, canvas, "n1")
	require.Contains(t, text, "<b>world</b>")
	require.Equal(t, "", media)
	require.Contains(t, markup, "nav:back")
	require.Contains(t, markup, "nav:exit")
}

func TestRenderNode_FileNode(t *testing.T) {
	nvs := model.NewNoteViews()
	nv := &model.NoteView{
		Path:    "docs/test.md",
		Title:   "Test",
		Content: []byte("# Test\nBody"),
	}
	nvs.PathMap["docs/test.md"] = nv

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return nvs },
		BotIDFunc:           func() int64 { return 1 },
		RenderNoteHTMLFunc: func(nv *model.NoteView) (string, string) {
			return "<b>Test</b>\nBody", ""
		},
	}

	raw := `{
		"nodes":[{"id":"f1","type":"file","file":"docs/test.md"}],
		"edges":[]
	}`
	canvas, err := obsidiancanvas.Parse([]byte(raw))
	require.NoError(t, err)

	text, media, _ := renderNode(env, canvas, "f1")
	require.Equal(t, "<b>Test</b>\nBody", text)
	require.Equal(t, "", media)
}

func TestRenderNode_FileNodeWithMedia(t *testing.T) {
	nvs := model.NewNoteViews()
	nv := &model.NoteView{
		Path:    "note.md",
		Title:   "Note",
		Content: []byte("Body"),
	}
	nvs.PathMap["note.md"] = nv

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return nvs },
		BotIDFunc:           func() int64 { return 1 },
		RenderNoteHTMLFunc: func(nv *model.NoteView) (string, string) {
			return "Caption text", "https://cdn.example.com/photo.jpg"
		},
	}

	raw := `{
		"nodes":[{"id":"f1","type":"file","file":"note.md"}],
		"edges":[]
	}`
	canvas, err := obsidiancanvas.Parse([]byte(raw))
	require.NoError(t, err)

	text, media, _ := renderNode(env, canvas, "f1")
	require.Equal(t, "Caption text", text)
	require.Equal(t, "https://cdn.example.com/photo.jpg", media)
}

func TestRenderNode_LinkNode(t *testing.T) {
	raw := `{
		"nodes":[{"id":"l1","type":"link","url":"https://example.com"}],
		"edges":[]
	}`
	canvas, err := obsidiancanvas.Parse([]byte(raw))
	require.NoError(t, err)

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return model.NewNoteViews() },
		BotIDFunc:           func() int64 { return 1 },
	}

	text, _, _ := renderNode(env, canvas, "l1")
	require.Contains(t, text, `href="https://example.com"`)
}

func TestRenderNode_WithEdges(t *testing.T) {
	raw := `{
		"nodes":[
			{"id":"a","type":"text","text":"Node A"},
			{"id":"b","type":"text","text":"Node B"}
		],
		"edges":[{"id":"e1","fromNode":"a","toNode":"b","label":"Go to B"}]
	}`
	canvas, err := obsidiancanvas.Parse([]byte(raw))
	require.NoError(t, err)

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return model.NewNoteViews() },
		BotIDFunc:           func() int64 { return 1 },
	}

	_, _, markup := renderNode(env, canvas, "a")
	require.Contains(t, markup, "Go to B")
	require.Contains(t, markup, "nav:open:b")
}

func TestRenderNode_NotFound(t *testing.T) {
	raw := `{"nodes":[],"edges":[]}`
	canvas, err := obsidiancanvas.Parse([]byte(raw))
	require.NoError(t, err)

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return model.NewNoteViews() },
		BotIDFunc:           func() int64 { return 1 },
	}

	text, _, _ := renderNode(env, canvas, "nonexistent")
	require.Equal(t, "(node not found)", text)
}
