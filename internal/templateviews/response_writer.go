package templateviews

import "github.com/valyala/fasthttp"

// ResponseWriter lets a Jet layout control the HTTP response Content-Type via
// {{ response.SetContentType("application/json") }}. Exposed to layouts as the
// `response` variable so a template can emit JSON, RSS, XML, CSV, etc. instead
// of the default text/html.
type ResponseWriter struct {
	Ctx *fasthttp.RequestCtx
}

// SetContentType sets the response Content-Type and returns "" so the call
// renders nothing in the template body.
func (rw *ResponseWriter) SetContentType(ct string) string {
	if rw.Ctx == nil {
		return ""
	}
	rw.Ctx.SetContentType(ct)
	return ""
}
