package appreq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"trip2g/internal/usertoken"

	"github.com/valyala/fasthttp"
)

var ErrNotFound = errors.New("appreq: not found in context")
var ErrInvalidType = errors.New("appreq: invalid type")
var ErrInvalidEnv = errors.New("appreq: invalid env")

type ctxKeyW struct{}

var ctxKey = &ctxKeyW{} //nolint:gochecknoglobals // it's a common pattern.

type Request struct {
	mu sync.Mutex

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

func (c *Request) StoreInContext() {
	c.Req.SetUserValue(ctxKey, c)
}

func FromCtx(ctx context.Context) (*Request, error) {
	c, ok := ctx.Value(ctxKey).(*Request)
	if !ok {
		return nil, ErrNotFound
	}

	return c, nil
}

func CtxEnv[T any](ctx context.Context, defaultValue T) T {
	req, err := FromCtx(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return defaultValue
		}

		panic(fmt.Sprintf("unexpected error: %v", err))
	}

	val, ok := req.Env.(T)
	if !ok {
		var zero T
		panic(fmt.Sprintf("req.Env(%T) not implemented: %T", req.Env, zero))
	}

	return val
}

func (c *Request) SetUserToken(token *usertoken.Data) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.token = token
	c.tokenExtracted = true
}

func (c *Request) UserToken() (*usertoken.Data, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenExtracted {
		return c.token, nil
	}

	if c.Req == nil {
		panic("appreq: request is nil")
	}

	token, err := c.TokenManager.Extract(c.Req)
	if err != nil && !errors.Is(err, usertoken.ErrTokenMissing) {
		return nil, err
	}

	c.token = token
	c.tokenExtracted = true

	return token, nil
}

//nolint:gochecknoglobals // it's a common pattern.
var ctxPool = &sync.Pool{
	New: func() any {
		return &Request{}
	},
}

func Acquire() *Request {
	return ctxPool.Get().(*Request) //nolint:errcheck // it's a common pattern for sync.Pool
}

func Release(c *Request) {
	c.Reset()
	ctxPool.Put(c)
}
