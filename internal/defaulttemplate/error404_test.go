package defaulttemplate

import (
	"net/http"
	"testing"

	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestWriteNotFound(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	WriteNotFound(ctx, stubEnv{}, &usertoken.Data{ID: 1, Role: "reader"})

	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Header.ContentType()), "text/html")
	body := string(ctx.Response.Body())
	require.Contains(t, body, "Page not found")
}

func TestWriteNotFound_SpeaksTheVisitorsLanguage(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetCookie(langCookieName, "ru")
	WriteNotFound(ctx, stubEnv{}, nil)

	body := string(ctx.Response.Body())
	require.Contains(t, body, "<title>Страница не найдена")
	require.Contains(t, body, `<p class="notfound__message">Страница не найдена</p>`)
	require.Contains(t, body, "← Главная")
	require.NotContains(t, body, "Page not found")
}

func TestWriteNotFound_PassesEditorBundle(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	WriteNotFound(ctx, stubEnv{}, nil)

	body := string(ctx.Response.Body())
	require.Contains(t, body, `editor_js_url: "/assets/ui/editor/pane/-/web.js?h=e1",`)
	require.Contains(t, body, `editor_locale_hashes: {"ru":"r1",},`)
	require.NotContains(t, body, `<script src="/assets/ui/editor/pane/-/web.js`, "the editor bundle loads only inside the editor frame")
}
