package tgtd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	tdsession "github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// AuthState represents the current state of authentication.
type AuthState string

const (
	AuthStateWaitingForCode     AuthState = "WAITING_FOR_CODE"
	AuthStateWaitingForPassword AuthState = "WAITING_FOR_PASSWORD"
	AuthStateAuthorized         AuthState = "AUTHORIZED"
	AuthStateError              AuthState = "ERROR"
)

// PendingAuth holds the state of an ongoing authentication.
type PendingAuth struct {
	Phone        string
	State        AuthState
	PasswordHint string
	ExpiresAt    time.Time

	apiID        int
	apiHash      string
	client       *telegram.Client
	storage      *tdsession.StorageMemory
	sentCodeHash string
	ctx          context.Context
	cancel       context.CancelFunc
	ready        chan struct{}
	err          error
}

// AuthResult contains the result of a successful authentication.
type AuthResult struct {
	SessionData []byte
	User        *tg.User
	DisplayName string
	IsPremium   bool
}

// AuthManager manages Telegram authentication flows.
type AuthManager struct {
	mu      sync.Mutex
	pending map[string]*PendingAuth
}

// NewAuthManager creates a new AuthManager.
func NewAuthManager() *AuthManager {
	m := &AuthManager{
		pending: make(map[string]*PendingAuth),
	}
	go m.cleanupLoop()
	return m
}

// StartAuth initiates authentication for a phone number.
func (m *AuthManager) StartAuth(ctx context.Context, phone string, apiID int, apiHash string) (*PendingAuth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if there's already a pending auth for this phone
	if existing, ok := m.pending[phone]; ok {
		// If it's still valid, return it
		if time.Now().Before(existing.ExpiresAt) {
			return existing, nil
		}
		// Otherwise clean it up
		existing.cancel()
		delete(m.pending, phone)
	}

	// Create new storage for this session
	storage := &tdsession.StorageMemory{}

	// Create client
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: storage,
	})

	// Create context with cancel
	authCtx, cancel := context.WithCancel(context.Background())

	pending := &PendingAuth{
		Phone:     phone,
		State:     AuthStateWaitingForCode,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		apiID:     apiID,
		apiHash:   apiHash,
		client:    client,
		storage:   storage,
		ctx:       authCtx,
		cancel:    cancel,
		ready:     make(chan struct{}),
	}

	// Start client in background
	go func() {
		err := client.Run(authCtx, func(ctx context.Context) error {
			// Signal that client is ready
			close(pending.ready)

			// Wait for context cancellation
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			pending.err = err
		}
	}()

	// Wait for client to be ready
	select {
	case <-pending.ready:
	case <-time.After(10 * time.Second):
		cancel()
		return nil, errors.New("timeout waiting for client to start")
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}

	// Send code
	sentCode, err := client.Auth().SendCode(authCtx, phone, auth.SendCodeOptions{})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to send code: %w", err)
	}

	// Type assert to get PhoneCodeHash
	switch sc := sentCode.(type) {
	case *tg.AuthSentCode:
		pending.sentCodeHash = sc.PhoneCodeHash
	case *tg.AuthSentCodeSuccess:
		// Already authorized, shouldn't happen for new auth
		cancel()
		return nil, errors.New("already authorized")
	default:
		cancel()
		return nil, fmt.Errorf("unexpected sent code type: %T", sentCode)
	}

	m.pending[phone] = pending
	return pending, nil
}

// CompleteAuth completes authentication with code and optional password.
func (m *AuthManager) CompleteAuth(ctx context.Context, phone, code, password string) (*AuthResult, error) {
	m.mu.Lock()
	pending, ok := m.pending[phone]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("no pending authentication for phone %s", phone)
	}
	m.mu.Unlock()

	if time.Now().After(pending.ExpiresAt) {
		m.cancelAndRemove(phone)
		return nil, errors.New("authentication expired")
	}

	// If we're waiting for password and password is provided, submit it directly
	if pending.State == AuthStateWaitingForPassword && password != "" {
		_, err := pending.client.Auth().Password(pending.ctx, password)
		if err != nil {
			return nil, fmt.Errorf("invalid password: %w", err)
		}
	} else {
		// Try to sign in with code
		_, err := pending.client.Auth().SignIn(pending.ctx, phone, code, pending.sentCodeHash)
		if err != nil {
			// Check if sign up is required
			var signUpErr *auth.SignUpRequired
			if errors.As(err, &signUpErr) {
				return nil, errors.New("sign up required, but not supported")
			}

			// Check if password is required
			if errors.Is(err, auth.ErrPasswordAuthNeeded) {
				// Need 2FA password
				if password == "" {
					// Get password hint
					pwdInfo, pwdErr := pending.client.API().AccountGetPassword(pending.ctx)
					if pwdErr != nil {
						return nil, fmt.Errorf("failed to get password info: %w", pwdErr)
					}

					pending.State = AuthStateWaitingForPassword
					pending.PasswordHint = pwdInfo.Hint

					m.mu.Lock()
					m.pending[phone] = pending
					m.mu.Unlock()

					return nil, errors.New("2FA password required")
				}

				// Submit password
				_, err = pending.client.Auth().Password(pending.ctx, password)
				if err != nil {
					return nil, fmt.Errorf("invalid password: %w", err)
				}
			} else {
				return nil, fmt.Errorf("sign in failed: %w", err)
			}
		}
	}

	// Get user info
	self, err := pending.client.Self(pending.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Export session data
	sessionData, err := pending.storage.Bytes(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to export session: %w", err)
	}

	// Build display name
	displayName := buildDisplayName(self)

	// Check premium status
	isPremium := self.Premium

	// Clean up
	m.cancelAndRemove(phone)

	return &AuthResult{
		SessionData: sessionData,
		User:        self,
		DisplayName: displayName,
		IsPremium:   isPremium,
	}, nil
}

// CancelAuth cancels a pending authentication.
func (m *AuthManager) CancelAuth(phone string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pending, ok := m.pending[phone]
	if !ok {
		return fmt.Errorf("no pending authentication for phone %s", phone)
	}

	pending.cancel()
	delete(m.pending, phone)
	return nil
}

// GetPendingAuth returns the pending auth state for a phone.
func (m *AuthManager) GetPendingAuth(phone string) *PendingAuth {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pending[phone]
}

// GetPendingAuthAPICredentials returns the API credentials for a pending auth.
func (m *AuthManager) GetPendingAuthAPICredentials(phone string) (int, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pending, exists := m.pending[phone]
	if !exists {
		return 0, "", false
	}
	return pending.apiID, pending.apiHash, true
}

func (m *AuthManager) cancelAndRemove(phone string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pending, ok := m.pending[phone]; ok {
		pending.cancel()
		delete(m.pending, phone)
	}
}

func (m *AuthManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for phone, pendingAuth := range m.pending {
			if now.After(pendingAuth.ExpiresAt) {
				pendingAuth.cancel()
				delete(m.pending, phone)
			}
		}
		m.mu.Unlock()
	}
}

func buildDisplayName(user *tg.User) string {
	var parts []string
	if user.FirstName != "" {
		parts = append(parts, user.FirstName)
	}
	if user.LastName != "" {
		parts = append(parts, user.LastName)
	}
	if user.Username != "" {
		parts = append(parts, "@"+user.Username)
	}
	return strings.Join(parts, " ")
}
