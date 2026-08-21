package codellm

import (
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

// DefaultSealPath is where the sealing form is served. The /_system/ prefix
// matches trip2g's own admin endpoints, and Caddy already forwards
// /_system/codellm/* to this service, so the default works in a browser with no
// infra change. Config.SealPath overrides it for a deployment where it collides.
const DefaultSealPath = "/_system/codellm/seal"

// ValidateSealPath rejects a configured seal path that cannot be served: the
// mux needs an absolute path, and registering one twice panics at startup. It
// lives here, with the routes, so a new endpoint cannot silently make an
// operator's configured path unservable.
func ValidateSealPath(path string) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("seal path %q must start with /", path)
	}
	for _, p := range []string{chatPath, graphqlPrefix, modelsPath, healthzPath} {
		if path == p {
			return fmt.Errorf("seal path %q is already served by codellm", path)
		}
	}
	return nil
}

// errSealFailed is the single outward-facing sealing failure. Like errUnsealFailed
// it says nothing about the named env var: NewManager's "must be exactly 32
// bytes, got %d" would turn this endpoint into a length-and-existence oracle
// over codellm's whole environment. The detail goes to codellm's own log.
var errSealFailed = errors.New("seal failed: check that the named env var holds a 32-byte key")

// sealPage renders the form and, after a POST, the blob to paste into a role
// note's frontmatter. The plaintext is deliberately absent from it: echoing the
// value back would leave the secret in the DOM and the back-forward cache.
//
//nolint:gochecknoglobals // parsed once at init, like any static template
var sealPage = template.Must(template.New("seal").Parse(`<!doctype html>
<meta charset="utf-8"><title>codellm seal</title>
<style>
body{font:14px/1.5 system-ui,sans-serif;max-width:44rem;margin:3rem auto;padding:0 1rem}
label{display:block;margin:1rem 0 .25rem;font-weight:600}
input,textarea,output{width:100%;box-sizing:border-box;font-family:ui-monospace,monospace;font-size:13px}
textarea{height:6rem}
output{display:block;word-break:break-all;background:#f4f4f5;padding:.75rem;border-radius:4px}
.err{color:#b00020}
</style>
<h1>Seal a secret</h1>
<p>Paste the value, copy the result into the role note's frontmatter and list the
field in <code>unseal</code>.</p>
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
{{if .Sealed}}<label>Sealed value</label><output>{{.Sealed}}</output>{{end}}
<form method="post" autocomplete="off">
<label for="env_key">Key env var</label>
<input id="env_key" name="env_key" value="{{.EnvKey}}" spellcheck="false">
<label for="value">Value to seal</label>
<textarea id="value" name="value" spellcheck="false" autofocus></textarea>
<p><button type="submit">Seal</button></p>
</form>
`))

type sealView struct {
	EnvKey string
	Sealed string
	Error  string
}

// handleSeal serves both verbs of the sealing form: GET renders it empty, POST
// seals the submitted value. It is registered inside cfg.Auth like its
// neighbours — an unauthenticated seal endpoint would be the weakest thing on
// the mux.
func (s *Server) handleSeal(w http.ResponseWriter, r *http.Request) {
	// A secret that reaches a URL lands in reverse-proxy access logs, browser
	// history and Referer headers. Nothing here takes a query parameter, so any
	// query string is either a mistake or an attempt to route the value through
	// one; refuse loudly instead of serving it.
	if r.URL.RawQuery != "" {
		http.Error(w, "the value travels in the POST body, never in the URL", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodGet {
		s.renderSeal(w, http.StatusOK, sealView{EnvKey: DefaultSealEnvKey})
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	// PostFormValue, not FormValue: the latter would also read the query string.
	envKey := strings.TrimSpace(r.PostFormValue("env_key"))
	if envKey == "" {
		envKey = DefaultSealEnvKey
	}
	// A browser posts a textarea's newlines as CRLF, which would corrupt a
	// multi-line secret; the trailing newline is the one the form adds, not part
	// of the credential — the same call the seal CLI makes on stdin.
	value := strings.TrimRight(strings.ReplaceAll(r.PostFormValue("value"), "\r\n", "\n"), "\n")
	if value == "" {
		s.renderSeal(w, http.StatusBadRequest, sealView{EnvKey: envKey, Error: "value is required"})
		return
	}

	blob, err := Seal(osEnv{}.get(envKey), value)
	if err != nil {
		//nolint:sloglint // codellm has no logger instance; global slog is intentional here
		slog.Error("codellm: seal failed", "env_key", envKey, "error", err)
		s.renderSeal(w, http.StatusUnprocessableEntity, sealView{EnvKey: envKey, Error: errSealFailed.Error()})
		return
	}
	s.renderSeal(w, http.StatusOK, sealView{EnvKey: envKey, Sealed: blob})
}

func (s *Server) renderSeal(w http.ResponseWriter, status int, view sealView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The page carries a blob and is reached through a session cookie: nothing
	// about it belongs in a shared cache or in a proxy's store.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = sealPage.Execute(w, view)
}
