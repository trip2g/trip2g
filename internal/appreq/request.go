package appreq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"trip2g/internal/personaltoken"
	"trip2g/internal/usertoken"

	"github.com/valyala/fasthttp"
)

var ErrNotFound = errors.New("appreq: not found in context")
var ErrInvalidType = errors.New("appreq: invalid type")
var ErrInvalidEnv = errors.New("appreq: invalid env")

// PersonalTokenResolver resolves a plaintext personal token (t2g_*) to user data.
type PersonalTokenResolver interface {
	Resolve(ctx context.Context, plaintext string) (*usertoken.Data, error)
}

type ctxKeyW struct{}

var ctxKey = &ctxKeyW{} //nolint:gochecknoglobals // it's a common pattern.

type Request struct {
	mu sync.Mutex

	Env interface{}
	Req *fasthttp.RequestCtx

	Path string

	TokenManager          *usertoken.Manager
	PersonalTokenResolver PersonalTokenResolver

	token *usertoken.Data

	tokenExtracted bool

	// Webhook auth data (for shortapitoken Bearer tokens).
	WebhookDepth         int
	WebhookReadPatterns  []string
	WebhookWritePatterns []string

	// SkipWebhooks indicates this API key should not trigger webhooks.
	SkipWebhooks bool
}

func (c *Request) Reset() {
	c.Env = nil
	c.Req = nil
	c.TokenManager = nil
	c.PersonalTokenResolver = nil
	c.token = nil
	c.Path = ""
	c.tokenExtracted = false
	c.WebhookDepth = 0
	c.WebhookReadPatterns = nil
	c.WebhookWritePatterns = nil
	c.SkipWebhooks = false
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

// WithAdminToken returns a context with a copy of the current appreq where
// UserToken is pre-set to an admin token. Used for internal GraphQL calls.
func WithAdminToken(ctx context.Context) context.Context {
	req, err := FromCtx(ctx)
	if err != nil {
		return ctx
	}
	adminReq := &Request{
		Req:          req.Req,
		Env:          req.Env,
		TokenManager: req.TokenManager,
	}
	adminReq.SetUserToken(&usertoken.Data{Role: "admin"})
	return context.WithValue(ctx, ctxKey, adminReq)
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

	// 1. Try cookie first.
	token, err := c.TokenManager.Extract(c.Req)
	if err != nil && !errors.Is(err, usertoken.ErrTokenMissing) {
		return nil, err
	}
	if token != nil {
		c.token = token
		c.tokenExtracted = true
		return token, nil
	}

	// 2. Try Authorization: Bearer <value> where value starts with t2g_.
	bearer := string(c.Req.Request.Header.Peek("Authorization"))
	if value, ok := strings.CutPrefix(bearer, "Bearer "); ok && personaltoken.IsPersonal(value) {
		data, resolveErr := c.resolvePersonalToken(value)
		if resolveErr != nil {
			return nil, resolveErr
		}
		c.token = data
		c.tokenExtracted = true
		return data, nil
		// Non-t2g_ Bearer (e.g. federation JWT) falls through to anonymous.
	}

	// 3. Try ?token=<value> where value starts with t2g_.
	if qtoken := string(c.Req.QueryArgs().Peek("token")); personaltoken.IsPersonal(qtoken) {
		data, resolveErr := c.resolvePersonalToken(qtoken)
		if resolveErr != nil {
			return nil, resolveErr
		}
		c.token = data
		c.tokenExtracted = true
		return data, nil
	}

	// 4. Anonymous.
	c.token = nil
	c.tokenExtracted = true
	return nil, nil
}

func (c *Request) resolvePersonalToken(plaintext string) (*usertoken.Data, error) {
	if c.PersonalTokenResolver == nil {
		return nil, errors.New("personal token resolver not configured")
	}
	return c.PersonalTokenResolver.Resolve(c.Req, plaintext)
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
