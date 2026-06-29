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

	// WebhookScoped is true when this request authenticated via a scoped
	// shortapitoken. It distinguishes "scoped to read/write patterns" (enforce,
	// empty patterns = deny-all) from "not scoped at all" (admin/user/api-key
	// — no webhook-pattern enforcement). Set by checkapikey on the shortapitoken
	// path and by stampShortAPIToken in the AroundOperations middleware.
	WebhookScoped bool

	// Webhook delivery identity (from the scoped shortapitoken). Used to
	// attribute note versions written by a fleet back to the delivery.
	WebhookDeliveryKind string
	WebhookDeliveryID   int64

	// SkipWebhooks indicates this API key should not trigger webhooks.
	SkipWebhooks bool

	// AdminActorUserID is the user id that internal admin GraphQL calls
	// (see WithAdminToken) should be attributed to — e.g. the owner of the
	// authenticating MCP API key. Zero when the actor is unknown.
	AdminActorUserID int

	// APIKeyID is the id of the API key that authenticated this request, set by
	// checkapikey for real (DB-backed) keys. nil when the request was not
	// api-key-authed (user session, admin bypass, or virtual webhook token).
	// Used to record who pushed a note version (created_by_api_key_id).
	APIKeyID *int64

	// Client is the trimmed value of the X-trip2g-client request header, set
	// when the request is built from the fasthttp context. Empty string when
	// the header is absent. Used to record which client pushed a note version
	// (created_by_client).
	Client string

	// FederatedSubgraphs carries the inbound federation identity's AllowedSubgraphs
	// when the token Role is usertoken.RoleFederated. nil for non-federated requests.
	FederatedSubgraphs []string
	federatedScoped    bool
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
	c.WebhookScoped = false
	c.WebhookDeliveryKind = ""
	c.WebhookDeliveryID = 0
	c.SkipWebhooks = false
	c.AdminActorUserID = 0
	c.APIKeyID = nil
	c.Client = ""
	c.FederatedSubgraphs = nil
	c.federatedScoped = false
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
// UserToken is pre-set to an admin token. Used for internal GraphQL calls
// (e.g. MCP graphql_request). The token's user id is taken from
// AdminActorUserID so mutations that record the acting admin (e.g. createAdmin
// writing granted_by) reference a real admins row instead of user id 0.
//
// TODO(refactor): this synthetic-request copying is hacky — it duplicates a
// subset of Request fields by hand and silently no-ops when ctx has no appreq.
// Internal GraphQL calls should carry a proper acting-user identity instead of
// forging an "admin" token here.
func WithAdminToken(ctx context.Context) context.Context {
	req, err := FromCtx(ctx)
	if err != nil {
		return ctx
	}
	adminReq := &Request{
		Req:              req.Req,
		Env:              req.Env,
		TokenManager:     req.TokenManager,
		AdminActorUserID: req.AdminActorUserID,
	}
	adminReq.SetUserToken(&usertoken.Data{ID: req.AdminActorUserID, Role: "admin"})
	return context.WithValue(ctx, ctxKey, adminReq)
}

// WithFederatedToken returns a context whose appreq carries a non-admin,
// subgraph-scoped federation identity. canreadnote.Resolve detects this and
// gates reads via ResolveWithSubgraphs(allowedSubgraphs) with NO userID DB lookup.
// allowedSubgraphs nil/empty => Free-only (fail-safe), never admin.
func WithFederatedToken(ctx context.Context, allowedSubgraphs []string) context.Context {
	req, err := FromCtx(ctx)
	if err != nil {
		// No appreq in ctx: cannot scope safely. Return ctx unchanged so the
		// downstream resolver sees an anonymous (nil) token => Free-only.
		return ctx
	}
	scopedReq := &Request{
		Req:                req.Req,
		Env:                req.Env,
		TokenManager:       req.TokenManager,
		FederatedSubgraphs: append([]string{}, allowedSubgraphs...), // non-nil even if empty
		federatedScoped:    true,
	}
	scopedReq.SetUserToken(&usertoken.Data{Role: usertoken.RoleFederated}) // ID=0, non-admin
	return context.WithValue(ctx, ctxKey, scopedReq)
}

