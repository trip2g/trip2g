package federationguest_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"trip2g/internal/case/federationguest"
	"trip2g/internal/case/mcp"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

const testReader = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func validForm() federationguest.Form {
	return federationguest.Form{Token: "handoff", Path: "/", Origin: "https://hub.test"}
}

func validClaims() mcp.GuestHandoffClaims {
	return mcp.GuestHandoffClaims{
		GuestHandoff: mcp.GuestHandoff{Path: "/", Hop: 1, Reader: testReader, Origin: "https://hub.test"},
		Caller:       mcp.FederationCaller{KID: "edge"},
		ID:           "rid-1",
		ExpiresAt:    time.Now().Add(30 * time.Second),
	}
}

func newEnv(claims mcp.GuestHandoffClaims) *EnvMock {
	spent := map[string]bool{}
	return &EnvMock{
		FederationGuestsEnabledFunc: func() bool { return true },
		FederationMaxDepthFunc:      func() int { return 3 },
		VerifyGuestHandoffFunc: func(context.Context, string) (mcp.GuestHandoffClaims, error) {
			return claims, nil
		},
		SpendFederationGuestIDFunc: func(id string, _ time.Time) bool {
			if spent[id] {
				return false
			}
			spent[id] = true
			return true
		},
		FederationGuestUserFunc: func(context.Context, string, string) (int64, error) { return 42, nil },
		GuestTicketSecretFunc:   func() string { return "ticket-secret" },
	}
}

func TestAnAdmittedGuestEntersAsTheSyntheticUserOfItsEdge(t *testing.T) {
	env := newEnv(validClaims())

	admitted, err := federationguest.Admit(t.Context(), env, validForm())
	require.NoError(t, err)

	data, path, err := federationguest.Enter(env, admitted.Ticket, admitted.Nonce)
	require.NoError(t, err)
	require.Equal(t, "/", path)
	want := usertoken.Data{ID: 42, Role: usertoken.RoleFederationGuest, Kid: "edge", Hop: 1, Reader: testReader}
	require.Equal(t, want, data)

	calls := env.FederationGuestUserCalls()
	require.Len(t, calls, 1)
	require.Equal(t, "edge", calls[0].Kid)
	require.Equal(t, testReader, calls[0].Reader)
}

func TestATicketOpensOnlyWithTheNonceOfItsPost(t *testing.T) {
	env := newEnv(validClaims())

	admitted, err := federationguest.Admit(t.Context(), env, validForm())
	require.NoError(t, err)
	require.NotEmpty(t, admitted.Nonce)
	require.NotContains(t, admitted.Ticket, admitted.Nonce)

	for _, nonce := range []string{"", "wrong", admitted.Nonce + "x"} {
		_, _, err = federationguest.Enter(env, admitted.Ticket, nonce)
		require.ErrorIs(t, err, federationguest.ErrRefused, "nonce %q", nonce)
	}

	_, _, err = federationguest.Enter(env, admitted.Ticket, admitted.Nonce)
	require.NoError(t, err, "a wrong nonce does not spend the ticket")
}

func TestATicketOpensOneSession(t *testing.T) {
	env := newEnv(validClaims())

	admitted, err := federationguest.Admit(t.Context(), env, validForm())
	require.NoError(t, err)
	_, _, err = federationguest.Enter(env, admitted.Ticket, admitted.Nonce)
	require.NoError(t, err)

	_, _, err = federationguest.Enter(env, admitted.Ticket, admitted.Nonce)
	require.ErrorIs(t, err, federationguest.ErrRefused)
}

