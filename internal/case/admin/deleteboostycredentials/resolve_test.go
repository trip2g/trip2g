package deleteboostycredentials_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/deleteboostycredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg deleteboostycredentials_test . Env

type Env interface {
	SoftDeleteBoostyCredentials(ctx context.Context, arg db.SoftDeleteBoostyCredentialsParams) (db.BoostyCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StopBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64) error
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.DeleteBoostyCredentialsInput
		mockFunc func() *envMock
		want     model.DeleteBoostyCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success",
			input: model.DeleteBoostyCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeleteBoostyCredentialsFunc = func(ctx context.Context, arg db.SoftDeleteBoostyCredentialsParams) (db.BoostyCredential, error) {
					require.Equal(t, int64(1), arg.ID)
					require.NotNil(t, arg.DeletedBy)
					require.Equal(t, int64(1), *arg.DeletedBy)
					return db.BoostyCredential{
						ID:        1,
						CreatedAt: time.Now(),
						CreatedBy: 1,
						DeletedAt: ptr.To(time.Now()),
						DeletedBy: ptr.To(int64(1)),
					}, nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				return mock
			},
			want: &model.DeleteBoostyCredentialsPayload{
				DeletedID: 1,
			},
			wantErr: false,
		},
		{
			name: "current admin user token error",
			input: model.DeleteBoostyCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "database error",
			input: model.DeleteBoostyCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeleteBoostyCredentialsFunc = func(ctx context.Context, arg db.SoftDeleteBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, errors.New("database error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "credentials not found",
			input: model.DeleteBoostyCredentialsInput{
				ID: 999,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeleteBoostyCredentialsFunc = func(ctx context.Context, arg db.SoftDeleteBoostyCredentialsParams) (db.BoostyCredential, error) {
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
			got, err := deleteboostycredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				require.Equal(t, tt.want, got)
			}
		})
	}
}
