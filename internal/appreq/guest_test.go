package appreq_test

import (
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func TestDenyGuestRefusesOnlyAFederationGuest(t *testing.T) {
	tests := []struct {
		name  string
		token *usertoken.Data
		want  error
	}{
		{name: "anonymous", token: nil},
		{name: "user", token: &usertoken.Data{ID: 1, Role: "user"}},
		{name: "admin", token: &usertoken.Data{ID: 1, Role: "admin"}},
		{name: "federated agent", token: &usertoken.Data{Role: usertoken.RoleFederated}},
		{name: "federation guest", token: &usertoken.Data{ID: 9, Role: usertoken.RoleFederationGuest, Kid: "edge"}, want: appreq.ErrGuestReadOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &appreq.Request{}
			req.SetUserToken(tt.token)
			ctx := appreq.NewContext(t.Context(), req)

			require.Equal(t, tt.want, appreq.DenyGuest(ctx))
		})
	}

	require.NoError(t, appreq.DenyGuest(t.Context()), "no request in the context is not a guest")
}
