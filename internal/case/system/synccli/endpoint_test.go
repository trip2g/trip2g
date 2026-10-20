package synccli_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/system/synccli"
)

func TestEndpoint(t *testing.T) {
	const cli = "console.log('sync')\n"

	withCLI := buildZip(t, map[string]string{
		"onboarding-vault/.obsidian/plugins/trip2g/trip2g-sync.mjs": cli,
	})

	tests := []struct {
		name        string
		zip         []byte
		ifNoneMatch string
		wantStatus  int
		wantBody    string
		wantHeaders map[string]string
	}{
		{
			name:       "serves the cli",
			zip:        withCLI,
			wantStatus: http.StatusOK,
			wantBody:   cli,
			wantHeaders: map[string]string{
				"Content-Type":  "text/javascript; charset=utf-8",
				"Cache-Control": "no-cache",
				"ETag":          etagOf(cli),
			},
		},
		{
			name:        "matching etag",
			zip:         withCLI,
			ifNoneMatch: etagOf(cli),
			wantStatus:  http.StatusNotModified,
			wantHeaders: map[string]string{"ETag": etagOf(cli)},
		},
		{
			name:        "stale etag",
			zip:         withCLI,
			ifNoneMatch: `"old"`,
			wantStatus:  http.StatusOK,
			wantBody:    cli,
		},
		{
			name:       "build without vault.zip",
			zip:        nil,
			wantStatus: http.StatusNotFound,
			wantBody:   "sync cli is not bundled in this build\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{OnboardingVaultZipFunc: func() []byte { return tt.zip }}

			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.Header.SetMethod(http.MethodGet)
			reqCtx.Request.SetRequestURI("/_system/sync-cli.mjs")
			if tt.ifNoneMatch != "" {
				reqCtx.Request.Header.Set("If-None-Match", tt.ifNoneMatch)
			}

			_, err := (&synccli.Endpoint{}).Handle(&appreq.Request{Env: env, Req: reqCtx})
			require.NoError(t, err)

			require.Equal(t, tt.wantStatus, reqCtx.Response.StatusCode())
			require.Equal(t, tt.wantBody, string(reqCtx.Response.Body()))
			for key, value := range tt.wantHeaders {
				require.Equal(t, value, string(reqCtx.Response.Header.Peek(key)), key)
			}
		})
	}
}

// The archive is read once per process, not on every boot of every pod.
func TestEndpoint_ReadsArchiveOnce(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"onboarding-vault/.obsidian/plugins/trip2g/trip2g-sync.mjs": "x",
	})
	env := &EnvMock{OnboardingVaultZipFunc: func() []byte { return zipData }}
	endpoint := &synccli.Endpoint{}

	for range 3 {
		reqCtx := &fasthttp.RequestCtx{}
		_, err := endpoint.Handle(&appreq.Request{Env: env, Req: reqCtx})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, reqCtx.Response.StatusCode())
	}

	require.Len(t, env.OnboardingVaultZipCalls(), 1)
}
