package restoreboostycredentials_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/restoreboostycredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg restoreboostycredentials_test . Env

type Env interface {
	RestoreBoostyCredentials(ctx context.Context, id int64) (db.BoostyCredential, error)
	StartBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64, immediately bool) error
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.RestoreBoostyCredentialsInput
		mockFunc func() *envMock
		want     model.RestoreBoostyCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success",
			input: model.RestoreBoostyCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.RestoreBoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					require.Equal(t, int64(1), id)
					return db.BoostyCredential{
						ID:        1,
						CreatedAt: time.Now(),
						CreatedBy: 1,
						AuthData:  "test-auth-data",
						DeviceID:  "device-123",
						BlogName:  "testblog",
						DeletedAt: sql.NullTime{Valid: false},
						DeletedBy: sql.NullInt64{Valid: false},
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				return mock
			},
			want: &model.RestoreBoostyCredentialsPayload{
				BoostyCredentials: &db.BoostyCredential{
					ID:        1,
					CreatedBy: 1,
					AuthData:  "test-auth-data",
					DeviceID:  "device-123",
					BlogName:  "testblog",
				},
			},
			wantErr: false,
		},
		{
			name: "database error",
			input: model.RestoreBoostyCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.RestoreBoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, errors.New("database error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "credentials not found",
			input: model.RestoreBoostyCredentialsInput{
				ID: 999,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.RestoreBoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, sql.ErrNoRows
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := tt.mockFunc()
			got, err := restoreboostycredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotPayload := got.(*model.RestoreBoostyCredentialsPayload)
				wantPayload := tt.want.(*model.RestoreBoostyCredentialsPayload)
				require.Equal(t, wantPayload.BoostyCredentials.ID, gotPayload.BoostyCredentials.ID)
				require.Equal(t, wantPayload.BoostyCredentials.CreatedBy, gotPayload.BoostyCredentials.CreatedBy)
				require.Equal(t, wantPayload.BoostyCredentials.AuthData, gotPayload.BoostyCredentials.AuthData)
				require.Equal(t, wantPayload.BoostyCredentials.DeviceID, gotPayload.BoostyCredentials.DeviceID)
				require.Equal(t, wantPayload.BoostyCredentials.BlogName, gotPayload.BoostyCredentials.BlogName)
			}
		})
	}
}
