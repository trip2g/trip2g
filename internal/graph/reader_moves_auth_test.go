package graph

// Tests for the readerMoves subscription auth lane. It must be stricter than
// noteChanges: live reader movement is admin-surface data, so only an admin
// session or a real instance API key may subscribe — webhook shortapitokens
// and ordinary signed-in users are rejected.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/usertoken"
)

func TestReaderMovesAuth(t *testing.T) {
	webhookToken, err := shortapitoken.Sign(shortapitoken.Data{
		ReadPatterns: []string{"**"},
	}, testShortAPISecret, time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name       string
		token      *usertoken.Data
		headers    map[string]string
		keyByValue func(value string) (db.ApiKey, error)
		wantErr    bool
	}{
		{
			name:  "admin session is accepted",
			token: &usertoken.Data{ID: 1, Role: "admin"},
		},
		{
			name:    "instance API key is accepted",
			headers: map[string]string{"X-API-Key": "instance-key"},
			keyByValue: func(value string) (db.ApiKey, error) {
				return db.ApiKey{ID: 7}, nil
			},
		},
		{
			name:    "webhook shortapitoken is rejected (unlike noteChanges)",
			headers: map[string]string{"Authorization": "Bearer " + webhookToken},
			wantErr: true,
		},
		{
			name:    "ordinary signed-in user is rejected (unlike noteChanges)",
			token:   &usertoken.Data{ID: 5, Role: "user"},
			wantErr: true,
		},
		{
			name:    "anonymous is rejected",
			wantErr: true,
		},
		{
			name:    "invalid API key is rejected",
			headers: map[string]string{"X-API-Key": "bogus"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := authCtx(tt.token, tt.headers)
			env := authEnvMock(tt.token, tt.keyByValue)

			authErr := readerMovesAuth(ctx, env)

			if tt.wantErr {
				require.Error(t, authErr)
				return
			}
			require.NoError(t, authErr)
		})
	}
}
