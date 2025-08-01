package updateboostycredentials_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/updateboostycredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updateboostycredentials_test . Env

type Env interface {
	BoostyCredentials(ctx context.Context, id int64) (db.BoostyCredential, error)
	UpdateBoostyCredentials(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error)
	StartBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64, immediately bool) error
	StopBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64) error
}

type envMock = EnvMock

func strPtr(s string) *string {
	return &s
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.UpdateBoostyCredentialsInput
		mockFunc func() *envMock
		want     model.UpdateBoostyCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success - update all fields",
			input: model.UpdateBoostyCredentialsInput{
				ID:       1,
				AuthData: strPtr("new-auth-data-123456789"),
				DeviceID: strPtr("new-device-123"),
				BlogName: strPtr("newblog"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "old-auth-data",
						DeviceID: "old-device",
						BlogName: "oldblog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					require.Equal(t, int64(1), arg.ID)
					require.Equal(t, "new-auth-data-123456789", arg.AuthData)
					require.Equal(t, "new-device-123", arg.DeviceID)
					require.Equal(t, "newblog", arg.BlogName)
					return db.BoostyCredential{
						ID:        1,
						CreatedAt: time.Now(),
						CreatedBy: 1,
						AuthData:  "new-auth-data-123456789",
						DeviceID:  "new-device-123",
						BlogName:  "newblog",
					}, nil
				}
				return mock
			},
			want: &model.UpdateBoostyCredentialsPayload{
				BoostyCredentials: &db.BoostyCredential{
					ID:        1,
					CreatedBy: 1,
					AuthData:  "new-auth-data-123456789",
					DeviceID:  "new-device-123",
					BlogName:  "newblog",
				},
			},
			wantErr: false,
		},
		{
			name: "success - update auth data only",
			input: model.UpdateBoostyCredentialsInput{
				ID:       1,
				AuthData: strPtr("new-auth-data-123456789"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "old-auth-data",
						DeviceID: "device-123",
						BlogName: "testblog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					require.Equal(t, int64(1), arg.ID)
					require.Equal(t, "new-auth-data-123456789", arg.AuthData)
					require.Equal(t, "device-123", arg.DeviceID)
					require.Equal(t, "testblog", arg.BlogName)
					return db.BoostyCredential{
						ID:        1,
						CreatedAt: time.Now(),
						CreatedBy: 1,
						AuthData:  "new-auth-data-123456789",
						DeviceID:  "device-123",
						BlogName:  "testblog",
					}, nil
				}
				return mock
			},
			want: &model.UpdateBoostyCredentialsPayload{
				BoostyCredentials: &db.BoostyCredential{
					ID:        1,
					CreatedBy: 1,
					AuthData:  "new-auth-data-123456789",
					DeviceID:  "device-123",
					BlogName:  "testblog",
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - no fields to update",
			input: model.UpdateBoostyCredentialsInput{
				ID: 1,
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "existing-auth-data",
						DeviceID: "existing-device",
						BlogName: "existing-blog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:        1,
						CreatedBy: 1,
						AuthData:  arg.AuthData,
						DeviceID:  arg.DeviceID,
						BlogName:  arg.BlogName,
					}, nil
				}
				return mock
			},
			want: &model.UpdateBoostyCredentialsPayload{
				BoostyCredentials: &db.BoostyCredential{
					ID:        1,
					CreatedBy: 1,
					AuthData:  "existing-auth-data",
					DeviceID:  "existing-device",
					BlogName:  "existing-blog",
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - auth data too short",
			input: model.UpdateBoostyCredentialsInput{
				ID:       1,
				AuthData: strPtr("short"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "existing-auth-data",
						DeviceID: "existing-device",
						BlogName: "existing-blog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:        1,
						CreatedBy: 1,
						AuthData:  arg.AuthData,
						DeviceID:  arg.DeviceID,
						BlogName:  arg.BlogName,
					}, nil
				}
				return mock
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "authData", Value: "the length must be between 10 and 10000"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - device id too short",
			input: model.UpdateBoostyCredentialsInput{
				ID:       1,
				DeviceID: strPtr("dev"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "existing-auth-data",
						DeviceID: "existing-device",
						BlogName: "existing-blog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:        1,
						CreatedBy: 1,
						AuthData:  arg.AuthData,
						DeviceID:  arg.DeviceID,
						BlogName:  arg.BlogName,
					}, nil
				}
				return mock
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "deviceId", Value: "the length must be between 5 and 100"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - blog name too long",
			input: model.UpdateBoostyCredentialsInput{
				ID:       1,
				BlogName: strPtr(string(make([]byte, 101))),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "existing-auth-data",
						DeviceID: "existing-device",
						BlogName: "existing-blog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:        1,
						CreatedBy: 1,
						AuthData:  arg.AuthData,
						DeviceID:  arg.DeviceID,
						BlogName:  arg.BlogName,
					}, nil
				}
				return mock
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "blogName", Value: "the length must be between 1 and 100"}},
			},
			wantErr: false,
		},
		{
			name: "credentials not found",
			input: model.UpdateBoostyCredentialsInput{
				ID:       999,
				AuthData: strPtr("new-auth-data-123456789"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "existing-auth-data",
						DeviceID: "existing-device",
						BlogName: "existing-blog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, sql.ErrNoRows
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "Boosty credentials not found",
			},
			wantErr: false,
		},
		{
			name: "database error",
			input: model.UpdateBoostyCredentialsInput{
				ID:       1,
				AuthData: strPtr("new-auth-data-123456789"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       id,
						AuthData: "existing-auth-data",
						DeviceID: "existing-device",
						BlogName: "existing-blog",
					}, nil
				}
				mock.StartBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64, immediately bool) error {
					return nil
				}
				mock.StopBoostyRefreshBackgroundJobFunc = func(ctx context.Context, credentialsID int64) error {
					return nil
				}
				mock.UpdateBoostyCredentialsFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, errors.New("database error")
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
			got, err := updateboostycredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gotPayload, ok := got.(*model.UpdateBoostyCredentialsPayload); ok {
					wantPayload := tt.want.(*model.UpdateBoostyCredentialsPayload)
					require.Equal(t, wantPayload.BoostyCredentials.ID, gotPayload.BoostyCredentials.ID)
					require.Equal(t, wantPayload.BoostyCredentials.CreatedBy, gotPayload.BoostyCredentials.CreatedBy)
					require.Equal(t, wantPayload.BoostyCredentials.AuthData, gotPayload.BoostyCredentials.AuthData)
					require.Equal(t, wantPayload.BoostyCredentials.DeviceID, gotPayload.BoostyCredentials.DeviceID)
					require.Equal(t, wantPayload.BoostyCredentials.BlogName, gotPayload.BoostyCredentials.BlogName)
				} else {
					require.Equal(t, tt.want, got)
				}
			}
		})
	}
}
