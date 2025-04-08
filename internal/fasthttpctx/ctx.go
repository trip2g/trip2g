package fasthttpctx

import (
	"context"

	"github.com/valyala/fasthttp"
)

type ctxKey struct{}

var key = ctxKey{}

func Inject(ctx *fasthttp.RequestCtx) {
	ctx.SetUserValue(key, ctx)
}

func Get(ctx context.Context) (*fasthttp.RequestCtx, bool) {
	v := ctx.Value(key)
	if v == nil {
		return nil, false
	}

	reqCtx, ok := v.(*fasthttp.RequestCtx)

	return reqCtx, ok
}
