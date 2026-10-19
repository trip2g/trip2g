package mcp_test

import (
	"testing"

	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
)

func BenchmarkToolsList(b *testing.B) {
	env := goldenEnv()
	for range b.N {
		ctx := &fasthttp.RequestCtx{}
		ctx.Init2(nil, nil, true)
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/_system/mcp")
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.Header.Set("Accept", acceptHeader)
		ctx.Request.SetBodyString(rpc(2, "tools/list", "{}"))
		req := newMCPRequest(ctx, env)
		_, err := (&mcp.Endpoint{}).Handle(req)
		if err != nil {
			b.Fatal(err)
		}
		appreq.Release(req)
	}
}

func BenchmarkToolsListParallel(b *testing.B) {
	env := goldenEnv()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := &fasthttp.RequestCtx{}
			ctx.Init2(nil, nil, true)
			ctx.Request.Header.SetMethod("POST")
			ctx.Request.SetRequestURI("/_system/mcp")
			ctx.Request.Header.SetContentType("application/json")
			ctx.Request.Header.Set("Accept", acceptHeader)
			ctx.Request.SetBodyString(rpc(2, "tools/list", "{}"))
			req := newMCPRequest(ctx, env)
			_, err := (&mcp.Endpoint{}).Handle(req)
			if err != nil {
				b.Error(err)
			}
			appreq.Release(req)
		}
	})
}
