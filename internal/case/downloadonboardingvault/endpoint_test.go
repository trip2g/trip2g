package downloadonboardingvault

import (
	"archive/zip"
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/usertoken"
)

func adminRequest(env Env, uri string) *appreq.Request {
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod(http.MethodGet)
	reqCtx.Request.SetRequestURI(uri)

	req := &appreq.Request{Env: env, Req: reqCtx}
	req.SetUserToken(&usertoken.Data{ID: 1, Role: "admin"})

	return req
}

// zipRoots collects the first path segment of every entry in the archive.
func zipRoots(t *testing.T, zipData []byte) map[string]bool {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	roots := map[string]bool{}
	for _, file := range reader.File {
		roots[strings.SplitN(file.Name, "/", 2)[0]] = true
	}

	return roots
}

func TestEndpoint_NameSetsFilenameAndRoot(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}
	req := adminRequest(env, "/_system/onboarding-vault?enable_admin_graphql&name=secondbrain")

	_, err := (&Endpoint{}).Handle(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, req.Req.Response.StatusCode())
	require.Equal(t, `attachment; filename="secondbrain.zip"`,
		string(req.Req.Response.Header.Peek("Content-Disposition")))
	require.Equal(t, map[string]bool{"secondbrain": true}, zipRoots(t, req.Req.Response.Body()))
}

func TestEndpoint_WithoutNameFallsBackToDomain(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}
	req := adminRequest(env, "/_system/onboarding-vault")

	_, err := (&Endpoint{}).Handle(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, req.Req.Response.StatusCode())
	require.Equal(t, `attachment; filename="example.com.zip"`,
		string(req.Req.Response.Header.Peek("Content-Disposition")))
	require.Equal(t, map[string]bool{"example.com": true}, zipRoots(t, req.Req.Response.Body()))
}

func TestEndpoint_InvalidNameIsRejected(t *testing.T) {
	cases := map[string]string{
		"empty":            "",
		"slash":            "a/b",
		"backslash":        "a%5Cb",
		"traversal":        "..",
		"traversal path":   "..%2Fetc",
		"leading dot":      ".hidden",
		"leading dash":     "-x",
		"quote":            "a%22b",
		"space":            "a%20b",
		"unicode":          "%D0%BC%D0%BE%D0%B7%D0%B3",
		"newline":          "a%0Ab",
		"nul":              "a%00b",
		"absolute":         "%2Fetc%2Fpasswd",
		"too long":         strings.Repeat("a", maxVaultNameLen+1),
		"only enable flag": "%20",
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			env := &mockEnv{publicURL: "https://example.com"}
			req := adminRequest(env, "/_system/onboarding-vault?name="+value)

			_, err := (&Endpoint{}).Handle(req)
			require.NoError(t, err)

			require.Equal(t, http.StatusBadRequest, req.Req.Response.StatusCode(),
				"name %q must be rejected", value)
			require.Empty(t, req.Req.Response.Header.Peek("Content-Disposition"))
		})
	}
}

func TestEndpoint_ValidNamesAccepted(t *testing.T) {
	for _, name := range []string{"secondbrain", "second-brain", "second_brain", "vault.2", "a", "A1"} {
		t.Run(name, func(t *testing.T) {
			env := &mockEnv{publicURL: "https://example.com"}
			req := adminRequest(env, "/_system/onboarding-vault?name="+name)

			_, err := (&Endpoint{}).Handle(req)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, req.Req.Response.StatusCode())
			require.Equal(t, map[string]bool{name: true}, zipRoots(t, req.Req.Response.Body()))
		})
	}
}

// A rejected name must not mint an API key — the request never got far enough.
func TestEndpoint_InvalidNameIssuesNoAPIKey(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}
	req := adminRequest(env, "/_system/onboarding-vault?enable_admin_graphql&name=..")

	_, err := (&Endpoint{}).Handle(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusBadRequest, req.Req.Response.StatusCode())
	require.Empty(t, env.setAdminToolsCalls)
}
