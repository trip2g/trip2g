package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"runtime"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

const testBodyLimit = 64 * 1024

const tooLargeBody = `{"errors":[{"message":"request body size exceeds the limit"}]}`

type bodyEcho struct {
	calls atomic.Int32
}

// handle answers with the length and hash of the body it received.
func (e *bodyEcho) handle(ctx *fasthttp.RequestCtx) {
	e.calls.Add(1)

	if string(ctx.Path()) == "/slow" {
		time.Sleep(time.Second)
	}

	ctx.SetBodyString(bodyDigest(ctx.PostBody()))
}

func bodyDigest(body []byte) string {
	sum := sha256.Sum256(body)

	return strconv.Itoa(len(body)) + ":" + hex.EncodeToString(sum[:])
}

func serveInMemory(t *testing.T, server *fasthttp.Server) *fasthttputil.InmemoryListener {
	t.Helper()

	ln := fasthttputil.NewInmemoryListener()

	go func() {
		_ = server.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	return ln
}

func dial(t *testing.T, ln *fasthttputil.InmemoryListener) net.Conn {
	t.Helper()

	conn, err := ln.Dial()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return conn
}

func readResponse(t *testing.T, br *bufio.Reader) *fasthttp.Response {
	t.Helper()

	resp := &fasthttp.Response{}
	err := resp.Read(br)
	require.NoError(t, err)

	return resp
}

func fixedRequest(path string, headers string, body []byte) []byte {
	head := "POST " + path + " HTTP/1.1\r\nHost: trip2g.test\r\n" + headers +
		"Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n"

	return append([]byte(head), body...)
}

func chunkedRequest(path string, body []byte) []byte {
	var buf bytes.Buffer

	buf.WriteString("POST " + path + " HTTP/1.1\r\nHost: trip2g.test\r\nTransfer-Encoding: chunked\r\n\r\n")

	for chunk := range slices.Chunk(body, 4096) {
		buf.WriteString(strconv.FormatInt(int64(len(chunk)), 16) + "\r\n")
		buf.Write(chunk)
		buf.WriteString("\r\n")
	}

	buf.WriteString("0\r\n\r\n")

	return buf.Bytes()
}

func patternBody(n int) []byte {
	body := make([]byte, n)
	for i := range body {
		body[i] = byte(i % 251)
	}

	return body
}

// send writes raw in the background, so a server that answers before it has
// read everything does not block the test.
func send(conn net.Conn, raw []byte) {
	go func() {
		_, _ = conn.Write(raw)
	}()
}

func TestServersLimitStreamedRequestBody(t *testing.T) {
	atLimit := patternBody(testBodyLimit)
	overLimit := patternBody(testBodyLimit + 1)

	servers := []struct {
		name string
		new  func(fasthttp.RequestHandler) *fasthttp.Server
	}{
		{
			name: "public",
			new: func(h fasthttp.RequestHandler) *fasthttp.Server {
				return newHTTPServer(h, time.Minute, testBodyLimit)
			},
		},
		{
			name: "internal",
			new: func(h fasthttp.RequestHandler) *fasthttp.Server {
				return newInternalServer(h, time.Minute, testBodyLimit)
			},
		},
	}

	cases := []struct {
		name    string
		raw     []byte
		status  int
		body    string
		handled bool
	}{
		{
			name:    "content-length at limit",
			raw:     fixedRequest("/", "", atLimit),
			status:  fasthttp.StatusOK,
			body:    bodyDigest(atLimit),
			handled: true,
		},
		{
			name:   "content-length over limit",
			raw:    fixedRequest("/", "", overLimit),
			status: fasthttp.StatusRequestEntityTooLarge,
			body:   tooLargeBody,
		},
		{
			name:    "chunked at limit",
			raw:     chunkedRequest("/", atLimit),
			status:  fasthttp.StatusOK,
			body:    bodyDigest(atLimit),
			handled: true,
		},
		{
			name:   "chunked over limit",
			raw:    chunkedRequest("/", overLimit),
			status: fasthttp.StatusRequestEntityTooLarge,
			body:   tooLargeBody,
		},
		{
			name:   "event stream over limit",
			raw:    fixedRequest("/", "Accept: text/event-stream\r\n", overLimit),
			status: fasthttp.StatusRequestEntityTooLarge,
			body:   tooLargeBody,
		},
	}

	for _, srv := range servers {
		for _, tt := range cases {
			t.Run(srv.name+"/"+tt.name, func(t *testing.T) {
				echo := &bodyEcho{}
				ln := serveInMemory(t, srv.new(echo.handle))
				conn := dial(t, ln)

				send(conn, tt.raw)
				resp := readResponse(t, bufio.NewReader(conn))

				require.Equal(t, tt.status, resp.StatusCode())
				require.Equal(t, tt.body, string(resp.Body()))
				require.Equal(t, tt.handled, echo.calls.Load() == 1)
			})
		}
	}
}

func TestPublicServerDoesNotReserveAnnouncedBody(t *testing.T) {
	const limit = 1024 * 1024
	const connections = 50

	echo := &bodyEcho{}
	ln := serveInMemory(t, newHTTPServer(echo.handle, time.Minute, limit))

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	head := []byte("POST / HTTP/1.1\r\nHost: trip2g.test\r\nContent-Length: " + strconv.Itoa(limit-1) + "\r\n\r\n")
	for range connections {
		conn := dial(t, ln)
		_, err := conn.Write(head)
		require.NoError(t, err)
	}

	time.Sleep(500 * time.Millisecond)

	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)

	growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	t.Logf("heap grew by %d bytes for %d idle bodies", growth, connections)
	require.Less(t, growth, int64(10*1024*1024), "heap grew by %d bytes for %d idle bodies", growth, connections)
	require.Zero(t, echo.calls.Load())
}

