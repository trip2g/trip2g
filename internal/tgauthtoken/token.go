package tgauthtoken

import (
	"errors"
	"fmt"
	"time"
	"trip2g/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

type fullData struct {
	model.TgAuthToken
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
}

var _ jwt.ClaimsValidator = (*fullData)(nil)

var ErrInvalidToken = errors.New("invalid token")

// Optional: custom validation logic (e.g., prevent old tokens).
func (c fullData) Validate() error {
	return nil
}

func NewManager(secret []byte) *Manager {
	return &Manager{
		secret: secret,
	}
}

func (m *Manager) NewToken(data model.TgAuthToken) (string, error) {
	now := time.Now()
	exp := now.Add(5 * time.Minute)

	claims := fullData{
		TgAuthToken: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(m.secret)
}

func (m *Manager) ParseToken(tokenString string) (*model.TgAuthToken, error) {
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

	data := claims.TgAuthToken

	return &data, nil
}
