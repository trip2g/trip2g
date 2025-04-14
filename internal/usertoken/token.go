package usertoken

import (
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
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

// Manager handles signing and parsing user tokens.
type Manager struct {
	cookieName string
	secret     []byte
}

var (
	ErrTokenMissing = errors.New("JWT cookie not found")
	ErrInvalidToken = errors.New("invalid or expired JWT")
)

// NewExtractor creates a new Extractor instance.
func NewManager(cookieName string, secret []byte) *Manager {
	return &Manager{
		cookieName: cookieName,
		secret:     secret,
	}
}

// Extract reads cookie, verifies JWT, parses Token.
func (e *Manager) Extract(ctx *fasthttp.RequestCtx) (*Data, error) {
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

	return &token, nil
}

// Store serializes Token as JWT and sets it as a secure httpOnly cookie.
func (e *Manager) Store(ctx *fasthttp.RequestCtx, data Data) (string, error) {
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

	ctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	return signed, nil
}

func (e *Manager) Delete(ctx *fasthttp.RequestCtx) error {
	c := fasthttp.AcquireCookie()
	c.SetKey(e.cookieName)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(true)
	c.SetExpire(fasthttp.CookieExpireDelete)

	ctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	return nil
}
