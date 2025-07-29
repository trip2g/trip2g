package createboostycredentials_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createboostycredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createboostycredentials_test . Env

func assertPayload(t *testing.T, want, got model.CreateBoostyCredentialsOrErrorPayload) {
	t.Helper()
	// Skip time comparison for CreatedAt field
	if payload, ok := got.(*model.CreateBoostyCredentialsPayload); ok {
		if wantPayload, wantOk := want.(*model.CreateBoostyCredentialsPayload); wantOk {
			require.Equal(t, wantPayload.BoostyCredentials.ID, payload.BoostyCredentials.ID)
			require.Equal(t, wantPayload.BoostyCredentials.CreatedBy, payload.BoostyCredentials.CreatedBy)
			require.Equal(t, wantPayload.BoostyCredentials.AuthData, payload.BoostyCredentials.AuthData)
			require.Equal(t, wantPayload.BoostyCredentials.DeviceID, payload.BoostyCredentials.DeviceID)
			require.Equal(t, wantPayload.BoostyCredentials.BlogName, payload.BoostyCredentials.BlogName)
			return
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve() = %v, want %v", got, want)
		for _, desc := range pretty.Diff(got, want) {
			t.Error(desc)
		}
	}
}

type Env interface {
	InsertBoostyCredentials(ctx context.Context, arg db.InsertBoostyCredentialsParams) (db.BoostyCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.CreateBoostyCredentialsInput
		mockFunc func() *envMock
		want     model.CreateBoostyCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "device-123",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertBoostyCredentialsFunc = func(ctx context.Context, arg db.InsertBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:        1,
						CreatedAt: time.Now(),
						CreatedBy: 1,
						AuthData:  arg.AuthData,
						DeviceID:  arg.DeviceID,
						BlogName:  arg.BlogName,
					}, nil
				}
				return mock
			},
			want: &model.CreateBoostyCredentialsPayload{
				BoostyCredentials: &db.BoostyCredential{
					ID:        1,
					CreatedBy: 1,
					AuthData:  "test-auth-data-123456789",
					DeviceID:  "device-123",
					BlogName:  "testblog",
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty auth data",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "",
				DeviceID: "device-123",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "authData", Value: "cannot be blank"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty device id",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "deviceId", Value: "cannot be blank"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty blog name",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "device-123",
				BlogName: "",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "blogName", Value: "cannot be blank"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - auth data too short",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "short",
				DeviceID: "device-123",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "authData", Value: "the length must be between 10 and 10000"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - device id too short",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "dev",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "deviceId", Value: "the length must be between 5 and 100"}},
			},
			wantErr: false,
		},
		{
			name: "current admin user token error",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "device-123",
				BlogName: "testblog",
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
			name: "unique constraint violation",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "device-123",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertBoostyCredentialsFunc = func(ctx context.Context, arg db.InsertBoostyCredentialsParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, errors.New("UNIQUE constraint failed: boosty_credentials.blog_name")
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "Boosty credentials already exist",
			},
			wantErr: false,
		},
		{
			name: "database error",
			input: model.CreateBoostyCredentialsInput{
				AuthData: "test-auth-data-123456789",
				DeviceID: "device-123",
				BlogName: "testblog",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertBoostyCredentialsFunc = func(ctx context.Context, arg db.InsertBoostyCredentialsParams) (db.BoostyCredential, error) {
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
			got, err := createboostycredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assertPayload(t, tt.want, got)
		})
	}
}
