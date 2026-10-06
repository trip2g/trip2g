package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"testing"
	"trip2g/assets"
	"trip2g/internal/appconfig"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestAssetsHandlerCacheHeaders(t *testing.T) {
	// The dev build reads assets from ./assets, relative to the repo root.
	t.Chdir("../..")

	content, err := fs.ReadFile(assets.FS, "favicon.svg")
	require.NoError(t, err)

	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])
	etag := `W/"` + hash + `"`

	tests := []struct {
		name         string
		devMode      bool
		uri          string
		headers      map[string]string
		status       int
		cacheControl string
		etag         string
	}{
		{
			name:         "hashed url is immutable",
			uri:          "/assets/favicon.svg?h=" + hash[:8],
			status:       fasthttp.StatusOK,
			cacheControl: "public, max-age=31536000, immutable",
			etag:         etag,
		},
		{
			name:         "stale hash is not immutable",
			uri:          "/assets/favicon.svg?h=deadbeef",
			status:       fasthttp.StatusOK,
			cacheControl: "public, max-age=3600",
			etag:         etag,
		},
		{
			name:         "unhashed url gets short max-age",
			uri:          "/assets/favicon.svg",
			status:       fasthttp.StatusOK,
			cacheControl: "public, max-age=3600",
			etag:         etag,
		},
		{
			name:         "matching if-none-match is 304",
			uri:          "/assets/favicon.svg",
			headers:      map[string]string{"If-None-Match": etag},
			status:       fasthttp.StatusNotModified,
			cacheControl: "public, max-age=3600",
			etag:         etag,
		},
		{
			name:         "stale if-none-match serves the file",
			uri:          "/assets/favicon.svg",
			headers:      map[string]string{"If-None-Match": `W/"0123456789abcdef"`},
			status:       fasthttp.StatusOK,
			cacheControl: "public, max-age=3600",
			etag:         etag,
		},
		{
			name:         "zero if-modified-since is not trusted",
			uri:          "/assets/favicon.svg",
			headers:      map[string]string{"If-Modified-Since": "Mon, 01 Jan 0001 00:00:00 GMT"},
			status:       fasthttp.StatusOK,
			cacheControl: "public, max-age=3600",
			etag:         etag,
		},
		{
			name:         "dev mode is not cached",
			devMode:      true,
			uri:          "/assets/favicon.svg?h=" + hash[:8],
			status:       fasthttp.StatusOK,
			cacheControl: "no-cache",
			etag:         etag,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{
				appState: &appState{
					config: &appconfig.Config{DevMode: tt.devMode},
					log:    &logger.TestLogger{},
				},
			}
			a.setupAssets()

			ctx := &fasthttp.RequestCtx{}
			ctx.Init(&fasthttp.Request{}, nil, nil)
			ctx.Request.SetRequestURI(tt.uri)
			for k, v := range tt.headers {
				ctx.Request.Header.Set(k, v)
			}

			a.assetsHandler()(ctx)

			resp := &ctx.Response
			require.Equal(t, tt.status, resp.StatusCode())
			require.Equal(t, tt.cacheControl, string(resp.Header.Peek("Cache-Control")))
			require.Equal(t, tt.etag, string(resp.Header.Peek("ETag")))
			require.Empty(t, resp.Header.Peek("Last-Modified"))

			if tt.status == fasthttp.StatusOK {
				require.Equal(t, content, resp.Body())
			} else {
				require.Empty(t, resp.Body())
			}
		})
	}
}
