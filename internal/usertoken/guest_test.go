package usertoken_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"trip2g/internal/usertoken"

	jwt "github.com/golang-jwt/jwt/v5"
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

func TestAGuestSessionLivesTwelveHoursAndAnOrdinaryOneKeepsItsConfig(t *testing.T) {
	mgr := newGuestTestManager()

	lifetime := func(data usertoken.Data) time.Duration {
		stored, err := mgr.Store(&fasthttp.RequestCtx{}, data)
		require.NoError(t, err)
		claims := jwt.MapClaims{}
		_, _, err = jwt.NewParser().ParseUnverified(stored.JWT, claims)
		require.NoError(t, err)
		exp, err := claims.GetExpirationTime()
		require.NoError(t, err)
		return time.Until(exp.Time)
	}

	guest := usertoken.Data{ID: 7, Role: usertoken.RoleFederationGuest, Kid: "e", Key: 1}
	require.InDelta(t, float64(usertoken.GuestSessionLifetime), float64(lifetime(guest)), float64(time.Minute))
	require.Equal(t, 12*time.Hour, usertoken.GuestSessionLifetime)
	require.InDelta(t, float64(time.Hour), float64(lifetime(usertoken.Data{ID: 7, Role: "user"})), float64(time.Minute))
}
