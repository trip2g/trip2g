package hotauthtoken

import (
	"errors"
	"fmt"
	"time"
	"trip2g/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

type fullData struct {
	model.HotAuthToken
	jwt.RegisteredClaims
}

type Config struct {
	Secret    string
	ExpiresIn time.Duration
}

type Manager struct {
	config Config
	secret []byte
}

var _ jwt.ClaimsValidator = (*fullData)(nil)

var ErrInvalidToken = errors.New("invalid token")

// Optional: custom validation logic (e.g., prevent old tokens).
func (c fullData) Validate() error {
	return nil
}

func DefaultConfig() Config {
	return Config{
		Secret:    "change_me_please",
		ExpiresIn: 5 * time.Minute,
	}
}

func NewManager(config Config) *Manager {
	return &Manager{
		secret: []byte(config.Secret),
		config: config,
	}
}

func (m *Manager) NewToken(data model.HotAuthToken) (string, error) {
	now := time.Now()
	exp := now.Add(m.config.ExpiresIn)

	claims := fullData{
		HotAuthToken: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(m.secret)
}

func (m *Manager) ParseToken(tokenString string) (*model.HotAuthToken, error) {
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

	hotAuthToken := claims.HotAuthToken

	return &hotAuthToken, nil
}
