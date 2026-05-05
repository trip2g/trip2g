package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"trip2g/internal/db"
)

var (
	ErrFedAuthUnknownKid = errors.New("federation auth unknown kid")
	ErrFedAuthBadSig     = errors.New("federation auth bad signature")
	ErrFedAuthExpired    = errors.New("federation auth expired")
	ErrFedAuthFutureIAT  = errors.New("federation auth future iat")
	ErrFedAuthRevoked    = errors.New("federation auth revoked")
)

const federationJWTSkew = 5 * time.Second

type federationVerifyEnv interface {
	FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error)
	ListFederationSecretSubgraphsByKID(ctx context.Context, kid string) ([]string, error)
	DecryptData([]byte) ([]byte, error)
}

type federationJWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

type federationJWTClaims struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Rid string `json:"rid"`
}

func signOutbound(secret []byte, kid, iss, rid string) (string, error) {
	return signOutboundAt(secret, kid, iss, rid, time.Now(), 30*time.Second)
}

func signOutboundAt(secret []byte, kid, iss, rid string, issuedAt time.Time, ttl time.Duration) (string, error) {
	header := federationJWTHeader{
		Alg: "HS256",
		Typ: "JWT",
		Kid: kid,
	}
	claims := federationJWTClaims{
		Iss: iss,
		Iat: issuedAt.Unix(),
		Exp: issuedAt.Add(ttl).Unix(),
		Rid: rid,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := hmacSHA256(secret, []byte(unsigned))

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func verifyInbound(ctx context.Context, env federationVerifyEnv, token string) (string, []string, error) {
	header, claims, unsigned, signature, err := parseFederationJWT(token)
	if err != nil {
		return "", nil, err
	}
	if header.Alg != "HS256" || header.Kid == "" {
		return "", nil, ErrFedAuthBadSig
	}

	secretRow, ok, err := env.FederationSecretByKID(ctx, header.Kid)
	if err != nil {
		return "", nil, fmt.Errorf("get federation secret by kid: %w", err)
	}
	if !ok {
		return "", nil, ErrFedAuthUnknownKid
	}
	if secretRow.RevokedAt != nil {
		return "", nil, ErrFedAuthRevoked
	}

	secret, err := env.DecryptData(secretRow.SecretCrypt)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt federation secret: %w", err)
	}

	expected := hmacSHA256(secret, []byte(unsigned))
	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return "", nil, ErrFedAuthBadSig
	}

	now := time.Now().Unix()
	if claims.Exp < now-int64(federationJWTSkew.Seconds()) {
		return "", nil, ErrFedAuthExpired
	}
	if claims.Iat > now+int64(federationJWTSkew.Seconds()) {
		return "", nil, ErrFedAuthFutureIAT
	}

	allowedSubgraphs, err := env.ListFederationSecretSubgraphsByKID(ctx, header.Kid)
	if err != nil {
		return "", nil, fmt.Errorf("list federation secret subgraphs by kid: %w", err)
	}

	return header.Kid, allowedSubgraphs, nil
}

func parseFederationJWT(token string) (federationJWTHeader, federationJWTClaims, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}

	var header federationJWTHeader
	if e := json.Unmarshal(headerBytes, &header); e != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}
	var claims federationJWTClaims
	if e := json.Unmarshal(claimsBytes, &claims); e != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}

	return header, claims, parts[0] + "." + parts[1], signature, nil
}

func hmacSHA256(secret, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return mac.Sum(nil)
}

func splitKBID(id string) (head, rest string) {
	head, rest, _ = strings.Cut(id, "/")
	return head, rest
}

func prefixKBID(localSegment string, items []SearchResultItem) {
	for i := range items {
		if items[i].Federation == nil || items[i].Federation.KBID == "" {
			continue
		}
		items[i].Federation.KBID = localSegment + "/" + items[i].Federation.KBID
	}
}
