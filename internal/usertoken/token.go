package usertoken

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

type ctxKey int

const (
	tokenKey     ctxKey = 1
	reqCtx       ctxKey = 2
	extractorKey ctxKey = 3
)

type Data struct {
	ID     int      `json:"i"`
	Opened []string `json:"o"`
}

type fullData struct {
	ID     int      `json:"i"`
	Opened []string `json:"o"`
	jwt.RegisteredClaims
}

var _ jwt.ClaimsValidator = (*fullData)(nil)

// Optional: custom validation logic (e.g., prevent old tokens).
func (c fullData) Validate() error {
	return nil
}

func Get(ctx context.Context) *Data {
	token, ok := ctx.Value(tokenKey).(*Data)
	if !ok {
		return nil
	}

	return token
}

func Store(ctx context.Context, token Data) (string, error) {
	extractor, ok := ctx.Value(extractorKey).(*Extractor)
	if !ok {
		return "", ErrExtractorNotFound
	}

	return extractor.Store(ctx, token)
}

func Delete(ctx context.Context) error {
	extractor, ok := ctx.Value(extractorKey).(*Extractor)
	if !ok {
		return ErrExtractorNotFound
	}

	return extractor.Delete(ctx)
}

// Extractor handles signing and parsing user tokens.
type Extractor struct {
	cookieName string
	secret     []byte
}

var (
	ErrCtxNotFound  = errors.New("fasthttp context not found")
	ErrTokenMissing = errors.New("JWT cookie not found")
	ErrInvalidToken = errors.New("invalid or expired JWT")

	ErrExtractorNotFound = errors.New("extractor not found in context")
)

// NewExtractor creates a new Extractor instance.
func NewExtractor(cookieName string, secret []byte) *Extractor {
	return &Extractor{
		cookieName: cookieName,
		secret:     secret,
	}
}

// Extract reads cookie, verifies JWT, parses Token.
func (e *Extractor) Extract(ctx *fasthttp.RequestCtx) (*Data, error) {
	ctx.SetUserValue(reqCtx, ctx)
	ctx.SetUserValue(extractorKey, e)

	raw := ctx.Request.Header.Cookie(e.cookieName)
	if len(raw) == 0 {
		return nil, ErrTokenMissing
	}

	parsed, err := jwt.ParseWithClaims(string(raw), &fullData{}, func(_ *jwt.Token) (interface{}, error) {
		return e.secret, nil
	}, jwt.WithLeeway(10*time.Second))
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*fullData)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	token := Data{
		ID:     claims.ID,
		Opened: claims.Opened,
	}

	ctx.SetUserValue(tokenKey, &token)

	return &token, nil
}

// Store serializes Token as JWT and sets it as a secure httpOnly cookie.
func (e *Extractor) Store(ctx context.Context, data Data) (string, error) {
	fctx, ok := ctx.Value(reqCtx).(*fasthttp.RequestCtx)
	if !ok {
		return "", ErrCtxNotFound
	}

	now := time.Now()
	exp := now.Add(24 * time.Hour)

	claims := fullData{
		ID:     data.ID,
		Opened: data.Opened,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(e.secret)
	if err != nil {
		return "", err
	}

	c := fasthttp.AcquireCookie()
	c.SetKey(e.cookieName)
	c.SetValue(signed)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(true)
	c.SetExpire(exp)
	c.SetMaxAge(int(exp.Sub(now).Seconds()))

	fctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	return signed, nil
}

func (e *Extractor) Delete(ctx context.Context) error {
	fctx, ok := ctx.Value(reqCtx).(*fasthttp.RequestCtx)
	if !ok {
		return ErrCtxNotFound
	}

	c := fasthttp.AcquireCookie()
	c.SetKey(e.cookieName)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(true)
	c.SetExpire(fasthttp.CookieExpireDelete)

	fctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	return nil
}
