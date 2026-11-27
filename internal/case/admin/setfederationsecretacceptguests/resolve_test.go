package setfederationsecretacceptguests_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"trip2g/internal/case/admin/setfederationsecretacceptguests"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func envFor(row db.FederationSecret, rowErr error) *EnvMock {
	return &EnvMock{
		CurrentAdminUserTokenFunc: func(context.Context) (*usertoken.Data, error) { return &usertoken.Data{ID: 1}, nil },
		FederationSecretByIDFunc:  func(context.Context, int64) (db.FederationSecret, error) { return row, rowErr },
		SetFederationSecretAcceptGuestsFunc: func(context.Context, db.SetFederationSecretAcceptGuestsParams) error {
			return nil
		},
	}
}

func TestSwitchingPeopleOnAnInboundKey(t *testing.T) {
	env := envFor(db.FederationSecret{ID: 7, Kid: "edge"}, nil)

	got, err := setfederationsecretacceptguests.Resolve(t.Context(), env, model.SetFederationSecretAcceptGuestsInput{ID: 7, AcceptGuests: true})
	require.NoError(t, err)
	require.Equal(t, &model.SetFederationSecretAcceptGuestsPayload{ID: 7, AcceptGuests: true}, got)
	calls := env.SetFederationSecretAcceptGuestsCalls()
	require.Len(t, calls, 1)
	require.Equal(t, db.SetFederationSecretAcceptGuestsParams{ID: 7, AcceptGuests: true}, calls[0].Arg)
}

func TestOnlyALiveInboundKeyCanAcceptPeople(t *testing.T) {
	outbound := "https://peer.test/_system/mcp"
	revoked := time.Now()

	tests := []struct {
		name   string
		row    db.FederationSecret
		rowErr error
	}{
		{name: "unknown key", rowErr: sql.ErrNoRows},
		{name: "outbound key", row: db.FederationSecret{ID: 7, KbUrl: &outbound}},
		{name: "revoked key", row: db.FederationSecret{ID: 7, RevokedAt: &revoked}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := envFor(tt.row, tt.rowErr)

			got, err := setfederationsecretacceptguests.Resolve(t.Context(), env, model.SetFederationSecretAcceptGuestsInput{ID: 7, AcceptGuests: true})
			require.NoError(t, err)
			require.IsType(t, &model.ErrorPayload{}, got)
			require.Empty(t, env.SetFederationSecretAcceptGuestsCalls())
		})
	}
}
