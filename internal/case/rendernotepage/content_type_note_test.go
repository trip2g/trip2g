package rendernotepage

import (
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestHandleContentTypeNote_MissingLayoutServesRawBody(t *testing.T) {
	note := &model.NoteView{
		Path:    "feed.md",
		Layout:  "rss",
		Content: []byte("---\ncontent_type: application/rss+xml\nlayout: rss\n---\nfeed body\n"),
		RawMeta: map[string]interface{}{"content_type": "application/rss+xml"},
	}
	resp := &Response{Note: note, NoteView: templateviews.NewNote(note)}
	env := renderLayoutEnv{layouts: &model.Layouts{Map: map[string]model.Layout{}}}
	ctx := &fasthttp.RequestCtx{}

	handled, err := handleContentTypeNote(ctx, env, resp, "rss")

	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, "application/rss+xml", string(ctx.Response.Header.ContentType()))
	require.Equal(t, "feed body\n", string(ctx.Response.Body()))
}
