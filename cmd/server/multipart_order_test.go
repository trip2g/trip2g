package main

import (
	"bytes"
	"mime/multipart"
	"net"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/valyala/fasthttp/fasthttputil"
)

func multipartUploadBody(t *testing.T) ([]byte, string) {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	require.NoError(t, mw.WriteField("operations", `{"query":"{ name }","variables":{"f":null}}`))
	require.NoError(t, mw.WriteField("map", `{"0":["variables.f"]}`))

	part, err := mw.CreateFormFile("0", "a.png")
	require.NoError(t, err)

	_, err = part.Write([]byte("png bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	return buf.Bytes(), mw.FormDataContentType()
}

func postUpload(t *testing.T, client *fasthttp.Client, contentType string, body []byte) (int, string) {
	t.Helper()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://trip2g.test/_system/graphql")
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType(contentType)
	req.SetBody(body)

	err := client.Do(req, resp)
	require.NoError(t, err)

	return resp.StatusCode(), string(resp.Body())
}

func TestServersKeepMultipartFieldOrder(t *testing.T) {
	newGraphQL := func() fasthttp.RequestHandler {
		gql := testserver.New()
		gql.AddTransport(transport.MultipartForm{})

		return fasthttpadaptor.NewFastHTTPHandler(gql)
	}

	servers := []struct {
		name   string
		server *fasthttp.Server
	}{
		{name: "public", server: newHTTPServer(newGraphQL(), time.Minute, 1024*1024)},
		{name: "internal", server: newInternalServer(newGraphQL(), 1024*1024)},
	}

	for _, tt := range servers {
		t.Run(tt.name, func(t *testing.T) {
			ln := fasthttputil.NewInmemoryListener()

			go func() {
				_ = tt.server.Serve(ln)
			}()
			t.Cleanup(func() {
				_ = tt.server.Shutdown()
			})

			client := &fasthttp.Client{
				Dial: func(string) (net.Conn, error) {
					return ln.Dial()
				},
			}

			body, contentType := multipartUploadBody(t)

			failures := 0
			firstFailure := ""
			for range 400 {
				status, respBody := postUpload(t, client, contentType, body)
				if status != fasthttp.StatusOK || respBody != `{"data":{"name":"test"}}` {
					if failures == 0 {
						firstFailure = respBody
					}
					failures++
				}
			}

			require.Zero(t, failures, "first failure: %s", firstFailure)
		})
	}
}

func TestInternalServerTakesTheConfiguredBodyLimit(t *testing.T) {
	ok := func(ctx *fasthttp.RequestCtx) {
		ctx.SetBodyString("ok")
	}

	server := newInternalServer(ok, 10*1024*1024)
	ln := fasthttputil.NewInmemoryListener()

	go func() {
		_ = server.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	client := &fasthttp.Client{
		Dial: func(string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	status, _ := postUpload(t, client, "application/json", bytes.Repeat([]byte("x"), 6*1024*1024))
	require.Equal(t, fasthttp.StatusOK, status)
}
