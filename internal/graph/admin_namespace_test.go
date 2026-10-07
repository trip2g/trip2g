package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/usertoken"
)

func TestAdminMutationNamespaceRefusesNonAdmins(t *testing.T) {
	tests := []struct {
		name    string
		token   *usertoken.Data
		wantErr bool
	}{
		{
			name:  "admin session is accepted",
			token: &usertoken.Data{ID: 1, Role: "admin"},
		},
		{
			name:    "signed-in user is refused",
			token:   &usertoken.Data{ID: 5, Role: "user"},
			wantErr: true,
		},
		{
			name:    "anonymous is refused",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mutationResolver{&Resolver{}}

			namespace, err := resolver.Admin(authCtx(tt.token, nil))

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, namespace)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, namespace)
		})
	}
}
