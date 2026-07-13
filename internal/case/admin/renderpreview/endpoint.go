package renderpreview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/case/checkapikey"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

// Endpoint handles POST /_system/renderlayout.
type Endpoint struct{}

type httpErrorBody struct {
	Error string `json:"error"`
}

// endpointEnv composes this endpoint's own Env with checkapikey.Env, needed
// only for the API-key auth fallback below. Kept local to the endpoint (not
// added to Env) because Resolve's auth is the caller's responsibility.
type endpointEnv interface {
	Env
	checkapikey.Env
}

func (e *Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req

	env, ok := req.Env.(endpointEnv)
	if !ok {
		return nil, appreq.ErrInvalidEnv
	}

	// Auth: admin session OR API key.
	token, _ := req.UserToken()
	if !token.IsAdmin() {
		_, err := checkapikey.Resolve(ctx, env, "render_preview")
		if err != nil {
			return writeErr(ctx, http.StatusUnauthorized, "unauthorized: "+err.Error())
		}
		token = adminToken()
	}
	_ = token

	var input graphmodel.RenderLayoutInput
	if err := json.Unmarshal(ctx.PostBody(), &input); err != nil {
		return writeErr(ctx, http.StatusBadRequest, "invalid JSON: "+err.Error())
	}

	resp, err := Resolve(ctx, env, input)
	if err != nil {
		return writeErr(ctx, http.StatusBadRequest, err.Error())
	}

	return writeJSON(ctx, http.StatusOK, resp)
}

func (*Endpoint) Path() string   { return "/_system/renderlayout" }
func (*Endpoint) Method() string { return http.MethodPost }

// GetEndpoint handles GET /_system/renderlayout (latest, ?preview_id, ?live, ?longpolling).
type GetEndpoint struct{}

func (e *GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req
	args := ctx.QueryArgs()

	env, ok := req.Env.(endpointEnv)
	if !ok {
		return nil, appreq.ErrInvalidEnv
	}
	buf := env.PreviewBuffer()

	// ?longpolling&since=N — no auth, returns {action, version} after ≤30 s.
	if args.Has("longpolling") {
		since, _ := strconv.Atoi(string(args.Peek("since")))
		version, notify := buf.Poll()
		if version > since {
			return writeLongPollJSON(ctx, "reload", version)
		}
		select {
		case <-notify:
			version, _ = buf.Poll()
			return writeLongPollJSON(ctx, "reload", version)
		case <-time.After(30 * time.Second):
			return writeLongPollJSON(ctx, "wait", since)
		}
	}

	// ?preview_id=xxx — no auth required.
	if id := string(args.Peek("preview_id")); id != "" {
		entry, ok := buf.GetByID(id)
		if !ok {
			ctx.SetStatusCode(http.StatusNotFound)
			ctx.SetBodyString("preview not found")
			return nil, nil
		}
		serveHTML(ctx, entry.HTML, false, 0)
		return nil, nil
	}

	// Default and ?live — require admin auth or API key.
	token, _ := req.UserToken()
	if !token.IsAdmin() {
		_, err := checkapikey.Resolve(ctx, env, "render_preview")
		if err != nil {
			ctx.SetStatusCode(http.StatusUnauthorized)
			ctx.SetBodyString("unauthorized")
			return nil, nil //nolint:nilerr // HTTP handler sets status code; error is not propagated
		}
	}

	entry, ok := buf.Latest()
	if !ok {
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetBodyString("no preview available yet")
		return nil, nil
	}

	live := args.Has("live")
	serveHTML(ctx, entry.HTML, live, entry.Version)
	return nil, nil
}

func (*GetEndpoint) Path() string   { return "/_system/renderlayout" }
func (*GetEndpoint) Method() string { return http.MethodGet }

func serveHTML(ctx interface {
	SetContentType(string)
	SetStatusCode(int)
	SetBodyString(string)
}, html string, injectScript bool, version int) {
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)
	if injectScript {
		html = injectLongPollScript(html, version)
	}
	ctx.SetBodyString(html)
}

const longPollScriptTpl = `<script>
(function(){
  var v=%d;
  function poll(){
    fetch('/_system/renderlayout?longpolling&since='+v)
      .then(function(r){return r.json();})
      .then(function(d){
        if(d.action==='reload'){location.reload();}
        else{poll();}
      })
      .catch(function(){setTimeout(poll,3000);});
  }
  poll();
})();
</script>`

func injectLongPollScript(html string, version int) string {
	script := fmt.Sprintf(longPollScriptTpl, version)
	if i := strings.LastIndex(html, "</body>"); i >= 0 {
		return html[:i] + script + html[i:]
	}
	return html + script
}

func writeLongPollJSON(ctx interface {
	SetContentType(string)
	SetStatusCode(int)
	SetBody([]byte)
}, action string, version int) (interface{}, error) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(http.StatusOK)
	data, _ := json.Marshal(map[string]interface{}{"action": action, "version": version})
	ctx.SetBody(data)
	return nil, nil
}

func writeJSON(ctx interface {
	SetContentType(string)
	SetStatusCode(int)
	SetBody([]byte)
}, status int, body any) (interface{}, error) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	data, _ := json.Marshal(body)
	ctx.SetBody(data)
	return nil, nil
}

func writeErr(ctx interface {
	SetContentType(string)
	SetStatusCode(int)
	SetBody([]byte)
}, status int, msg string) (interface{}, error) {
	return writeJSON(ctx, status, httpErrorBody{Error: msg})
}

func adminToken() *usertoken.Data {
	return &usertoken.Data{Role: "admin"}
}
