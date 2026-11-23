package usertoken_test

import (
	"context"
	"fmt"
	"testing"

	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func newGuestTestManager() *usertoken.Manager {
	return usertoken.NewManager(usertoken.Config{CookieName: "trip2g_token", Secret: "test-secret", ExpiresIn: 3600e9, Insecure: true})
}

func TestAGuestSessionCarriesItsEdgeThroughTheCookie(t *testing.T) {
	mgr := newGuestTestManager()
	guest := usertoken.Data{ID: 7, Role: usertoken.RoleFederationGuest, Kid: "edge-a", Hop: 2, Reader: "abc"}

	stored, err := mgr.Store(&fasthttp.RequestCtx{}, guest)
	require.NoError(t, err)

	got, err := mgr.ParseToken(t.Context(), stored.JWT)
	require.NoError(t, err)
	require.Equal(t, &guest, got)
	require.True(t, got.IsFederationGuest())
	require.False(t, got.IsAdmin(), "a guest is never an admin")
}

func TestAValidatorThatEndsTheSessionLeavesTheReaderAnonymous(t *testing.T) {
	mgr := newGuestTestManager()
	mgr.AddValidator(func(context.Context, *usertoken.Data) error {
		return fmt.Errorf("edge revoked: %w", usertoken.ErrSessionEnded)
	})

	stored, err := mgr.Store(&fasthttp.RequestCtx{}, usertoken.Data{ID: 7, Role: usertoken.RoleFederationGuest, Kid: "edge-a"})
	require.NoError(t, err)

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetCookie("trip2g_token", stored.JWT)
	got, err := mgr.Extract(fctx)
	require.NoError(t, err)
	require.Nil(t, got)
	require.Contains(t, string(fctx.Response.Header.PeekCookie("trip2g_token")), "expires=", "the dead cookie is cleared")

	got, err = mgr.ParseToken(t.Context(), stored.JWT)
	require.NoError(t, err)
	require.Nil(t, got)
}
