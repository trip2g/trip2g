package appreq

import (
	"context"
	"errors"
	"sync"
	"trip2g/internal/usertoken"

	"github.com/valyala/fasthttp"
)

type ctxKeyT struct{}

//nolint:gochecknoglobals // it's a common pattern.
var ctxKey = ctxKeyT{}

var ErrNotFound = errors.New("appenv: not found")
var ErrInvalidType = errors.New("appenv: invalid type")

type Request struct {
	sync.Mutex

	Env interface{}
	Req *fasthttp.RequestCtx

	TokenManager *usertoken.Manager

	token *usertoken.Data

	tokenExtracted bool
}

func (c *Request) Reset() {
	c.Env = nil
	c.Req = nil
	c.TokenManager = nil
	c.token = nil
	c.tokenExtracted = false
}

//nolint:gochecknoglobals // it's a common pattern.
var ctxPool = &sync.Pool{
	New: func() any {
		return &Request{}
	},
}

func Acquire() *Request {
	return ctxPool.Get().(*Request)
}

func Release(c *Request) {
	c.Reset()
	ctxPool.Put(c)
}

func GetEnv[T any](ctx context.Context) (T, error) {
	var zero T

	v := ctx.Value(ctxKey)
	if v == nil {
		return zero, ErrNotFound
	}

	reqCtx, ok := v.(*Request)
	if !ok {
		return zero, ErrInvalidType
	}

	val, ok := reqCtx.Env.(T)
	if !ok {
		return zero, ErrInvalidType
	}

	return val, nil
}

// GetToken extracts the token from the request context.
// Can returns nil, nil if the token is not present.
func GetToken(ctx context.Context) (*usertoken.Data, error) {
	v := ctx.Value(ctxKey)
	if v == nil {
		return nil, ErrNotFound
	}

	reqCtx, ok := v.(*Request)
	if !ok {
		return nil, ErrInvalidType
	}

	reqCtx.Lock()
	defer reqCtx.Unlock()

	if reqCtx.tokenExtracted {
		return reqCtx.token, nil
	}

	token, err := reqCtx.TokenManager.Extract(reqCtx.Req)
	if err != nil && !errors.Is(err, usertoken.ErrTokenMissing) {
		return nil, err
	}

	reqCtx.token = token
	reqCtx.tokenExtracted = true

	return token, nil
}

func Set(ctx context.Context, value *Request) context.Context {
	return context.WithValue(ctx, ctxKey, value)
}
