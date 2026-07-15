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
