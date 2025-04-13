package appreq

import (
	"context"
	"errors"
	"sync"
	"trip2g/internal/usertoken"

	"github.com/valyala/fasthttp"
)

var ErrNotFound = errors.New("appreq: not found in context")
var ErrInvalidType = errors.New("appreq: invalid type")

type ctxKeyW struct{}

var ctxKey = &ctxKeyW{}

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

func (c *Request) UserToken() (*usertoken.Data, error) {
	c.Lock()
	defer c.Unlock()

	if c.tokenExtracted {
		return c.token, nil
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
	return ctxPool.Get().(*Request)
}

func Release(c *Request) {
	c.Reset()
	ctxPool.Put(c)
}
