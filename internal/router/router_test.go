package router

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/logger"
)

func TestMain(m *testing.M) {
	if err := defaulttemplate.Init(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestWriteSystemMessage_RendersThePageTheEndpointAskedFor(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}

	handled := writeSystemMessage(ctx, &appreq.SystemMessageError{
		Code: http.StatusUnauthorized,
		Msg:  "hat_expired",
		Err:  errors.New("token has invalid claims: token is expired"),
	})

	require.True(t, handled)
	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Body()), "This link has expired")
}

// The cause travels with the error for the log, but must not reach the page.
func TestWriteSystemMessage_KeepsTheCauseOffThePage(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}

	writeSystemMessage(ctx, &appreq.SystemMessageError{
		Code: http.StatusUnauthorized,
		Msg:  "hat_expired",
		Err:  errors.New("failed to parse token: token is expired by 42h"),
	})

	require.NotContains(t, string(ctx.Response.Body()), "failed to parse token")
}

func TestWriteSystemMessage_FindsTheErrorThroughWrapping(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	wrapped := fmt.Errorf("handling request: %w", &appreq.SystemMessageError{
		Code: http.StatusBadRequest,
		Msg:  "hat_invalid",
	})

	require.True(t, writeSystemMessage(ctx, wrapped))
	require.Equal(t, http.StatusBadRequest, ctx.Response.StatusCode())
}

// Every other error keeps the existing handling — this branch must not swallow
// them.
func TestWriteSystemMessage_LeavesOrdinaryErrorsAlone(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}

	require.False(t, writeSystemMessage(ctx, errors.New("database is down")))
	require.Empty(t, ctx.Response.Body())
}

// stubEnv embeds Env so the test only has to supply the one method Handle
// reaches on this path; anything else would panic loudly rather than pass.
type stubEnv struct {
	Env
}

func (stubEnv) Logger() logger.Logger { return &logger.TestLogger{Prefix: "router_test"} }

type stubEndpoint struct {
	err error
}

func (e stubEndpoint) Handle(*appreq.Request) (interface{}, error) { return nil, e.err }
func (stubEndpoint) Path() string                                  { return "/_system/stub" }
func (stubEndpoint) Method() string                                { return http.MethodGet }

// Handle must report the request as handled and swallow the error: passing it
// up makes cmd/server replace the rendered page with a 503.
func TestHandle_SystemMessageIsAFinishedResponse(t *testing.T) {
	endpoint := stubEndpoint{err: &appreq.SystemMessageError{
		Code: http.StatusUnauthorized,
		Msg:  "hat_expired",
		Err:  errors.New("token is expired"),
	}}

	router := &Router{
		env:       stubEnv{},
		getRoutes: map[string]Endpoint{endpoint.Path(): endpoint},
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI(endpoint.Path())

	handled, err := router.Handle(&appreq.Request{Req: ctx})

	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Body()), "This link has expired")
	require.NotContains(t, string(ctx.Response.Body()), "token is expired")
}
