package readreplica

import (
	"net"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// fakeLeader starts an in-memory fasthttp server wrapping appHandler with the
// same X-Replica-Auth enforcement startReplicaIntake uses, and returns a
// Forwarder wired to it.
func fakeLeader(t *testing.T, secret string, appHandler fasthttp.RequestHandler) *Forwarder {
	t.Helper()

	ln := fasthttputil.NewInmemoryListener()

	intake := func(ctx *fasthttp.RequestCtx) {
		auth := string(ctx.Request.Header.Peek(AuthHeader))
		if err := VerifyAuth(secret, auth, time.Now()); err != nil {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}
		appHandler(ctx)
	}

	go func() { _ = fasthttp.Serve(ln, intake) }()
	t.Cleanup(func() { _ = ln.Close() })

	return &Forwarder{
		client: &fasthttp.HostClient{
			Addr: "leader",
			Dial: func(addr string) (net.Conn, error) { return ln.Dial() },
		},
		secret: secret,
		now:    time.Now,
	}
}

func TestForwardRelaysAuthenticatedWrite(t *testing.T) {
	const secret = "shared-secret"

	var gotMethod, gotPath, gotBody string
	fwd := fakeLeader(t, secret, func(ctx *fasthttp.RequestCtx) {
		gotMethod = string(ctx.Method())
		gotPath = string(ctx.Path())
		gotBody = string(ctx.PostBody())
		ctx.SetStatusCode(fasthttp.StatusCreated)
		ctx.Response.Header.Set("Set-Cookie", "session=abc")
		ctx.SetBodyString("leader-handled")
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/_system/graphql")
	ctx.Request.Header.SetHost("example.test")
	ctx.Request.SetBodyString(`{"query":"mutation{...}"}`)

	fwd.Forward(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201", ctx.Response.StatusCode())
	}
	if gotMethod != "POST" || gotPath != "/_system/graphql" {
		t.Errorf("leader saw %s %s, want POST /_system/graphql", gotMethod, gotPath)
	}
	if gotBody != `{"query":"mutation{...}"}` {
		t.Errorf("leader body = %q", gotBody)
	}
	if got := string(ctx.Response.Body()); got != "leader-handled" {
		t.Errorf("relayed body = %q", got)
	}
	if got := string(ctx.Response.Header.Peek("Set-Cookie")); got != "session=abc" {
		t.Errorf("Set-Cookie not relayed: %q", got)
	}
}

func TestIntakeRejectsUnsignedForward(t *testing.T) {
	const secret = "shared-secret"

	handlerCalled := false
	fwd := fakeLeader(t, secret, func(ctx *fasthttp.RequestCtx) {
		handlerCalled = true
		ctx.SetStatusCode(fasthttp.StatusOK)
	})

	// Forge a forwarder that signs with the WRONG secret — intake must reject.
	fwd.secret = "wrong-secret"

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/admin")
	ctx.Request.Header.SetHost("example.test")

	fwd.Forward(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
	if handlerCalled {
		t.Error("app handler ran for an unauthenticated forward")
	}
}
