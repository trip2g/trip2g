package usertoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

type Data struct {
	ID   int    `json:"i"`
	Role string `json:"r"`
}

type fullData struct {
	ID   int    `json:"i"`
	Role string `json:"r"`
	jwt.RegisteredClaims
}

type ValidateError struct {
	Value error
}

func (err *ValidateError) Error() string {
	return fmt.Sprintf("validation error: %v", err.Value)
}

type Validator func(ctx context.Context, data *Data) error

var _ jwt.ClaimsValidator = (*fullData)(nil)

// Optional: custom validation logic (e.g., prevent old tokens).
func (c fullData) Validate() error {
	return nil
}

// Manager handles signing and parsing user tokens.
type Manager struct {
	cookieName string
	secret     []byte

	// will call this function after parsing the token
	// for example for check banned users.
	validators []Validator
}

var (
	ErrTokenMissing = errors.New("JWT cookie not found")
	ErrInvalidToken = errors.New("invalid or expired JWT")
)

// NewManager creates a new Manager instance.
func NewManager(cookieName string, secret []byte) *Manager {
	return &Manager{
		cookieName: cookieName,
		secret:     secret,
	}
}

func (e *Manager) AddValidator(v Validator) {
	e.validators = append(e.validators, v)
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
		if errors.Is(err, jwt.ErrTokenExpired) {
			deleteErr := e.Delete(ctx)
			if deleteErr != nil {
				return nil, fmt.Errorf("failed to delete expired token: %w", deleteErr)
			}

			return nil, nil
		}

		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*fullData)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	token := Data{
		ID:   claims.ID,
		Role: claims.Role,
	}

	for _, v := range e.validators {
		validationErr := v(ctx, &token)
		if validationErr != nil {
			return nil, &ValidateError{Value: validationErr}
		}
	}

	return &token, nil
}

type StoreData struct {
	JWT  string
	Data Data
}

// Store serializes Token as JWT and sets it as a secure httpOnly cookie.
func (e *Manager) Store(ctx *fasthttp.RequestCtx, data Data) (*StoreData, error) {
	now := time.Now()
	exp := now.Add(24 * time.Hour)

	claims := fullData{
		ID:   data.ID,
		Role: data.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(e.secret)
	if err != nil {
		return nil, err
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

	storeData := StoreData{
		JWT:  signed,
		Data: data,
	}

	return &storeData, nil
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
