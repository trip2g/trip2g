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
	ID   int    `json:"i,omitzero"`
	Role string `json:"r"`
}

func (d *Data) IsAdmin() bool {
	return d != nil && d.Role == "admin"
}

// RoleFederated marks a non-admin, subgraph-scoped federation identity.
// It must NEVER satisfy IsAdmin().
const RoleFederated = "federated"

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

type Config struct {
	CookieName string
	Secret     string
	ExpiresIn  time.Duration
	Insecure   bool
}

// Manager handles signing and parsing user tokens.
type Manager struct {
	config Config

	secret   []byte
	insecure bool

	// will call this function after parsing the token
	// for example for check banned users.
	validators []Validator
}

var (
	ErrTokenMissing = errors.New("JWT cookie not found")
	ErrInvalidToken = errors.New("invalid or expired JWT")
)

func DefaultConfig() Config {
	return Config{
		CookieName: "trip2g_token",
		Secret:     "please-change-me-to-something-secret",
		ExpiresIn:  30 * 24 * time.Hour,
	}
}

// NewManager creates a new Manager instance.
func NewManager(config Config) *Manager {
	return &Manager{
		config:   config,
		secret:   []byte(config.Secret),
		insecure: config.Insecure,
	}
}

func (e *Manager) SetInsecure(v bool) {
	e.insecure = v
}

func (e *Manager) AddValidator(v Validator) {
	e.validators = append(e.validators, v)
}

// parseClaims verifies a raw session-token JWT and returns its Data, or an
// error if the token is malformed/expired/invalid-signature.
func (e *Manager) parseClaims(raw string) (*Data, error) {
	parsed, err := jwt.ParseWithClaims(raw, &fullData{}, func(_ *jwt.Token) (interface{}, error) {
		return e.secret, nil
	}, jwt.WithLeeway(10*time.Second))
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*fullData)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}

	return &Data{ID: claims.ID, Role: claims.Role}, nil
}

func (e *Manager) runValidators(ctx context.Context, token *Data) error {
	for _, v := range e.validators {
		if validationErr := v(ctx, token); validationErr != nil {
			return &ValidateError{Value: validationErr}
		}
	}
	return nil
}

// Extract reads the session cookie, verifies the JWT, and parses Token.
// An invalid/expired cookie is cleared and treated as anonymous (nil, nil).
func (e *Manager) Extract(ctx *fasthttp.RequestCtx) (*Data, error) {
	raw := ctx.Request.Header.Cookie(e.config.CookieName)
	if len(raw) == 0 {
		return nil, ErrTokenMissing
	}

	token, err := e.parseClaims(string(raw))
	if err != nil {
		// Any JWT parsing error (expired, invalid signature, malformed, etc.)
		// should clear the cookie and treat as anonymous user.
		if deleteErr := e.Delete(ctx); deleteErr != nil {
			return nil, fmt.Errorf("failed to delete invalid token: %w", deleteErr)
		}
		return nil, nil
	}

	if validationErr := e.runValidators(ctx, token); validationErr != nil {
		return nil, validationErr
	}

	return token, nil
}

// ParseToken verifies a raw session-token string (e.g. from an
// Authorization: Bearer header) and runs validators. Unlike Extract it has no
// cookie to clear; an invalid/expired token yields (nil, nil) → anonymous.
func (e *Manager) ParseToken(ctx context.Context, raw string) (*Data, error) {
	token, err := e.parseClaims(raw)
	if err != nil {
		return nil, nil
	}

	if validationErr := e.runValidators(ctx, token); validationErr != nil {
		return nil, validationErr
	}

	return token, nil
}

type StoreData struct {
	JWT  string
	Data Data
}

// Store serializes Token as JWT and sets it as a secure httpOnly cookie.
func (e *Manager) Store(ctx *fasthttp.RequestCtx, data Data) (*StoreData, error) {
	now := time.Now()
	exp := now.Add(e.config.ExpiresIn)

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
	c.SetKey(e.config.CookieName)
	c.SetValue(signed)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(!e.insecure)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
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
	c.SetKey(e.config.CookieName)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(!e.insecure)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	c.SetExpire(fasthttp.CookieExpireDelete)

	ctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	return nil
}
