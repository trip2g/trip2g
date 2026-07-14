package defaulttemplate

import (
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

// makeVisibleNVS is like makeNVS but also populates NoteViews.List, which
// NVS.List() (and thus RSSFeeds) iterates over via VisibleList().
func makeVisibleNVS(notes []*model.NoteView) *templateviews.NVS {
	nvs := model.NewNoteViews()
	for _, nv := range notes {
		nvs.PathMap[nv.Path] = nv
		nvs.Map[nv.Permalink] = nv
		nvs.List = append(nvs.List, nv)
	}
	return templateviews.NewNVS(nvs, "live")
}

func TestRSSFeeds_DiscoversFeedNote(t *testing.T) {
	feedNote := makeNote("feed.md", map[string]interface{}{
		"content_type": "application/rss+xml; charset=utf-8",
		"layout":       "rss",
		"rss_title":    "My site feed",
	})
	feedNote.Permalink = "/feed.xml"
	otherNote := makeNote("blog/post1.md", nil)

	nvs := makeVisibleNVS([]*model.NoteView{feedNote, otherNote})
	ctx := &Ctx{Notes: nvs}

	feeds := ctx.RSSFeeds()
	require.Len(t, feeds, 1)
	require.Equal(t, "My site feed", feeds[0].Title)
	require.Equal(t, "/feed.xml", feeds[0].Href)
}

func TestRSSFeeds_IgnoresNonFeedNotes(t *testing.T) {
	otherNote := makeNote("blog/post1.md", nil)
	nvs := makeVisibleNVS([]*model.NoteView{otherNote})
	ctx := &Ctx{Notes: nvs}

	require.Empty(t, ctx.RSSFeeds())
}

func TestRSSFeeds_NilNotes(t *testing.T) {
	ctx := &Ctx{}
	require.Empty(t, ctx.RSSFeeds())
}
