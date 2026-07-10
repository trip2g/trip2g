package oidcauth

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// BoolClaim preserves the four inbound wire states of an optional JSON
// boolean: omitted, true, false, and an explicit null. Null is present but
// invalid, so a caller cannot accidentally treat malformed provider output as
// omission. When marshaled, omitted and invalid values both encode as null;
// production uses this type to decode provider claims.
type BoolClaim struct {
	Present bool
	Valid   bool
	Value   bool
}

func (c *BoolClaim) UnmarshalJSON(data []byte) error {
	c.Present = true
	c.Valid = false
	c.Value = false
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(data, &c.Value); err != nil {
		return fmt.Errorf("boolean claim must be true, false, or null: %w", err)
	}
	c.Valid = true
	return nil
}

func (c BoolClaim) MarshalJSON() ([]byte, error) {
	if !c.Present || !c.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(c.Value)
}

type Endpoints struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

type UserInfo struct {
	Sub           string    `json:"sub"`
	Email         string    `json:"email"`
	EmailVerified BoolClaim `json:"email_verified"`
	Name          string    `json:"name"`
	Groups        []string  `json:"groups"`
}
