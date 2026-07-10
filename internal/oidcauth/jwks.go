package oidcauth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/sync/singleflight"
)

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

// KeyCache fetches and caches JWKS public keys per jwks_uri. On an unknown kid
// it refetches the JWKS so key rotation is handled transparently. Safe for
// concurrent use. State lives on the cache, not in a package-level var.
type KeyCache struct {
	mu    sync.Mutex
	byURI map[string]map[string]crypto.PublicKey // jwksURI -> kid -> key
	group singleflight.Group

	// fetch retrieves the raw JWKS document; defaults to a fasthttp GET.
	// It is a field (not a hardcoded call) so tests can drive rotation
	// deterministically without a network dependency.
	fetch func(ctx context.Context, jwksURI string) ([]byte, error)
}

func NewKeyCache() *KeyCache {
	return &KeyCache{
		byURI: map[string]map[string]crypto.PublicKey{},
		fetch: fetchJWKS,
	}
}

// keyForKID returns the public key for kid at jwksURI, fetching (or, on an
// unknown kid, refetching) the JWKS as needed.
func (c *KeyCache) keyForKID(ctx context.Context, jwksURI, kid string) (crypto.PublicKey, error) {
	c.mu.Lock()
	if keys, ok := c.byURI[jwksURI]; ok {
		if key, found := keys[kid]; found {
			c.mu.Unlock()
			return key, nil
		}
	}
	c.mu.Unlock()

	// Cache miss or unknown kid: (re)fetch and replace the cached set. The lock
	// is not held during the fetch so a slow endpoint doesn't block cached
	// reads; singleflight collapses concurrent fetches of the same URI.
	v, err, _ := c.group.Do(jwksURI, func() (any, error) {
		keys, lerr := c.load(ctx, jwksURI)
		if lerr != nil {
			return nil, lerr
		}
		c.mu.Lock()
		c.byURI[jwksURI] = keys
		c.mu.Unlock()
		return keys, nil
	})
	if err != nil {
		return nil, err
	}

	key, ok := v.(map[string]crypto.PublicKey)[kid]
	if !ok {
		return nil, fmt.Errorf("no JWKS key for kid %q", kid)
	}
	return key, nil
}

func (c *KeyCache) load(ctx context.Context, jwksURI string) (map[string]crypto.PublicKey, error) {
	body, err := c.fetch(ctx, jwksURI)
	if err != nil {
		return nil, err
	}

	var doc jwksDocument
	if uerr := json.Unmarshal(body, &doc); uerr != nil {
		return nil, fmt.Errorf("failed to unmarshal JWKS: %w", uerr)
	}

	keys := map[string]crypto.PublicKey{}
	for _, k := range doc.Keys {
		var (
			pub  crypto.PublicKey
			perr error
		)
		switch k.Kty {
		case "RSA":
			pub, perr = k.rsaPublicKey()
		case "EC":
			pub, perr = k.ecPublicKey()
		default:
			continue
		}
		if perr != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS contained no usable keys")
	}
	return keys, nil
}

func (k jwk) rsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("invalid JWKS modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("invalid JWKS exponent: %w", err)
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid JWKS exponent: zero")
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func (k jwk) ecPublicKey() (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported JWKS curve %q", k.Crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("invalid JWKS x coordinate: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid JWKS y coordinate: %w", err)
	}

	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}
	if !curve.IsOnCurve(pub.X, pub.Y) {
		return nil, errors.New("JWKS EC point not on curve")
	}
	return pub, nil
}

func fetchJWKS(ctx context.Context, jwksURI string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(jwksURI)
	req.Header.SetMethod(fasthttp.MethodGet)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	httpClient := &fasthttp.Client{NoDefaultUserAgentHeader: true}

	timeout := 10 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			timeout = remaining
		}
	}

	if err := httpClient.DoTimeout(req, resp, timeout); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("jwks fetch failed: status %d", resp.StatusCode())
	}

	// Copy the body out before the response is released to the pool.
	return append([]byte(nil), resp.Body()...), nil
}
