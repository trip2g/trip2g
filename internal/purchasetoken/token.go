package purchasetoken

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"trip2g/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

type fullData struct {
	model.PurchaseToken
	jwt.RegisteredClaims
}

type Manager struct {
	secret     []byte
	cookieName string
}

var _ jwt.ClaimsValidator = (*fullData)(nil)

var ErrInvalidToken = errors.New("invalid token")

const tokenDelimiter = "|||"

// Optional: custom validation logic (e.g., prevent old tokens).
func (c fullData) Validate() error {
	return nil
}

func NewManager(cookieName string, secret []byte) *Manager {
	return &Manager{
		secret:     secret,
		cookieName: cookieName,
	}
}

func (m *Manager) NewToken(data model.PurchaseToken) (string, error) {
	now := time.Now()
	exp := now.Add(24 * time.Hour) // TODO: change to 30 minutes

	claims := fullData{
		PurchaseToken: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(m.secret)
}

func (m *Manager) ParseToken(tokenString string) (*model.PurchaseToken, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &fullData{}, func(_ *jwt.Token) (interface{}, error) {
		return m.secret, nil
	}, jwt.WithLeeway(10*time.Second))

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := parsed.Claims.(*fullData)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	return &claims.PurchaseToken, nil
}

func (e *Manager) rawTokens(ctx *fasthttp.RequestCtx) []string {
	raw := string(ctx.Request.Header.Cookie(e.cookieName))
	if len(raw) == 0 {
		return nil
	}

	return strings.Split(raw, tokenDelimiter)
}

func (e *Manager) Extract(ctx *fasthttp.RequestCtx) ([]*model.PurchaseToken, error) {
	rawTokens := e.rawTokens(ctx)
	if len(rawTokens) == 0 {
		return nil, nil
	}

	tokens := make([]*model.PurchaseToken, 0, len(rawTokens))
	validTokens := make([]string, 0, len(rawTokens))

	for _, rawToken := range rawTokens {
		token, err := e.ParseToken(rawToken)
		if err == nil && token != nil {
			tokens = append(tokens, token)
			validTokens = append(validTokens, rawToken)
		}
	}

	e.setCookie(ctx, validTokens)

	return tokens, nil
}

func (e *Manager) Store(ctx *fasthttp.RequestCtx, data model.PurchaseToken) (string, error) {
	rawToken, err := e.NewToken(data)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	fmt.Printf("before: %+v %d\n", e.rawTokens(ctx), len(e.rawTokens(ctx)))

	e.setCookie(ctx, append(e.rawTokens(ctx), rawToken))

	return rawToken, nil
}

func (e *Manager) Delete(ctx *fasthttp.RequestCtx) error {
	e.setCookie(ctx, nil)
	return nil
}

func (e *Manager) setCookie(ctx *fasthttp.RequestCtx, tokens []string) {
	current := ctx.Request.Header.Cookie(e.cookieName)
	if len(current) == 0 && len(tokens) == 0 {
		return
	}

	now := time.Now()
	exp := now.Add(24 * time.Hour) // TODO: change to 30 minutes

	c := fasthttp.AcquireCookie()
	c.SetKey(e.cookieName)

	if len(tokens) > 0 {
		fmt.Printf("after: %+v %d\n", tokens, len(tokens))
		c.SetValue(strings.Join(tokens, tokenDelimiter))
	}

	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(true)
	c.SetExpire(exp)
	c.SetMaxAge(int(exp.Sub(now).Seconds()))

	ctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}
