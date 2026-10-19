package defaulttemplate

import (
	"strconv"
	"testing"

	"trip2g/internal/model"
)

func BenchmarkRSSFeeds(b *testing.B) {
	notes := make([]*model.NoteView, 0, 10001)
	for i := range 10000 {
		notes = append(notes, makeNote("notes/n"+strconv.Itoa(i)+".md", map[string]interface{}{"title": "Note"}))
	}
	feedNote := makeNote("feed.md", map[string]interface{}{"content_type": "application/rss+xml", "rss_title": "Feed"})
	feedNote.Permalink = "/feed.xml"
	notes = append(notes, feedNote)

	ctx := &Ctx{Notes: makeVisibleNVS(notes)}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if len(ctx.RSSFeeds()) != 1 {
			b.Fatal("expected one feed")
		}
	}
}
