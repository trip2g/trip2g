package fleet

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"trip2g/internal/hotauthtoken"
	"trip2g/internal/model"
)

// hatExpiry is the lifetime of a minted admin HAT; it only needs to survive the
// single /_system/hat exchange, so a short window is plenty.
const hatExpiry = 5 * time.Minute

// hatAuthenticator mints a Hot-Auth-Token from the shared JWT secret, exchanges
// it at /_system/hat for an admin session cookie, and caches that cookie for
// reuse on the raw admin GraphQL lane. Safe for concurrent use: the fleet calls
// the admin lane from the reconcile loop while deliveries run.
type hatAuthenticator struct {
	baseURL    string
	adminEmail string
	manager    *hotauthtoken.Manager
	hc         *http.Client

	mu      sync.RWMutex
	cookies []*http.Cookie
}

// newHATAuthenticator builds a hatAuthenticator. The HAT exchange must NOT
// follow the 302 redirect /_system/hat returns, otherwise the Set-Cookie is
// lost; a derived client with CheckRedirect short-circuits it while preserving
// the caller's Transport and Timeout.
func newHATAuthenticator(baseURL, jwtSecret, adminEmail string, hc *http.Client) *hatAuthenticator {
	if hc == nil {
		hc = http.DefaultClient
	}
	noRedirect := &http.Client{
		Transport: hc.Transport,
		Timeout:   hc.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &hatAuthenticator{
		baseURL:    baseURL,
		adminEmail: adminEmail,
		manager:    hotauthtoken.NewManager(hotauthtoken.Config{Secret: jwtSecret, ExpiresIn: hatExpiry}),
		hc:         noRedirect,
	}
}

// mintToken signs a short-lived admin HAT for adminEmail (AdminEnter elevates
// and self-provisions the admin user at the exchange endpoint).
func (a *hatAuthenticator) mintToken() (string, error) {
	return a.manager.NewToken(model.HotAuthToken{Email: a.adminEmail, AdminEnter: true})
}

// authenticate mints a HAT, exchanges it at /_system/hat, and caches the admin
// session cookie(s) from the response. It replaces any previously cached
// cookies on success.
func (a *hatAuthenticator) authenticate(ctx context.Context) error {
	token, err := a.mintToken()
	if err != nil {
		return fmt.Errorf("mint HAT: %w", err)
	}
	form := url.Values{"token": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/_system/hat", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return fmt.Errorf("HAT exchange set no session cookie (HTTP %d)", resp.StatusCode)
	}
	a.mu.Lock()
	a.cookies = cookies
	a.mu.Unlock()
	return nil
}

// cached returns the currently cached admin session cookie(s); empty before the
// first successful authenticate.
func (a *hatAuthenticator) cached() []*http.Cookie {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cookies
}
