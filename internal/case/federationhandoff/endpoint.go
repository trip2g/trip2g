package federationhandoff

import (
	"errors"
	"html"
	"net/http"
	"net/url"

	"trip2g/internal/appreq"
	"trip2g/internal/case/rendernotepage"
)

const (
	Path = "/_system/federation/handoff"
	// OpenPath is the link a person is given: it shows where it leads and posts
	// the handoff on a button, never on its own.
	OpenPath = "/_system/federation/open"

	msgRefused = "federation_handoff_refused"
)

// Endpoint answers the KB-note button with a page that posts the signed
// handoff to the peer, the same auto-submitting form HAT links use.
type Endpoint struct{}

func (*Endpoint) Path() string   { return Path }
func (*Endpoint) Method() string { return http.MethodPost }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	handoff, err := Resolve(req.Req, req.Env.(Env), string(req.Req.PostArgs().Peek("note")), string(req.Req.PostArgs().Peek("path")))
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

// GetEndpoint is the page behind an open link.
type GetEndpoint struct{}

func (*GetEndpoint) Path() string   { return OpenPath }
func (*GetEndpoint) Method() string { return http.MethodGet }

func (*GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	notePath := string(req.Req.QueryArgs().Peek("note"))

	offer, err := Open(req.Req, env, notePath, string(req.Req.QueryArgs().Peek("path")))
	if errors.Is(err, ErrAnonymous) {
		return nil, signInFirst(req, env, notePath, err)
	}
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
	req.Req.Response.Header.Set("Content-Security-Policy", "frame-ancestors 'self'")
	req.Req.SetBodyString(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>` + html.EscapeString(offer.Site) + `</title></head>
<body>
<form method="post" action="` + Path + `">
<p>` + html.EscapeString(siteName(offer.Site)) + `</p>
<input type="hidden" name="note" value="` + html.EscapeString(offer.Note) + `">
<input type="hidden" name="path" value="` + html.EscapeString(offer.Path) + `">
<button type="submit">Open</button>
</form>
</body></html>`)
	return nil, nil
}

// signInFirst answers an anonymous reader with the sign-in wall of the KB-note
// at the open link's own address, so that signing in reloads this same address.
func signInFirst(req *appreq.Request, env Env, notePath string, cause error) error {
	nvs := env.LiveNoteViews()
	if nvs == nil || nvs.PathMap[notePath] == nil {
		return &appreq.SystemMessageError{Code: http.StatusForbidden, Msg: msgRefused, Err: cause}
	}

	return rendernotepage.WriteSigninWall(req, req.Env.(rendernotepage.Env), nvs.PathMap[notePath])
}

func siteName(site string) string {
	parsed, err := url.Parse(site)
	if err != nil || parsed.Host == "" {
		return site
	}
	return parsed.Host
}
