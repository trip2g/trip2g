package rendernotepage

import (
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// ogTestEnv embeds a nil Env and overrides only the method buildOGTags/
// ogURLForNote actually call (PublicURL) — safe as long as the test scenario
// never reaches an unoverridden method.
type ogTestEnv struct {
	Env
	publicURL string
}

func (e ogTestEnv) PublicURL() string { return e.publicURL }

func ogTestRequest(host string) *appreq.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/post")
	ctx.Request.Header.SetHost(host)
	return &appreq.Request{Req: ctx}
}

// TestBuildOGTags_ImageURLsAreAbsolute pins the fix for broken link
// unfurling: note asset URLs are site-relative (see model.NoteAssetURLPath),
// but og:image is fetched by external crawlers (Slack, Telegram, Twitter,
// ...) from outside trip2g's origin, so it must be absolute — same as og:url.
func TestBuildOGTags_ImageURLsAreAbsolute(t *testing.T) {
	env := ogTestEnv{publicURL: "https://example.com"}
	req := ogTestRequest("example.com")

	t.Run("explicit og_image via frontmatter asset", func(t *testing.T) {
		note := &model.NoteView{
			Permalink: "/post",
			RawMeta:   map[string]interface{}{"og_image": "cover.png"},
			AssetReplaces: map[string]*model.NoteAssetReplace{
				"cover.png": {Hash: "abc", FileName: "cover.png", URL: model.NoteAssetURLPath("abc", "cover.png")},
			},
		}
		resp := &Response{Note: note}

		tags := buildOGTags(req, env, resp)

		require.Equal(t, "https://example.com/_system/assets/abc/cover.png", tags["og:image"])
	})

	t.Run("fallback to first body image", func(t *testing.T) {
		firstImage := "photo.png"
		note := &model.NoteView{
			Permalink:  "/post",
			FirstImage: &firstImage,
			AssetReplaces: map[string]*model.NoteAssetReplace{
				"photo.png": {Hash: "def", FileName: "photo.png", URL: model.NoteAssetURLPath("def", "photo.png")},
			},
		}
		resp := &Response{Note: note}

		tags := buildOGTags(req, env, resp)

		require.Equal(t, "https://example.com/_system/assets/def/photo.png", tags["og:image"])
	})

	t.Run("no image means no og:image tag", func(t *testing.T) {
		note := &model.NoteView{Permalink: "/post"}
		resp := &Response{Note: note}

		tags := buildOGTags(req, env, resp)

		_, ok := tags["og:image"]
		require.False(t, ok)
	})
}