func TestPublicServerKeepsConnectionAfterHandlerTimeout(t *testing.T) {
	echo := &bodyEcho{}
	ln := serveInMemory(t, newHTTPServer(echo.handle, 300*time.Millisecond, testBodyLimit))
	conn := dial(t, ln)
	br := bufio.NewReader(conn)

	first := patternBody(20 * 1024)
	send(conn, fixedRequest("/slow", "", first))
	resp := readResponse(t, br)
	require.Equal(t, fasthttp.StatusRequestTimeout, resp.StatusCode())

	second := patternBody(30 * 1024)
	send(conn, fixedRequest("/", "", second))
	resp = readResponse(t, br)
	require.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	require.Equal(t, bodyDigest(second), string(resp.Body()))
}

func TestPublicServerAcceptsExpectContinue(t *testing.T) {
	echo := &bodyEcho{}
	ln := serveInMemory(t, newHTTPServer(echo.handle, time.Minute, testBodyLimit))
	conn := dial(t, ln)
	br := bufio.NewReader(conn)

	body := patternBody(40 * 1024)
	head := "POST / HTTP/1.1\r\nHost: trip2g.test\r\nExpect: 100-continue\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n"
	_, err := conn.Write([]byte(head))
	require.NoError(t, err)

	continueLine := "HTTP/1.1 100 Continue\r\n\r\n"
	got := make([]byte, len(continueLine))
	_, err = io.ReadFull(br, got)
	require.NoError(t, err)
	require.Equal(t, continueLine, string(got))

	send(conn, body)
	resp := readResponse(t, br)
	require.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	require.Equal(t, bodyDigest(body), string(resp.Body()))
}

func TestServersBoundBodyByReadTimeout(t *testing.T) {
	body := patternBody(40 * 1024)
	raw := fixedRequest("/", "", body)
	headEnd := len(raw) - len(body)

	cases := []struct {
		name   string
		split  int
		pause  time.Duration
		status int
	}{
		{name: "pause shorter than read timeout", split: headEnd, pause: 150 * time.Millisecond, status: fasthttp.StatusOK},
		{name: "pause longer than read timeout", split: headEnd + 16*1024, pause: time.Second, status: fasthttp.StatusBadRequest},
	}

	servers := []struct {
		name string
		make func(h fasthttp.RequestHandler) *fasthttp.Server
	}{
		{name: "public", make: func(h fasthttp.RequestHandler) *fasthttp.Server {
			return newHTTPServer(h, 500*time.Millisecond, testBodyLimit)
		}},
		{name: "internal", make: func(h fasthttp.RequestHandler) *fasthttp.Server {
			return newInternalServer(h, 500*time.Millisecond, testBodyLimit)
		}},
	}

	for _, srv := range servers {
		for _, tt := range cases {
			t.Run(srv.name+"/"+tt.name, func(t *testing.T) {
				echo := &bodyEcho{}
				ln := serveInMemory(t, srv.make(echo.handle))
				conn := dial(t, ln)

				go func() {
					_, _ = conn.Write(raw[:tt.split])
					time.Sleep(tt.pause)
					_, _ = conn.Write(raw[tt.split:])
				}()

				resp := readResponse(t, bufio.NewReader(conn))
				require.Equal(t, tt.status, resp.StatusCode())

				if tt.status == fasthttp.StatusOK {
					require.Equal(t, bodyDigest(body), string(resp.Body()))
					return
				}

				require.Zero(t, echo.calls.Load())
			})
		}
	}
}
