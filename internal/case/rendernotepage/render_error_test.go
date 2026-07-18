package rendernotepage

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// serverErrEnv is a minimal Env stub exposing only the methods
// defaulttemplate.WriteServerError touches. Embedding the interface satisfies
// the full Env contract; any unstubbed method panics if called (none should be).
type serverErrEnv struct {
	Env
}

func (serverErrEnv) IsDevMode() bool { return false }
func (serverErrEnv) ActiveHTMLInjections(context.Context) ([]db.HtmlInjection, error) {
	return nil, nil
}
func (serverErrEnv) UserJSURLs() []string                { return nil }
func (serverErrEnv) UserLocaleHashes() map[string]string { return nil }
func (serverErrEnv) UserCSSURLs() []string               { return nil }

func TestWriteLayoutRenderError(t *testing.T) {
	const layoutName = "mesh/index"
	const errMsg = `Jet Runtime Error ("/mesh/index":411): identifier "coalesce" not available`

	tests := []struct {
		name        string
		token       *usertoken.Data
		wantErrText bool
	}{
		{name: "admin sees error text", token: &usertoken.Data{ID: 1, Role: "admin"}, wantErrText: true},
		{name: "public gets generic page", token: &usertoken.Data{ID: 2, Role: "reader"}, wantErrText: false},
		{name: "anonymous gets generic page", token: nil, wantErrText: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			// Simulate partial layout output already staged on the response to
			// prove the error path discards it rather than leaking it.
			ctx.SetBodyString(`<video poster="`)

			resp := &Response{UserToken: tt.token}
			writeLayoutRenderError(ctx, serverErrEnv{}, resp, layoutName, errMsg)

			require.Equal(t, http.StatusInternalServerError, ctx.Response.StatusCode())
			body := string(ctx.Response.Body())
			require.NotContains(t, body, `<video poster="`, "partial layout output must not leak")

			if tt.wantErrText {
				// The detail is rendered through quicktemplate's own HTML escaping
				// (which quotes as &quot;, unlike html.EscapeString's &#34;).
				require.Contains(t, body, strings.ReplaceAll(errMsg, `"`, "&quot;"), "admin must see the (escaped) error text")
				require.Contains(t, body, "coalesce", "admin must see the failing identifier")
				require.Contains(t, body, layoutName)
			} else {
				require.NotContains(t, body, errMsg, "public must not see internal error text")
				require.NotContains(t, body, "coalesce", "public must not see internal error text")
				require.Contains(t, body, "Something went wrong", "public gets the generic 500 page")
			}
		})
	}
}

// renderLayoutEnv is a minimal Env for driving renderLayout end-to-end with a
// real failing Jet layout. Only the methods renderLayout touches are stubbed.
type renderLayoutEnv struct {
	Env
	layouts *model.Layouts
}

func (e renderLayoutEnv) Layouts() *model.Layouts { return e.layouts }
func (renderLayoutEnv) Logger() logger.Logger     { return &logger.DummyLogger{} }
func (renderLayoutEnv) PublicURL() string         { return "https://example.test" }
func (renderLayoutEnv) IsDevMode() bool           { return true }
func (renderLayoutEnv) ActiveHTMLInjections(context.Context) ([]db.HtmlInjection, error) {
	return nil, nil
}
func (renderLayoutEnv) UserJSURLs() []string                { return nil }
func (renderLayoutEnv) UserLocaleHashes() map[string]string { return nil }
func (renderLayoutEnv) UserCSSURLs() []string               { return nil }

// TestRenderLayoutExecuteFailure drives the real renderLayout path with a Jet
// layout that fails mid-render, proving the buffer -> error-page wiring: no
// partial output leaks, status is 500, admins see the error, the public does not.
func TestRenderLayoutExecuteFailure(t *testing.T) {
	loader := jet.NewInMemLoader()
	// "pre " renders, then coalesce (an undefined identifier) fails mid-render.
	loader.Set("/broken", `pre <video poster="{{ coalesce(1) }}"> post`)
	set := jet.NewSet(loader, jet.DevelopmentMode(true), jet.WithSafeWriter(nil))
	view, err := set.GetTemplate("/broken")
	require.NoError(t, err)

	env := renderLayoutEnv{layouts: &model.Layouts{
		Map: map[string]model.Layout{"/broken": {Path: "/broken", View: view}},
	}}

	tests := []struct {
		name        string
		token       *usertoken.Data
		wantErrText bool
	}{
		{name: "admin", token: &usertoken.Data{ID: 1, Role: "admin"}, wantErrText: true},
		{name: "public", token: &usertoken.Data{ID: 2, Role: "reader"}, wantErrText: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			resp := &Response{UserToken: tt.token}

			processed, rerr := renderLayout(ctx, env, resp, "broken")
			require.NoError(t, rerr)
			require.True(t, processed)

			require.Equal(t, http.StatusInternalServerError, ctx.Response.StatusCode())
			body := string(ctx.Response.Body())
			require.NotContains(t, body, `<video poster="`, "partial layout output must not leak")

			if tt.wantErrText {
				require.Contains(t, body, "coalesce", "admin must see the failing identifier")
			} else {
				require.NotContains(t, body, "coalesce", "public must not see internal error text")
				require.Contains(t, body, "Something went wrong")
			}
		})
	}
}
