package rendernotepage

import (
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/model"
)

// WriteSigninWall answers an anonymous reader with the sign-in wall of a note at
// the address they asked for. Signing in reloads that address.
func WriteSigninWall(req *appreq.Request, env Env, note *model.NoteView) error {
	request := Request{Path: note.Permalink, Host: string(req.Req.Host())}

	resp, err := Resolve(req.Req, env, request)
	if resp == nil || resp.NoteView == nil {
		return err
	}

	ctx := req.Req
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.Response.Header.Set("Cache-Control", "no-store")
	ctx.Response.Header.Set("Content-Security-Policy", "frame-ancestors 'self'")

	layoutParams := renderlayout.Params{MetaRobots: "noindex, nofollow"}
	dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
	dtCtx.SigninWallError = &defaulttemplate.SigninWallError{Note: resp.NoteView}

	ctx.SetStatusCode(http.StatusUnauthorized)
	defaulttemplate.WriteRender(ctx, dtCtx)
	return nil
}