func TestAdmitRefuses(t *testing.T) {
	tests := []struct {
		name   string
		claims func(*mcp.GuestHandoffClaims)
		env    func(*EnvMock)
		path   string
		origin string
	}{
		{name: "posted from another site", origin: "https://attacker.test"},
		{name: "posted with no Origin", origin: "-"},
		{name: "posted with Origin null", origin: "null"},
		{name: "signed origin is not an origin", claims: func(c *mcp.GuestHandoffClaims) { c.Origin = "" }},
		{name: "setting off", env: func(e *EnvMock) { e.FederationGuestsEnabledFunc = func() bool { return false } }},
		{name: "bad signature", env: func(e *EnvMock) {
			e.VerifyGuestHandoffFunc = func(context.Context, string) (mcp.GuestHandoffClaims, error) {
				return mcp.GuestHandoffClaims{}, mcp.ErrFedAuthBadSig
			}
		}},
		{name: "token for another path", path: "/other"},
		{name: "absolute target", claims: func(c *mcp.GuestHandoffClaims) { c.Path = "https://evil.test/" }, path: "https://evil.test/"},
		{name: "protocol-relative target", claims: func(c *mcp.GuestHandoffClaims) { c.Path = "//evil.test/" }, path: "//evil.test/"},
		{name: "no hop", claims: func(c *mcp.GuestHandoffClaims) { c.Hop = 0 }},
		{name: "hop over the limit", claims: func(c *mcp.GuestHandoffClaims) { c.Hop = 4 }},
		{name: "reader that is not a hash", claims: func(c *mcp.GuestHandoffClaims) { c.Reader = "admin@example.com" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := validClaims()
			if tt.claims != nil {
				tt.claims(&claims)
			}
			env := newEnv(claims)
			if tt.env != nil {
				tt.env(env)
			}
			form := validForm()
			if tt.path != "" {
				form.Path = tt.path
			}
			switch tt.origin {
			case "":
			case "-":
				form.Origin = ""
			default:
				form.Origin = tt.origin
			}

			_, err := federationguest.Admit(t.Context(), env, form)
			require.ErrorIs(t, err, federationguest.ErrRefused)
			require.Empty(t, env.FederationGuestUserCalls(), "a refused handoff neither looks up nor creates a user")
			require.Empty(t, env.SpendFederationGuestIDCalls(), "a refused handoff does not spend the token")
		})
	}
}

func TestTheOriginIsComparedNormalised(t *testing.T) {
	claims := validClaims()
	claims.Origin = "https://hub.test"
	env := newEnv(claims)

	form := validForm()
	form.Origin = "HTTPS://Hub.Test:443"
	_, err := federationguest.Admit(t.Context(), env, form)
	require.NoError(t, err)
}

func TestAReplayedHandoffIsRefused(t *testing.T) {
	env := newEnv(validClaims())

	_, err := federationguest.Admit(t.Context(), env, validForm())
	require.NoError(t, err)

	_, err = federationguest.Admit(t.Context(), env, validForm())
	require.ErrorIs(t, err, federationguest.ErrRefused)
	require.Len(t, env.FederationGuestUserCalls(), 1)
}

func TestTurningTheSettingOffRefusesAnOutstandingTicket(t *testing.T) {
	env := newEnv(validClaims())
	admitted, err := federationguest.Admit(t.Context(), env, validForm())
	require.NoError(t, err)

	env.FederationGuestsEnabledFunc = func() bool { return false }

	_, _, err = federationguest.Enter(env, admitted.Ticket, admitted.Nonce)
	require.ErrorIs(t, err, federationguest.ErrRefused)
}

func TestATamperedTicketIsRefused(t *testing.T) {
	env := newEnv(validClaims())
	admitted, err := federationguest.Admit(t.Context(), env, validForm())
	require.NoError(t, err)

	_, _, err = federationguest.Enter(env, strings.TrimSuffix(admitted.Ticket, admitted.Ticket[len(admitted.Ticket)-2:])+"xx", admitted.Nonce)
	require.ErrorIs(t, err, federationguest.ErrRefused)
}

func TestAFailureToCreateTheGuestIsNotARefusal(t *testing.T) {
	env := newEnv(validClaims())
	env.FederationGuestUserFunc = func(context.Context, string, string) (int64, error) {
		return 0, errors.New("disk full")
	}

	_, err := federationguest.Admit(t.Context(), env, validForm())
	require.Error(t, err)
	require.NotErrorIs(t, err, federationguest.ErrRefused)
}
