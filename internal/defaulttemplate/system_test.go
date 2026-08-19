package defaulttemplate

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// The system page reads its copy from the translation bundle, which the server
// loads once at startup.
func TestMain(m *testing.M) {
	if err := Init(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestWriteSystemMessage_TellsTheVisitorWhatToDo(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	WriteSystemMessage(ctx, http.StatusUnauthorized, "hat_expired")

	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Header.ContentType()), "text/html")

	body := string(ctx.Response.Body())
	require.Contains(t, body, "This link has expired")
	require.Contains(t, body, "Ask for a new one")
	require.Contains(t, body, `<html lang="en">`)
}

func TestWriteSystemMessage_SpeaksTheVisitorsLanguage(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetCookie(langCookieName, "ru")

	WriteSystemMessage(ctx, http.StatusUnauthorized, "hat_expired")

	body := string(ctx.Response.Body())
	require.Contains(t, body, "Ссылка устарела")
	require.Contains(t, body, `<html lang="ru">`)
	require.Contains(t, body, "← Главная")
}

// A key the bundle does not know must never reach the page as a key.
func TestWriteSystemMessage_UnknownKeyFallsBackToAnApology(t *testing.T) {
	for _, msg := range []string{"", "hat_typo"} {
		ctx := &fasthttp.RequestCtx{}
		WriteSystemMessage(ctx, http.StatusInternalServerError, msg)

		body := string(ctx.Response.Body())
		require.Contains(t, body, "Something went wrong")
		require.NotContains(t, body, "_title")
		require.NotContains(t, body, "hat_typo")
	}
}

// The page has to stand on its own: the requests that need it are the ones that
// never got as far as loading bundles or note chrome.
func TestWriteSystemMessage_CarriesNoScriptsOrBundles(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	WriteSystemMessage(ctx, http.StatusUnauthorized, "hat_no_account")

	body := string(ctx.Response.Body())
	require.NotContains(t, body, "<script")
	require.NotContains(t, body, "mol_view_root")
	require.NotContains(t, body, "stylesheet")
}

func TestWriteSystemMessage_ReplacesAnythingAlreadyWritten(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.SetBodyString("half a response")

	WriteSystemMessage(ctx, http.StatusBadRequest, "hat_invalid")

	body := string(ctx.Response.Body())
	require.NotContains(t, body, "half a response")
	require.Equal(t, http.StatusBadRequest, ctx.Response.StatusCode())
}
