package federationhandoff

import (
	"errors"
	"html"
	"net/http"

	"trip2g/internal/appreq"
)

const (
	Path = "/_system/federation/handoff"

	msgRefused = "federation_handoff_refused"
)

// Endpoint answers the KB-note button with a page that posts the signed
// handoff to the peer, the same auto-submitting form HAT links use.
type Endpoint struct{}

func (*Endpoint) Path() string   { return Path }
func (*Endpoint) Method() string { return http.MethodPost }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	handoff, err := Resolve(req.Req, req.Env.(Env), string(req.Req.PostArgs().Peek("note")))
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, ErrRefused) {
			code = http.StatusForbidden
		}
		return nil, &appreq.SystemMessageError{Code: code, Msg: msgRefused, Err: err}
	}

	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetContentType("text/html; charset=utf-8")
	req.Req.Response.Header.Set("Cache-Control", "no-store")
	// Not no-referrer: under it a browser sends the form's POST with Origin:
	// null, and the peer accepts the handoff only from this site's origin.
	req.Req.Response.Header.Set("Referrer-Policy", "strict-origin")
	req.Req.SetBodyString(`<!doctype html>
<html><head><meta charset="utf-8"><title>…</title></head>
<body>
<form id="handoff" method="post" action="` + html.EscapeString(handoff.Action) + `">
<input type="hidden" name="token" value="` + html.EscapeString(handoff.Token) + `">
<input type="hidden" name="path" value="` + html.EscapeString(handoff.Path) + `">
<noscript><button type="submit">Continue</button></noscript>
</form>
<script>document.getElementById("handoff").submit()</script>
</body></html>`)
	return nil, nil
}
