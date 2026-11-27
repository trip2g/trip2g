package rendernotepage_test

import (
	"context"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestClosedPageMetaNeverNamesTheFirstBodyImage(t *testing.T) {
	note, views := cacheTestNote()
	note.Free = false
	firstImage := "photo.png"
	note.FirstImage = &firstImage
	note.AssetReplaces = map[string]*model.NoteAssetReplace{
		"photo.png": {Hash: "zzhashzz", FileName: "photo.png", URL: model.NoteAssetURLPath("zzhashzz", "photo.png")},
	}
	env, _, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{})
	runHandle(t, env, ctx, nil)

	html := string(body(t, ctx))
	require.NotContains(t, html, "zzhashzz")
	require.Contains(t, html, note.Title)
}

func TestOpenPageMetaStillNamesTheFirstBodyImage(t *testing.T) {
	note, views := cacheTestNote()
	firstImage := "photo.png"
	note.FirstImage = &firstImage
	note.AssetReplaces = map[string]*model.NoteAssetReplace{
		"photo.png": {Hash: "zzhashzz", FileName: "photo.png", URL: model.NoteAssetURLPath("zzhashzz", "photo.png")},
	}
	env, _, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{})
	runHandle(t, env, ctx, nil)

	require.Contains(t, string(body(t, ctx)), `og:image`)
}

func TestHrefLangsListOnlyAlternativesTheViewerCanRead(t *testing.T) {
	en, views := cacheTestNote()
	en.Lang = "en"
	ru := &model.NoteView{Path: "/ru-note", Title: "Ru", PathID: 2, VersionID: 43, Permalink: "/ru-note", Lang: "ru", Free: false, HTML: "<p>ru</p>"}
	group := &model.LangGroup{
		Hub: en,
		Versions: []model.LangRedirect{
			{Lang: "en", Note: en, URL: "/test-note"},
			{Lang: "ru", Note: ru, URL: "/ru-note"},
		},
	}
	en.LangGroup = group
	ru.LangGroup = group
	views.Map["/ru-note"] = ru
	views.List = append(views.List, ru)

	env, _, _ := cacheTestEnv(views, nil)
	env.CanReadNoteFunc = func(_ context.Context, n *model.NoteView) (bool, error) {
		return n.IsAnonymouslyReadable(), nil
	}

	ctx := newReqCtx(reqOpts{})
	runHandle(t, env, ctx, nil)
	anonymous := string(body(t, ctx))

	require.NotContains(t, anonymous, "/ru-note")

	env.CanReadNoteFunc = func(context.Context, *model.NoteView) (bool, error) { return true, nil }
	ctx = newReqCtx(reqOpts{})
	runHandle(t, env, ctx, memberToken())

	require.Contains(t, string(body(t, ctx)), `hreflang="ru"`)
}
