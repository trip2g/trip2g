package main

import (
	"testing"
	"trip2g/internal/appconfig"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestHandleReplicaIntakeRejectsOrdinaryRequest(t *testing.T) {
	a := &app{
		appState: &appState{
			config: &appconfig.Config{},
			log:    &logger.TestLogger{},
		},
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.SetRequestURI("/_system/graphql")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)

	a.handleReplicaIntake(ctx)

	require.Equal(t, fasthttp.StatusNotFound, ctx.Response.StatusCode())
	require.Equal(t, "404 internal endpoint", string(ctx.Response.Body()))
}

func TestHandleCorsAllowsPluginOrigins(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		allowed bool
	}{
		{name: "obsidian desktop", origin: "app://obsidian.md", allowed: true},
		{name: "obsidian ios capacitor", origin: "capacitor://localhost", allowed: true},
		{name: "obsidian android capacitor", origin: "http://localhost", allowed: true},
		{name: "local dev ui", origin: "http://localhost:9081", allowed: true},
		{name: "unrelated site", origin: "https://evil.example.com", allowed: false},
		{name: "no origin", origin: "", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{
				appState: &appState{
					config: &appconfig.Config{},
					log:    &logger.TestLogger{},
				},
			}

			ctx := &fasthttp.RequestCtx{}
			ctx.Init(&fasthttp.Request{}, nil, nil)
			ctx.Request.Header.SetMethod(fasthttp.MethodPost)
			if tt.origin != "" {
				ctx.Request.Header.Set("Origin", tt.origin)
			}

			a.handleCors(ctx)

			got := string(ctx.Response.Header.Peek("Access-Control-Allow-Origin"))
			if tt.allowed {
				require.Equal(t, tt.origin, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

func TestWriteSitemap(t *testing.T) {
	raw := []byte("<urlset/>")
	gz := []byte("gzipped")

	tests := []struct {
		name           string
		acceptEncoding string
		wantBody       []byte
		wantEncoding   string
		wantVary       string
	}{
		{name: "gzip client gets pre-gzipped bytes", acceptEncoding: "gzip, deflate", wantBody: gz, wantEncoding: "gzip", wantVary: "Accept-Encoding"},
		{name: "plain client gets raw bytes", acceptEncoding: "", wantBody: raw},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Init(&fasthttp.Request{}, nil, nil)
			if tt.acceptEncoding != "" {
				ctx.Request.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			writeSitemap(ctx, raw, gz)

			require.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
			require.Equal(t, "application/xml; charset=utf-8", string(ctx.Response.Header.ContentType()))
			require.Equal(t, tt.wantBody, ctx.Response.Body())
			require.Equal(t, tt.wantEncoding, string(ctx.Response.Header.ContentEncoding()))
			require.Equal(t, tt.wantVary, string(ctx.Response.Header.Peek("Vary")))
		})
	}
}
