package federatedassetrelay_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"

	"trip2g/internal/appreq"
	"trip2g/internal/case/system/federatedassetrelay"
	"trip2g/internal/federatedassetlink"
	"trip2g/internal/federation"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

const (
	linkSecret = "instance-secret"
	maxSize    = 1 << 20
	fileBody   = "<html><script>alert(1)</script></html>"
)

// serve runs handler behind an in-memory HTTP server and returns a client
// dialing it.
func serve(t *testing.T, handler fasthttp.RequestHandler, client *fasthttp.Client) {
	t.Helper()

	ln := fasthttputil.NewInmemoryListener()
	server := &fasthttp.Server{Handler: handler}
	go func() {
		_ = server.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = ln.Close()
	})

	client.Dial = func(string) (net.Conn, error) {
		return ln.Dial()
	}
}

// peerFetcher is the real asset client talking to a stub peer that answers
// every request with peer.
func peerFetcher(t *testing.T, peer fasthttp.RequestHandler) model.FederationAssetFetcher {
	t.Helper()

	client := &fasthttp.Client{StreamResponseBody: true, MaxResponseBodySize: 16}
	serve(t, peer, client)

	return newFetcher(client)
}

func newFetcher(client *fasthttp.Client) model.FederationAssetFetcher {
	pairing := model.FederationPeer{
		KBURL:      "http://peer.test/_system/mcp",
		KID:        "consumer",
		Secret:     []byte("0123456789abcdef0123456789abcdef"),
		PrevSecret: []byte("fedcba9876543210fedcba9876543210"),
		Issuer:     "http://consumer.test",
	}
	limits := federation.AssetLimits{MaxSize: maxSize, Timeout: 5 * time.Second}
	return federation.NewAssetClient(pairing, client, limits)
}

func peerAnswers(status int) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(status)
	}
}

type relay struct {
	client *fasthttp.Client
}

func newRelay(t *testing.T, fetcher model.FederationAssetFetcher) *relay {
	t.Helper()

	var spent federatedassetlink.Spent
	env := &EnvMock{
		FederationAssetFetcherFunc: func(context.Context, string) (model.FederationAssetFetcher, error) {
			return fetcher, nil
		},
		FederatedAssetLinkSecretFunc: func() string { return linkSecret },
		SpendFederatedAssetLinkFunc:  spent.SpendFederatedAssetLink,
		LoggerFunc:                   func() logger.Logger { return &logger.DummyLogger{} },
	}

	endpoint := &federatedassetrelay.Endpoint{}
	r := &relay{client: &fasthttp.Client{}}
	serve(t, func(ctx *fasthttp.RequestCtx) {
		req := appreq.Acquire()
		defer appreq.Release(req)
		req.Env = env
		req.Req = ctx
		_, err := endpoint.Handle(req)
		if err != nil {
			ctx.SetStatusCode(http.StatusInternalServerError)
		}
	}, r.client)

	return r
}

func link(t *testing.T, name string, expiresAt time.Time) string {
	t.Helper()

	l := federatedassetlink.Link{KBID: "peer", NotePath: "skills/reel.md", Name: name}
	token, err := federatedassetlink.Sign(l, linkSecret, expiresAt)
	require.NoError(t, err)

	return token
}

func (r *relay) get(t *testing.T, token string) *fasthttp.Response {
	t.Helper()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("http://consumer.test" + federatedassetrelay.Path + "?" + federatedassetrelay.LinkParam + "=" + url.QueryEscape(token))

	resp := &fasthttp.Response{}
	err := r.client.DoTimeout(req, resp, 5*time.Second)
	require.NoError(t, err)

	return resp
}

func servesFile(ctx *fasthttp.RequestCtx) {
	ctx.SetBodyString(fileBody)
}

