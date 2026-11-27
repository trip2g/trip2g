package rendernotepage_test

import (
	"context"
	"sort"
	"testing"

	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

const (
	secretText = "zzsecretzz"
	openText   = "zzopentextzz"
)

func loadVault(t *testing.T, files map[string]string) *model.NoteViews {
	t.Helper()

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	sources := make([]mdloader.SourceFile, 0, len(files))
	for i, path := range paths {
		sources = append(sources, mdloader.SourceFile{
			Path:      path,
			PathID:    int64(i + 1),
			VersionID: int64(i + 1),
			Content:   []byte(files[path]),
		})
	}

	views, err := mdloader.Load(mdloader.Options{Log: &logger.TestLogger{}, Sources: sources})
	require.NoError(t, err)

	views.Version = "live"

	return views
}

type viewerKind int

const (
	viewerAnonymous viewerKind = iota
	viewerMember
)

func embedEnv(t *testing.T, files map[string]string, layoutSources []model.LayoutSourceFile) (*EnvMock, *viewerKind) {
	t.Helper()

	views := loadVault(t, files)

	layouts, err := layoutloader.Load(cacheLoaderEnv{}, layoutSources, layoutloader.Options{})
	require.NoError(t, err)

	env, _, _ := cacheTestEnv(views, layouts)

	viewer := new(viewerKind)

	env.CanReadNoteFunc = func(ctx context.Context, note *model.NoteView) (bool, error) {
		if *viewer == viewerMember {
			return true, nil
		}

		return note.IsAnonymouslyReadable(), nil
	}

	return env, viewer
}

func pageBody(t *testing.T, env *EnvMock, path string, token *usertoken.Data) string {
	t.Helper()

	ctx := newReqCtx(reqOpts{path: path})
	runHandle(t, env, ctx, token)

	return string(body(t, ctx))
}

func memberToken() *usertoken.Data {
	return &usertoken.Data{ID: 7, Role: "user"}
}

func TestEmbedAccess_PartialDoesNotLaunderClosedContentIntoAnonymousPage(t *testing.T) {
	env, _ := embedEnv(t, map[string]string{
		"free.md":     "---\nfree: true\n---\n" + openText + " ![[_partial]]\n",
		"_partial.md": "partial-text ![[closed]]\n",
		"closed.md":   secretText + "\n",
	}, nil)

	page := pageBody(t, env, "/free", nil)

	require.Contains(t, page, openText)
	require.Contains(t, page, "partial-text")
	require.NotContains(t, page, secretText)
	require.Contains(t, page, `href="/closed"`)
}

func TestEmbedAccess_MemberStillSeesAllowedEmbedsInline(t *testing.T) {
	env, viewer := embedEnv(t, map[string]string{
		"free.md":     "---\nfree: true\n---\n" + openText + " ![[_partial]]\n",
		"_partial.md": "partial-text ![[closed]]\n",
		"closed.md":   secretText + "\n",
	}, nil)

	*viewer = viewerMember

	page := pageBody(t, env, "/free", memberToken())

	require.Contains(t, page, "partial-text")
	require.Contains(t, page, secretText)
	require.NotContains(t, page, "<!--t2e:")
}

func TestEmbedAccess_DefaultTemplateContentReferenceNeedsReadAccess(t *testing.T) {
	files := map[string]string{
		"home.md":   "---\nfree: true\ncontent: \"[[closed]]\"\n---\nbody\n",
		"closed.md": secretText + "\n",
	}

	env, viewer := embedEnv(t, files, nil)

	require.NotContains(t, pageBody(t, env, "/home", nil), secretText)

	*viewer = viewerMember
	require.Contains(t, pageBody(t, env, "/home", memberToken()), secretText)
}

func TestEmbedAccess_SidebarContentWidgetNeedsReadAccess(t *testing.T) {
	env, viewer := embedEnv(t, map[string]string{
		"home.md":   "---\nfree: true\nright_sidebar: \"[[closed]]\"\n---\nbody\n",
		"closed.md": secretText + "\n",
	}, nil)

	require.NotContains(t, pageBody(t, env, "/home", nil), secretText)

	*viewer = viewerMember
	require.Contains(t, pageBody(t, env, "/home", memberToken()), secretText)
}

func TestEmbedAccess_CustomLayoutIncludeNeedsReadAccess(t *testing.T) {
	env, viewer := embedEnv(t, map[string]string{
		"home.md":   "---\nfree: true\nlayout: inc\n---\nbody\n",
		"closed.md": secretText + "\n",
	}, []model.LayoutSourceFile{
		{ID: "/inc", Path: "_layouts/inc.html", Content: `<main>{{ nvs.ByPath("closed.md").HTMLString() | unsafe }}</main>`},
	})

	require.NotContains(t, pageBody(t, env, "/home", nil), secretText)

	*viewer = viewerMember
	require.Contains(t, pageBody(t, env, "/home", memberToken()), secretText)
}

func TestEmbedAccess_AnonymousCacheNeverCarriesAMemberVariant(t *testing.T) {
	env, viewer := embedEnv(t, map[string]string{
		"free.md":     "---\nfree: true\n---\n" + openText + " ![[_partial]]\n",
		"_partial.md": "partial-text ![[closed]]\n",
		"closed.md":   secretText + "\n",
	}, nil)

	gzipPage := func(token *usertoken.Data) string {
		ctx := newReqCtx(reqOpts{path: "/free", acceptEncoding: "gzip"})
		runHandle(t, env, ctx, token)

		return string(body(t, ctx))
	}

	first := gzipPage(nil)
	require.NotContains(t, first, secretText)

	*viewer = viewerMember
	member := gzipPage(memberToken())
	require.Contains(t, member, secretText)

	*viewer = viewerAnonymous
	again := gzipPage(nil)
	require.NotContains(t, again, secretText)
	require.Equal(t, first, again)
}

func TestEmbedAccess_MemberFirstDoesNotPoisonTheAnonymousCache(t *testing.T) {
	env, viewer := embedEnv(t, map[string]string{
		"free.md":     "---\nfree: true\n---\n" + openText + " ![[_partial]]\n",
		"_partial.md": "partial-text ![[closed]]\n",
		"closed.md":   secretText + "\n",
	}, nil)

	*viewer = viewerMember
	ctx := newReqCtx(reqOpts{path: "/free", acceptEncoding: "gzip"})
	runHandle(t, env, ctx, memberToken())
	require.Contains(t, string(body(t, ctx)), secretText)

	*viewer = viewerAnonymous
	ctx = newReqCtx(reqOpts{path: "/free", acceptEncoding: "gzip"})
	runHandle(t, env, ctx, nil)
	require.NotContains(t, string(body(t, ctx)), secretText)
}

func TestEmbedAccess_MetaDescriptionNeverQuotesAClosedEmbed(t *testing.T) {
	env, _ := embedEnv(t, map[string]string{
		"free.md":     "---\nfree: true\n---\n![[_partial]]\n",
		"_partial.md": "partial-text ![[closed]]\n",
		"closed.md":   secretText + "\n",
	}, nil)

	require.NotContains(t, pageBody(t, env, "/free", nil), secretText)
}
