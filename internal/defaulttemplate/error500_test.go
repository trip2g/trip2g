package defaulttemplate

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// stubEnv is a minimal Env exposing zero-value chrome data.
type stubEnv struct{}

func (stubEnv) UserJSURLs() []string                { return nil }
func (stubEnv) UserLocaleHashes() map[string]string { return nil }
func (stubEnv) UserCSSURLs() []string               { return nil }
func (stubEnv) IsDevMode() bool                     { return false }
func (stubEnv) ActiveHTMLInjections(context.Context) ([]db.HtmlInjection, error) {
	return nil, nil
}

func TestWriteServerError(t *testing.T) {
	const detail = `Jet Runtime Error ("/mesh/index":411): identifier "coalesce" not available`

	t.Run("admin sees escaped detail", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		WriteServerError(ctx, stubEnv{}, ServerErrorParams{Admin: true, Detail: detail})

		require.Equal(t, http.StatusInternalServerError, ctx.Response.StatusCode())
		body := string(ctx.Response.Body())
		require.Contains(t, body, strings.ReplaceAll(detail, `"`, "&quot;"))
		require.Contains(t, body, "coalesce")
		require.NotContains(t, body, `Jet Runtime Error ("`, "detail must be HTML-escaped, not raw")
	})

	t.Run("public gets no detail", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		WriteServerError(ctx, stubEnv{}, ServerErrorParams{Admin: false, Detail: detail})

		require.Equal(t, http.StatusInternalServerError, ctx.Response.StatusCode())
		body := string(ctx.Response.Body())
		require.NotContains(t, body, "coalesce", "public must not see internal error text")
		require.Contains(t, body, "Something went wrong")
	})
}

func TestWriteServerErrorFallbackEscapesDetail(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	writeServerErrorFallback(ctx, ServerErrorParams{Admin: true, Detail: `boom <script>alert(1)</script>`})

	body := string(ctx.Response.Body())
	require.NotContains(t, body, "<script>alert(1)</script>", "detail must be HTML-escaped")
	require.Contains(t, body, "&lt;script&gt;")
	require.Contains(t, body, "Critical server error")
}
