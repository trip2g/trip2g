package appresp

import "github.com/valyala/fasthttp"

// Redirect sends the browser to location, written to the Location header
// exactly as given.
//
// fasthttp's ctx.Redirect resolves a relative target against the request URI
// and emits the absolute result, so behind a TLS-terminating proxy — where the
// request always arrives over plain http — every relative redirect downgrades
// the browser to http. The Secure cookies just set on the https response are
// then not sent, and a sign-in that succeeded looks like one that failed.
// A relative Location is resolved by the browser against the page it is
// actually on, which is the only place the real scheme still exists.
func Redirect(ctx *fasthttp.RequestCtx, location string, statusCode int) {
	ctx.Response.Header.Set(fasthttp.HeaderLocation, location)
	ctx.Response.SetStatusCode(statusCode)
}