// FederatedScope reports whether this request is a federated-scoped identity
// and returns its AllowedSubgraphs.
func (c *Request) FederatedScope() ([]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.FederatedSubgraphs, c.federatedScoped
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

	// 2. Try Authorization: Bearer <value>.
	bearer := string(c.Req.Request.Header.Peek("Authorization"))
	if value, ok := strings.CutPrefix(bearer, "Bearer "); ok { //nolint:nestif // auth fallback chain requires nested branching
		// 2a. Personal token (t2g_*).
		if personaltoken.IsPersonal(value) {
			data, resolveErr := c.resolvePersonalToken(value)
			if resolveErr != nil {
				return nil, resolveErr
			}
			c.token = data
			c.tokenExtracted = true
			return data, nil
		}
		// 2b. Otherwise try a session-token JWT. This is how the control plane
		// (simplepanel) reaches an instance by its private address: it harvests
		// the HAT-issued session token and re-sends it as a Bearer (the Secure
		// session cookie can't be replayed over plain http). Invalid token →
		// fall through to anonymous.
		if c.TokenManager != nil {
			data, parseErr := c.TokenManager.ParseToken(c.Req, value)
			if parseErr != nil {
				return nil, parseErr
			}
			if data != nil {
				c.token = data
				c.tokenExtracted = true
				return data, nil
			}
		}
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

// NewContext returns a copy of parent with req stored as the appreq value.
// Use this to pass an appreq through a standard context.Context when the
// original fasthttp.RequestCtx is no longer available (e.g. SSE streams).
func NewContext(parent context.Context, req *Request) context.Context {
	return context.WithValue(parent, ctxKey, req)
}

// Scoped reports whether the request authenticated via a scoped shortapitoken
// (so read/write-pattern enforcement applies). Returns false when ctx has no
// appreq (internal calls, tests) or for admin/user/api-key auth.
func Scoped(ctx context.Context) bool {
	req, err := FromCtx(ctx)
	if err != nil {
		return false
	}
	return req.WebhookScoped
}

// WebhookWritePatterns returns the write-scope globs stamped on the request
// by a shortapitoken delivery. Empty/nil means unscoped (no enforcement).
func WebhookWritePatterns(ctx context.Context) []string {
	req, err := FromCtx(ctx)
	if err != nil {
		return nil
	}
	return req.WebhookWritePatterns
}

// WebhookReadPatterns returns the read-scope globs stamped on the request
// by a shortapitoken delivery. Empty/nil means unscoped (no enforcement).
func WebhookReadPatterns(ctx context.Context) []string {
	req, err := FromCtx(ctx)
	if err != nil {
		return nil
	}
	return req.WebhookReadPatterns
}

// WebhookDeliveryKind returns the delivery kind ("change"/"cron") for a
// scoped-token request, or "" if none.
func WebhookDeliveryKind(ctx context.Context) string {
	req, err := FromCtx(ctx)
	if err != nil {
		return ""
	}
	return req.WebhookDeliveryKind
}

// WebhookDeliveryID returns the delivery id for a scoped-token request, or 0.
func WebhookDeliveryID(ctx context.Context) int64 {
	req, err := FromCtx(ctx)
	if err != nil {
		return 0
	}
	return req.WebhookDeliveryID
}

// Snapshot returns an independent deep copy of c suitable for use beyond the
// request lifetime. The snapshot's Req field is a fresh fasthttp.RequestCtx
// with headers, cookies, query args, and remote address copied from the
// original. Use before releasing the pooled Request (e.g. for SSE streams).
func (c *Request) Snapshot() *Request {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshotFCtx := &fasthttp.RequestCtx{}
	snapshotFCtx.Init(&c.Req.Request, c.Req.RemoteAddr(), nil)

	var readPatterns []string
	if c.WebhookReadPatterns != nil {
		readPatterns = make([]string, len(c.WebhookReadPatterns))
		copy(readPatterns, c.WebhookReadPatterns)
	}

	var writePatterns []string
	if c.WebhookWritePatterns != nil {
		writePatterns = make([]string, len(c.WebhookWritePatterns))
		copy(writePatterns, c.WebhookWritePatterns)
	}

	var federatedSubgraphs []string
	if c.FederatedSubgraphs != nil {
		federatedSubgraphs = make([]string, len(c.FederatedSubgraphs))
		copy(federatedSubgraphs, c.FederatedSubgraphs)
	}

	var apiKeyID *int64
	if c.APIKeyID != nil {
		v := *c.APIKeyID
		apiKeyID = &v
	}

	return &Request{
		Env:                   c.Env,
		Req:                   snapshotFCtx,
		Path:                  c.Path,
		TokenManager:          c.TokenManager,
		PersonalTokenResolver: c.PersonalTokenResolver,
		token:                 c.token,
		tokenExtracted:        c.tokenExtracted,
		WebhookDepth:          c.WebhookDepth,
		WebhookReadPatterns:   readPatterns,
		WebhookWritePatterns:  writePatterns,
		WebhookScoped:         c.WebhookScoped,
		WebhookDeliveryKind:   c.WebhookDeliveryKind,
		WebhookDeliveryID:     c.WebhookDeliveryID,
		SkipWebhooks:          c.SkipWebhooks,
		APIKeyID:              apiKeyID,
		Client:                c.Client,
		FederatedSubgraphs:    federatedSubgraphs,
		federatedScoped:       c.federatedScoped,
	}
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
