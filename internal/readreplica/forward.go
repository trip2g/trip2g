package readreplica

import (
	"time"

	"github.com/valyala/fasthttp"
)

// authTTL bounds how long a signed X-Replica-Auth token is accepted. Short
// enough to limit replay, long enough to absorb clock skew between nodes.
const authTTL = 30 * time.Second

// forwardTimeout bounds a single forwarded request to the leader.
const forwardTimeout = 60 * time.Second

// Forwarder reverse-proxies a replica's mutating requests to the leader's
// intake. It dials a fixed leader address (HostClient) while preserving the
// original Host header, so the leader's multidomain routing still selects the
// right site. Each forwarded request is signed with X-Replica-Auth so the
// leader's intake can authorize it. The transport is plain HTTP: the intake is
// the leader's internal port, reached over the private network.
type Forwarder struct {
	client *fasthttp.HostClient
	secret string
	now    func() time.Time
}

// NewForwarder builds a Forwarder for the given leader internal address
// (host:port, e.g. "10.20.0.2:8082") and shared secret (the deployment's
// --jwt-secret).
func NewForwarder(leaderAddr, secret string) *Forwarder {
	return &Forwarder{
		client: &fasthttp.HostClient{
			Addr:                leaderAddr,
			MaxConns:            512,
			ReadTimeout:         forwardTimeout,
			WriteTimeout:        forwardTimeout,
			MaxIdleConnDuration: 90 * time.Second,
		},
		secret: secret,
		now:    time.Now,
	}
}

// Forward proxies the current request to the leader and copies the leader's
// response back verbatim (status, headers incl. Set-Cookie, body). On a
// transport error it returns 502 so the caller surfaces a clear failure.
func (f *Forwarder) Forward(ctx *fasthttp.RequestCtx) {
	upReq := fasthttp.AcquireRequest()
	upResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(upReq)
	defer fasthttp.ReleaseResponse(upResp)

	// Copy method, URI (path+query), headers (incl. original Host + cookies),
	// and body. HostClient dials its fixed Addr, so the preserved Host header
	// drives the leader's site routing rather than the dial target.
	ctx.Request.CopyTo(upReq)

	upReq.Header.Set(AuthHeader, SignAuth(f.secret, f.now(), authTTL))
	// Relay the real client IP for the leader's logging/rate-limiting.
	upReq.Header.Set("X-Forwarded-For", ctx.RemoteIP().String())

	if err := f.client.DoTimeout(upReq, upResp, forwardTimeout); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.SetBodyString("502 read replica: leader unreachable")
		return
	}

	upResp.CopyTo(&ctx.Response)
}
