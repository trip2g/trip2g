package renderadminpage

import (
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/rendernotepage"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=.

// renderPage serves the admin app shell (HTML + admin JS bundle). The shell is
// not a secret: for guests the bundle renders the auth form ($trip2g_admin
// extends $trip2g_auth) and every admin query is authorized server-side in the
// GraphQL layer, the same way /graphql serves the playground to everyone while
// the queries themselves stay gated.
func renderPage(req *appreq.Request) (interface{}, error) {
	ctx := req.Req

	resp, err := Resolve(ctx, req.Env.(Env), Request{})
	if err != nil {
		return nil, err
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	WritePage(ctx, resp)

	return nil, nil
}

// Endpoint serves the legacy /admin path. It keeps working for backward
// compatibility until a live note at /admin replaces it — then the note takes
// over the path and the admin app remains at its canonical /_system/admin.
type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	if nvs := req.Env.(Env).LiveNoteViews(); nvs != nil && nvs.GetByPath("/admin") != nil {
		return rendernotepage.Endpoint{}.Handle(req)
	}

	return renderPage(req)
}

func (Endpoint) Path() string {
	return "/admin"
}

func (Endpoint) Method() string {
	return http.MethodGet
}

// GetEndpoint serves the canonical /_system/admin path, mirroring
// /_system/graphql: always mounted, part of the /_system namespace that is the
// single home for system endpoints.
type GetEndpoint struct{}

func (e GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	return renderPage(req)
}

func (GetEndpoint) Path() string {
	return "/_system/admin"
}

func (GetEndpoint) Method() string {
	return http.MethodGet
}
