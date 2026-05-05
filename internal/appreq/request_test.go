package appreq

import (
	"context"
	"testing"

	"trip2g/internal/personaltoken"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// mockResolver is a test double for PersonalTokenResolver.
type mockResolver struct {
	resolveFunc func(ctx context.Context, plaintext string) (*usertoken.Data, error)
	calls       int
}

func (m *mockResolver) Resolve(ctx context.Context, plaintext string) (*usertoken.Data, error) {
	m.calls++
	return m.resolveFunc(ctx, plaintext)
}

func newFasthttpCtx() *fasthttp.RequestCtx {
	return &fasthttp.RequestCtx{}
}

func newFasthttpCtxWithBearer(bearer string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Authorization", "Bearer "+bearer)
	return ctx
}

func newFasthttpCtxWithQueryToken(token string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.QueryArgs().Set("token", token)
	return ctx
}

// validPersonalUser returned by mock resolver on success.
var validPersonalUser = &usertoken.Data{ID: 42, Role: "user"} //nolint:gochecknoglobals // test package global

// fakeJWTCookie simulates a valid cookie value; Manager.Extract will fail on it
// (no valid secret), which returns nil, nil (anonymous) — sufficient for tests
// that need "cookie absent".
const fakePersonalToken = "t2g_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab"
const fakeFederationJWT = "header.payload.signature"

func TestUserToken_CookieWinsOverBearer(t *testing.T) {
	// Case 1: Cookie valid + Bearer t2g_* valid -> cookie user wins; Resolve NOT called.
	// We use a TokenManager with insecure mode. Store a token then check Extract returns it.
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	// Store a valid cookie and get the JWT back.
	cookieUser := usertoken.Data{ID: 7, Role: "user"}
	storeData, err := mgr.Store(newFasthttpCtx(), cookieUser)
	require.NoError(t, err)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetCookie("trip2g_token", storeData.JWT)
	reqCtx.Request.Header.Set("Authorization", "Bearer "+fakePersonalToken)

	mock := &mockResolver{
		resolveFunc: func(_ context.Context, _ string) (*usertoken.Data, error) {
			return validPersonalUser, nil
		},
	}

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	req.PersonalTokenResolver = mock

	tok, err := req.UserToken()
	require.NoError(t, err)
	require.NotNil(t, tok)
	require.Equal(t, 7, tok.ID)
	require.Equal(t, 0, mock.calls, "Resolve must not be called when cookie wins")

	Release(req)
}

func TestUserToken_BearerPersonalToken(t *testing.T) {
	// Case 2: Cookie absent + Bearer t2g_* valid -> personal user returned.
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	reqCtx := newFasthttpCtxWithBearer(fakePersonalToken)
	mock := &mockResolver{
		resolveFunc: func(_ context.Context, plaintext string) (*usertoken.Data, error) {
			require.Equal(t, fakePersonalToken, plaintext)
			return validPersonalUser, nil
		},
	}

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	req.PersonalTokenResolver = mock

	tok, err := req.UserToken()
	require.NoError(t, err)
	require.Equal(t, validPersonalUser, tok)
	require.Equal(t, 1, mock.calls)

	Release(req)
}

func TestUserToken_BearerPersonalTokenInvalid(t *testing.T) {
	// Case 3: Cookie absent + Bearer t2g_* invalid -> ErrInvalidToken (hard error, NOT nil).
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	reqCtx := newFasthttpCtxWithBearer(fakePersonalToken)
	mock := &mockResolver{
		resolveFunc: func(_ context.Context, _ string) (*usertoken.Data, error) {
			return nil, personaltoken.ErrInvalidToken
		},
	}

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	req.PersonalTokenResolver = mock

	tok, err := req.UserToken()
	require.Error(t, err)
	require.Nil(t, tok)
	require.ErrorIs(t, err, personaltoken.ErrInvalidToken)

	Release(req)
}

func TestUserToken_BearerFederationJWT(t *testing.T) {
	// Case 4: Cookie absent + Bearer non-t2g_ (federation JWT format) -> nil from UserToken().
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	reqCtx := newFasthttpCtxWithBearer(fakeFederationJWT)
	mock := &mockResolver{
		resolveFunc: func(_ context.Context, _ string) (*usertoken.Data, error) {
			t.Fatal("Resolve must not be called for non-t2g_ Bearer")
			return nil, nil
		},
	}

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	req.PersonalTokenResolver = mock

	tok, err := req.UserToken()
	require.NoError(t, err)
	require.Nil(t, tok)
	require.Equal(t, 0, mock.calls)

	Release(req)
}

func TestUserToken_QueryTokenPersonal(t *testing.T) {
	// Case 5: Cookie absent + ?token=t2g_* valid -> personal user returned.
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	reqCtx := newFasthttpCtxWithQueryToken(fakePersonalToken)
	mock := &mockResolver{
		resolveFunc: func(_ context.Context, plaintext string) (*usertoken.Data, error) {
			require.Equal(t, fakePersonalToken, plaintext)
			return validPersonalUser, nil
		},
	}

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	req.PersonalTokenResolver = mock

	tok, err := req.UserToken()
	require.NoError(t, err)
	require.Equal(t, validPersonalUser, tok)
	require.Equal(t, 1, mock.calls)

	Release(req)
}

func TestUserToken_NoAuth(t *testing.T) {
	// Case 6: Cookie absent + no auth headers -> nil (anonymous).
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	reqCtx := newFasthttpCtx()
	mock := &mockResolver{
		resolveFunc: func(_ context.Context, _ string) (*usertoken.Data, error) {
			t.Fatal("Resolve must not be called when no auth present")
			return nil, nil
		},
	}

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	req.PersonalTokenResolver = mock

	tok, err := req.UserToken()
	require.NoError(t, err)
	require.Nil(t, tok)

	Release(req)
}

func TestUserToken_ResolverNotWired(t *testing.T) {
	// Case 7: PersonalTokenResolver == nil + Bearer t2g_* present -> clear error (no panic).
	mgr := usertoken.NewManager(usertoken.Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})

	reqCtx := newFasthttpCtxWithBearer(fakePersonalToken)

	req := Acquire()
	req.Req = reqCtx
	req.TokenManager = mgr
	// PersonalTokenResolver intentionally NOT set (nil)

	tok, err := req.UserToken()
	require.Error(t, err)
	require.Nil(t, tok)
	require.Contains(t, err.Error(), "personal token resolver not configured")

	Release(req)
}

func TestReset_NilsPersonalTokenResolver(t *testing.T) {
	// Reset() must nil PersonalTokenResolver field.
	req := Acquire()
	req.PersonalTokenResolver = &mockResolver{}
	req.Reset()
	require.Nil(t, req.PersonalTokenResolver)
}

// Ensure the mock satisfies the interface at compile time.
var _ PersonalTokenResolver = (*mockResolver)(nil)
