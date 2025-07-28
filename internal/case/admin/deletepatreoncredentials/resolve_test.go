package deletepatreoncredentials_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/deletepatreoncredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg deletepatreoncredentials_test . Env

func assertPayload(t *testing.T, want, got model.DeletePatreonCredentialsOrErrorPayload) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve() = %v, want %v", got, want)
		for _, desc := range pretty.Diff(got, want) {
			t.Error(desc)
		}
	}
}

type Env interface {
	SoftDeletePatreonCredentials(ctx context.Context, arg db.SoftDeletePatreonCredentialsParams) (db.PatreonCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StopPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.DeletePatreonCredentialsInput
		mockFunc func() *envMock
		want     model.DeletePatreonCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success",
			input: model.DeletePatreonCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeletePatreonCredentialsFunc = func(ctx context.Context, arg db.SoftDeletePatreonCredentialsParams) (db.PatreonCredential, error) {
					deletedAt := time.Now()
					return db.PatreonCredential{
						ID:                 1,
						CreatedAt:          time.Now(),
						CreatedBy:          1,
						DeletedAt:          sql.NullTime{Time: deletedAt, Valid: true},
						DeletedBy:          sql.NullInt64{Int64: 1, Valid: true},
						CreatorAccessToken: "test-token",
					}, nil
				}
				mock.StopPatreonRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				return mock
			},
			want: &model.DeletePatreonCredentialsPayload{
				DeletedID: 1,
			},
			wantErr: false,
		},
		{
			name: "current admin user token error",
			input: model.DeletePatreonCredentialsInput{
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
			input: model.DeletePatreonCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeletePatreonCredentialsFunc = func(ctx context.Context, arg db.SoftDeletePatreonCredentialsParams) (db.PatreonCredential, error) {
					return db.PatreonCredential{}, errors.New("database error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not found error (no rows affected)",
			input: model.DeletePatreonCredentialsInput{
				ID: 999,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeletePatreonCredentialsFunc = func(ctx context.Context, arg db.SoftDeletePatreonCredentialsParams) (db.PatreonCredential, error) {
					// Simulate no rows affected (record not found or already deleted)
					return db.PatreonCredential{}, errors.New("no rows affected")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "stop background job error",
			input: model.DeletePatreonCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.SoftDeletePatreonCredentialsFunc = func(ctx context.Context, arg db.SoftDeletePatreonCredentialsParams) (db.PatreonCredential, error) {
					deletedAt := time.Now()
					return db.PatreonCredential{
						ID:                 1,
						CreatedAt:          time.Now(),
						CreatedBy:          1,
						DeletedAt:          sql.NullTime{Time: deletedAt, Valid: true},
						DeletedBy:          sql.NullInt64{Int64: 1, Valid: true},
						CreatorAccessToken: "test-token",
					}, nil
				}
				mock.StopPatreonRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return errors.New("failed to stop background job")
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
			got, err := deletepatreoncredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assertPayload(t, tt.want, got)

			// Verify StopPatreonRefreshBackgroundJob was called for successful cases
			if tt.name == "success" && !tt.wantErr && len(env.StopPatreonRefreshBackgroundJobCalls()) != 1 {
				t.Errorf("Expected StopPatreonRefreshBackgroundJob to be called once, got %d calls", len(env.StopPatreonRefreshBackgroundJobCalls()))
			}
		})
	}
}
