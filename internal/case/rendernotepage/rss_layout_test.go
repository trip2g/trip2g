package rendernotepage

import (
	"bytes"
	"encoding/xml"
	"html/template"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// shippedRSSLayoutPath points at the real vault layout so this test breaks if the
// shipped file drifts from what the feature relies on.
const shippedRSSLayoutPath = "../../../onboarding-vault/_layouts/rss.html"

type rssXMLItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
	Encoded string `xml:"encoded"` // content:encoded (matched by local name)
}

type rssXMLFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title       string       `xml:"title"`
		Link        string       `xml:"link"`
		Description string       `xml:"description"`
		Items       []rssXMLItem `xml:"item"`
	} `xml:"channel"`
}

func strptr(s string) *string { return &s }

// TestRSSLayout_RendersValidFeedAndHonorsAccessGates renders the shipped RSS
// layout against a mixed set of notes and asserts the output is valid RSS 2.0
// that never leaks paywalled, sign-in-gated, or system notes.
func TestRSSLayout_RendersValidFeedAndHonorsAccessGates(t *testing.T) {
	layoutBytes, err := os.ReadFile(shippedRSSLayoutPath)
	require.NoError(t, err, "shipped RSS layout must exist on disk")

	const publicURL = "https://example.com"
	created := time.Date(2024, 3, 2, 10, 30, 0, 0, time.UTC)
	freeBody := `<p>Hello ]]> world <img src="/_system/assets/x.png"></p>`

	nvs := model.NewNoteViews()
	nvs.PathMap["blog/free.md"] = &model.NoteView{
		Path: "blog/free.md", Title: "Free Note", Permalink: "/blog/free",
		Free: true, CreatedAt: created, Description: strptr("A free note"),
		HTML: template.HTML(freeBody),
	}
	nvs.PathMap["blog/paid.md"] = &model.NoteView{
		Path: "blog/paid.md", Title: "Paid Note", Permalink: "/blog/paid",
		Free: false, CreatedAt: created,
	}
	nvs.PathMap["blog/gated.md"] = &model.NoteView{
		Path: "blog/gated.md", Title: "Gated Note", Permalink: "/blog/gated",
		Free: true, CreatedAt: created,
		Subgraphs: map[string]*model.NoteSubgraph{"members": {RequireSignin: true}},
	}
	nvs.PathMap["blog/_secret.md"] = &model.NoteView{
		Path: "blog/_secret.md", Title: "System Note", Permalink: "/blog/_secret",
		Free: true, CreatedAt: created,
	}

	feedNV := &model.NoteView{
		Path: "feed.md", Title: "Feed", Permalink: "/feed.xml", Free: true,
		RawMeta: map[string]interface{}{
			"rss_title":       "My Feed",
			"rss_description": "Latest posts",
			"rss_glob":        "blog/*.md",
			"rss_limit":       20,
		},
	}

	sources := []model.LayoutSourceFile{{
		ID:      "/rss",
		Path:    "_layouts/rss.html",
		Content: string(layoutBytes),
	}}
	env := &loaderTestEnv{log: &logger.TestLogger{}}
	layouts, err := layoutloader.Load(env, sources, layoutloader.Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(feedNV))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(nvs, "live"))
	vars["publicURL"] = reflect.ValueOf(publicURL)

	var buf bytes.Buffer
	require.NoError(t, layouts.Map["/rss"].View.Execute(&buf, vars, nil))
	out := buf.String()

	// Valid XML / RSS 2.0 shape.
	var feed rssXMLFeed
	require.NoError(t, xml.Unmarshal(buf.Bytes(), &feed), "output must be valid XML")
	require.Equal(t, "My Feed", feed.Channel.Title)
	require.Equal(t, publicURL+"/feed.xml", feed.Channel.Link)
	require.Equal(t, "Latest posts", feed.Channel.Description)
	require.Len(t, feed.Channel.Items, 1, "only the free note is exposed")

	item := feed.Channel.Items[0]
	require.Equal(t, "Free Note", item.Title)
	require.Equal(t, publicURL+"/blog/free", item.Link)
	require.Equal(t, publicURL+"/blog/free", item.GUID)
	require.True(t, strings.HasPrefix(item.Link, publicURL), "item link must be absolute")

	_, err = time.Parse(time.RFC1123Z, item.PubDate)
	require.NoError(t, err, "pubDate must be RFC1123Z")

	// Body survives html()-encoding round-trip, including the ]]> and the asset URL.
	require.Contains(t, item.Encoded, "]]>")
	require.Contains(t, item.Encoded, "/_system/assets/x.png")

	// Access gates: none of the private notes leak anywhere in the output.
	require.NotContains(t, out, "Paid Note")
	require.NotContains(t, out, "/blog/paid")
	require.NotContains(t, out, "Gated Note")
	require.NotContains(t, out, "/blog/gated")
	require.NotContains(t, out, "System Note")
	require.NotContains(t, out, "/blog/_secret")
}
