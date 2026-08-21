package codellm

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/delegatedadmin"
)

func sealServer(t *testing.T, path string) *Server {
	t.Helper()
	t.Setenv(DefaultSealEnvKey, testKey)
	return New(Config{SealPath: path})
}

func postSeal(path, envKey, value string) *http.Request {
	form := url.Values{"env_key": {envKey}, "value": {value}}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestSealForm_GETRendersEmptyForm(t *testing.T) {
	rec := serve(sealServer(t, ""), httptest.NewRequest(http.MethodGet, DefaultSealPath, nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	body := rec.Body.String()
	require.Contains(t, body, `name="value"`)
	require.Contains(t, body, DefaultSealEnvKey, "the key field is prefilled with the default")
	require.NotContains(t, body, sealedPrefix, "the empty form carries no blob")
}

func TestSealForm_POSTSealsAndRoundTrips(t *testing.T) {
	rec := serve(sealServer(t, ""), postSeal(DefaultSealPath, "", "krisp-token"))

	require.Equal(t, http.StatusOK, rec.Code)
	blob := extractBlob(t, rec.Body.String())
	got, err := openSealed(testKey, blob)
	require.NoError(t, err)
	require.Equal(t, "krisp-token", got)
}

// The rendered result must not carry the plaintext back to the browser: it
// would sit in the DOM, the back-forward cache and any page the operator saves.
func TestSealForm_ResultDoesNotEchoThePlaintext(t *testing.T) {
	rec := serve(sealServer(t, ""), postSeal(DefaultSealPath, "", "plaintext-never-echoed"))
	require.NotContains(t, rec.Body.String(), "plaintext-never-echoed")
}

func TestSealForm_CustomEnvKey(t *testing.T) {
	srv := sealServer(t, "")
	t.Setenv("SEAL_KEY_V2", testKey)

	rec := serve(srv, postSeal(DefaultSealPath, "SEAL_KEY_V2", "v"))

	require.Equal(t, http.StatusOK, rec.Code)
	got, err := openSealed(testKey, extractBlob(t, rec.Body.String()))
	require.NoError(t, err)
	require.Equal(t, "v", got)
}

// A browser posts a textarea's newlines as CRLF, and the shell's trailing
// newline is not part of a credential — the CLI trims it for the same reason.
func TestSealForm_NormalizesNewlines(t *testing.T) {
	rec := serve(sealServer(t, ""), postSeal(DefaultSealPath, "", "line1\r\nline2\r\n"))

	got, err := openSealed(testKey, extractBlob(t, rec.Body.String()))
	require.NoError(t, err)
	require.Equal(t, "line1\nline2", got)
}

func TestSealForm_EmptyValueIsRejected(t *testing.T) {
	rec := serve(sealServer(t, ""), postSeal(DefaultSealPath, "", ""))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotContains(t, rec.Body.String(), sealedPrefix)
}

// The value must travel in the POST body. A secret in a URL lands in
// reverse-proxy access logs, browser history and Referer headers, so a request
// carrying a query string is refused rather than quietly served.
func TestSealForm_QueryStringIsRefused(t *testing.T) {
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, DefaultSealPath+"?value=s3cret", nil),
		postSeal(DefaultSealPath+"?value=s3cret", "", "v"),
	} {
		rec := serve(sealServer(t, ""), req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.NotContains(t, rec.Body.String(), "s3cret")
	}
}

// Same rule as unseal: the response must not become a length-and-existence
// oracle over codellm's environment.
func TestSealForm_ErrorDoesNotLeakEnvDetail(t *testing.T) {
	srv := sealServer(t, "")
	t.Setenv("SHORT_KEY", "too-short")

	for _, envKey := range []string{"SHORT_KEY", "ABSENT_KEY"} {
		rec := serve(srv, postSeal(DefaultSealPath, envKey, "v"))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := rec.Body.String()
		require.NotContains(t, body, "32 bytes")
		require.NotContains(t, body, "too-short")
		require.NotContains(t, body, sealedPrefix)
	}
}

func TestSealForm_CustomPath(t *testing.T) {
	srv := sealServer(t, "/seal")

	require.Equal(t, http.StatusOK, serve(srv, httptest.NewRequest(http.MethodGet, "/seal", nil)).Code)
	require.Equal(t, http.StatusNotFound, serve(srv, httptest.NewRequest(http.MethodGet, DefaultSealPath, nil)).Code)
}

// Both verbs go through cfg.Auth, the same way the GraphQL playground's GET
// does: an ungated seal endpoint would be the weakest thing on the mux.
func TestSealForm_NoCookie_401(t *testing.T) {
	mw, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: "http://127.0.0.1:1"})
	require.NoError(t, err)
	srv := New(Config{Auth: BrowserAuth(mw.Wrap, nil)})

	require.Equal(t, http.StatusUnauthorized,
		serve(srv, httptest.NewRequest(http.MethodGet, DefaultSealPath, nil)).Code)
	require.Equal(t, http.StatusUnauthorized,
		serve(srv, postSeal(DefaultSealPath, "", "v")).Code)
}

// extractBlob pulls the sealed value out of the rendered page.
func extractBlob(t *testing.T, page string) string {
	t.Helper()
	i := strings.Index(page, sealedPrefix)
	require.GreaterOrEqual(t, i, 0, "page carries no sealed value")
	blob := page[i:]
	if end := strings.IndexAny(blob, "<\n"); end >= 0 {
		blob = blob[:end]
	}
	// html/template escapes base64's "+" as &#43;; a browser renders it back.
	return html.UnescapeString(strings.TrimSpace(blob))
}
