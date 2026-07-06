package renderadminpage_test

import (
	"net/http"
	"strings"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/renderadminpage"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type stubEnv struct {
	views *model.NoteViews
}

func (stubEnv) AdminJSURL() string { return "/assets/ui/admin/-/web.js?h=deadbeef" }

func (e stubEnv) LiveNoteViews() *model.NoteViews { return e.views }

func newRequest(env renderadminpage.Env) *appreq.Request {
	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)

	return &appreq.Request{
		Env: env,
		Req: fctx,
	}
}

func TestCanonicalSystemAdminServesShell(t *testing.T) {
	ep := renderadminpage.GetEndpoint{}
	require.Equal(t, "/_system/admin", ep.Path())
	require.Equal(t, http.MethodGet, ep.Method())

	req := newRequest(stubEnv{})

	resp, err := ep.Handle(req)
	require.NoError(t, err)
	require.Nil(t, resp)

	require.Equal(t, http.StatusOK, req.Req.Response.StatusCode())

	body := string(req.Req.Response.Body())
	require.Contains(t, body, `mol_view_root="$trip2g_admin"`)
	require.Contains(t, body, "/assets/ui/admin/-/web.js?h=deadbeef")
	require.True(t, strings.HasPrefix(string(req.Req.Response.Header.ContentType()), "text/html"))
}

func TestLegacyAdminServesShellWithoutNote(t *testing.T) {
	ep := renderadminpage.Endpoint{}
	require.Equal(t, "/admin", ep.Path())
	require.Equal(t, http.MethodGet, ep.Method())

	// No live note at /admin: the legacy path keeps serving the admin shell,
	// with no server-side token gate — auth is handled by the shell itself and
	// per-query GraphQL authorization.
	req := newRequest(stubEnv{views: &model.NoteViews{Map: map[string]*model.NoteView{}}})

	resp, err := ep.Handle(req)
	require.NoError(t, err)
	require.Nil(t, resp)

	require.Equal(t, http.StatusOK, req.Req.Response.StatusCode())
	require.Contains(t, string(req.Req.Response.Body()), `mol_view_root="$trip2g_admin"`)
}
