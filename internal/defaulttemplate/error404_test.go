package defaulttemplate

import (
	"net/http"
	"testing"

	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestWriteNotFound(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	WriteNotFound(ctx, stubEnv{}, &usertoken.Data{ID: 1, Role: "reader"})

	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Header.ContentType()), "text/html")
	body := string(ctx.Response.Body())
	require.Contains(t, body, "Page not found")
}