func TestRelayStreamsAnAttachmentAsADownload(t *testing.T) {
	r := newRelay(t, peerFetcher(t, servesFile))

	resp := r.get(t, link(t, "files/page.html", time.Now().Add(time.Minute)))

	require.Equal(t, http.StatusOK, resp.StatusCode(), string(resp.Body()))
	require.Equal(t, fileBody, string(resp.Body()))
	require.Equal(t, `attachment; filename=page.html`, string(resp.Header.Peek("Content-Disposition")))
	require.Equal(t, "nosniff", string(resp.Header.Peek("X-Content-Type-Options")))
	require.Equal(t, "sandbox", string(resp.Header.Peek("Content-Security-Policy")))
	require.Equal(t, "no-store", string(resp.Header.Peek("Cache-Control")))
	require.Equal(t, strconv.Itoa(len(fileBody)), string(resp.Header.Peek("Content-Length")))
}

func TestRelayNamesANonASCIIFileForBrowsers(t *testing.T) {
	r := newRelay(t, peerFetcher(t, servesFile))

	resp := r.get(t, link(t, "files/отчёт 1.pdf", time.Now().Add(time.Minute)))

	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.Equal(t, `attachment; filename*=utf-8''%D0%BE%D1%82%D1%87%D1%91%D1%82%201.pdf`, string(resp.Header.Peek("Content-Disposition")))
}

func TestALinkWorksOnce(t *testing.T) {
	r := newRelay(t, peerFetcher(t, servesFile))
	token := link(t, "files/attach.zip", time.Now().Add(time.Minute))

	first := r.get(t, token)
	require.Equal(t, http.StatusOK, first.StatusCode())

	second := r.get(t, token)
	require.Equal(t, http.StatusGone, second.StatusCode())
	require.Contains(t, string(second.Body()), "call federated_asset again")

	fresh := r.get(t, link(t, "files/attach.zip", time.Now().Add(time.Minute)))
	require.Equal(t, http.StatusOK, fresh.StatusCode(), "a new link to the same file still works")
}

func TestAFailedDownloadSpendsTheLink(t *testing.T) {
	r := newRelay(t, peerFetcher(t, peerAnswers(http.StatusBadGateway)))
	token := link(t, "files/attach.zip", time.Now().Add(time.Minute))

	require.Equal(t, http.StatusBadGateway, r.get(t, token).StatusCode())
	require.Equal(t, http.StatusGone, r.get(t, token).StatusCode())
}

func TestRelayRefusals(t *testing.T) {
	noLength := func(ctx *fasthttp.RequestCtx) {
		ctx.SetBodyStream(strings.NewReader(fileBody), -1)
	}
	unreachable := &fasthttp.Client{
		StreamResponseBody: true,
		Dial: func(string) (net.Conn, error) {
			return nil, errors.New("connection refused")
		},
	}

	peerSays := func(handler fasthttp.RequestHandler) func(t *testing.T) model.FederationAssetFetcher {
		return func(t *testing.T) model.FederationAssetFetcher {
			return peerFetcher(t, handler)
		}
	}

	cases := []struct {
		name    string
		fetcher func(t *testing.T) model.FederationAssetFetcher
		expires time.Duration
		status  int
	}{
		{"an expired link", peerSays(servesFile), -time.Minute, http.StatusNotFound},
		{"the peer has no such attachment", peerSays(peerAnswers(http.StatusNotFound)), time.Minute, http.StatusNotFound},
		{"the peer says it is too large", peerSays(peerAnswers(http.StatusRequestEntityTooLarge)), time.Minute, http.StatusRequestEntityTooLarge},
		{"the peer sends no Content-Length", peerSays(noLength), time.Minute, http.StatusBadGateway},
		{"the peer is unreachable", func(*testing.T) model.FederationAssetFetcher { return newFetcher(unreachable) }, time.Minute, http.StatusBadGateway},
		{"the peer revoked the pairing", peerSays(peerAnswers(http.StatusUnauthorized)), time.Minute, http.StatusBadGateway},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRelay(t, tc.fetcher(t))

			resp := r.get(t, link(t, "files/attach.zip", time.Now().Add(tc.expires)))

			require.Equal(t, tc.status, resp.StatusCode(), string(resp.Body()))
			require.Empty(t, resp.Header.Peek("Content-Disposition"), "a refusal is not a download")
		})
	}
}
