package usertoken

import (
	"context"
	"errors"
	"time"
	"trip2g/internal/fasthttpctx"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

type ctxKey struct{}

var tokenKey = ctxKey{}
var extractorKey = ctxKey{}

type Token struct {
	ID     int      `json:"i"`
	Opened []string `json:"o"`
}

type Claims struct {
	ID     int      `json:"i"`
	Opened []string `json:"o"`
	jwt.RegisteredClaims
}

var _ jwt.ClaimsValidator = (*Claims)(nil)

// Optional: custom validation logic (e.g., prevent old tokens)
func (c Claims) Validate() error {
	return nil
}

func Get(ctx context.Context) *Token {
	fctx, ok := fasthttpctx.Get(ctx)
	if !ok {
		return nil
	}

	token, ok := fctx.UserValue(tokenKey).(*Token)
	if !ok {
		return nil
	}

	return token
}

func Store(ctx context.Context, token Token) error {
	fctx, ok := fasthttpctx.Get(ctx)
	if !ok {
		return ErrCtxNotFound
	}

	extractor, ok := fctx.UserValue(extractorKey).(*Extractor)
	if !ok {
		return ErrCtxNotFound
	}

	return extractor.Store(ctx, token)
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
)

// NewExtractor creates a new Extractor instance.
func NewExtractor(cookieName string, secret []byte) *Extractor {
	return &Extractor{
		cookieName: cookieName,
		secret:     secret,
	}
}

// Extract reads cookie, verifies JWT, parses Token.
func (e *Extractor) Extract(ctx *fasthttp.RequestCtx) error {
	raw := ctx.Request.Header.Cookie(e.cookieName)
	if len(raw) == 0 {
		return nil
	}

	parsed, err := jwt.ParseWithClaims(string(raw), &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return e.secret, nil
	}, jwt.WithLeeway(10*time.Second))
	if err != nil {
		return ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return ErrInvalidToken
	}

	token := &Token{
		ID:     claims.ID,
		Opened: claims.Opened,
	}

	ctx.SetUserValue(tokenKey, &token)
	ctx.SetUserValue(extractorKey, e)

	return nil
}

// Store serializes Token as JWT and sets it as a secure httpOnly cookie.
func (e *Extractor) Store(ctx context.Context, data Token) error {
	fctx, ok := fasthttpctx.Get(ctx)
	if !ok {
		return ErrCtxNotFound
	}

	now := time.Now()
	exp := now.Add(24 * time.Hour)

	claims := Claims{
		ID:     data.ID,
		Opened: data.Opened,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(e.secret)
	if err != nil {
		return err
	}

	c := fasthttp.Cookie{}
	c.SetKey(e.cookieName)
	c.SetValue(signed)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(true)
	c.SetExpire(exp)
	c.SetMaxAge(int(exp.Sub(now).Seconds()))

	fctx.Response.Header.SetCookie(&c)
	return nil
}
