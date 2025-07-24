package createpatreoncredentials_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createpatreoncredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createpatreoncredentials_test . Env

func assertPayload(t *testing.T, want, got model.CreatePatreonCredentialsOrErrorPayload) {
	t.Helper()
	// Skip time comparison for CreatedAt field
	if payload, ok := got.(*model.CreatePatreonCredentialsPayload); ok {
		if wantPayload, wantOk := want.(*model.CreatePatreonCredentialsPayload); wantOk {
			require.Equal(t, wantPayload.PatreonCredentials.ID, payload.PatreonCredentials.ID)
			require.Equal(t, wantPayload.PatreonCredentials.CreatedBy, payload.PatreonCredentials.CreatedBy)
			require.Equal(t, wantPayload.PatreonCredentials.CreatorAccessToken, payload.PatreonCredentials.CreatorAccessToken)
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
	InsertPatreonCredentials(ctx context.Context, arg db.InsertPatreonCredentialsParams) (db.PatreonCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.CreatePatreonCredentialsInput
		mockFunc func() *envMock
		want     model.CreatePatreonCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success",
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: "test-token-123456789",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertPatreonCredentialsFunc = func(ctx context.Context, arg db.InsertPatreonCredentialsParams) (db.PatreonCredential, error) {
					return db.PatreonCredential{
						ID:                 1,
						CreatedAt:          time.Now(),
						CreatedBy:          1,
						CreatorAccessToken: arg.CreatorAccessToken,
					}, nil
				}
				return mock
			},
			want: &model.CreatePatreonCredentialsPayload{
				PatreonCredentials: &db.PatreonCredential{
					ID:                 1,
					CreatedBy:          1,
					CreatorAccessToken: "test-token-123456789",
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty token",
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: "",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "creatorAccessToken", Value: "cannot be blank"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - token too short",
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: "short",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "creatorAccessToken", Value: "the length must be between 10 and 500"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - token too long",
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: string(make([]byte, 501)),
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "creatorAccessToken", Value: "the length must be between 10 and 500"}},
			},
			wantErr: false,
		},
		{
			name: "current admin user token error",
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: "test-token-123456789",
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
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: "test-token-123456789",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertPatreonCredentialsFunc = func(ctx context.Context, arg db.InsertPatreonCredentialsParams) (db.PatreonCredential, error) {
					return db.PatreonCredential{}, errors.New("UNIQUE constraint failed: patreon_credentials.creator_access_token")
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "Patreon credentials already exist",
			},
			wantErr: false,
		},
		{
			name: "database error",
			input: model.CreatePatreonCredentialsInput{
				CreatorAccessToken: "test-token-123456789",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertPatreonCredentialsFunc = func(ctx context.Context, arg db.InsertPatreonCredentialsParams) (db.PatreonCredential, error) {
					return db.PatreonCredential{}, errors.New("database error")
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

			got, err := createpatreoncredentials.Resolve(context.Background(), tt.mockFunc(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assertPayload(t, tt.want, got)
		})
	}
}
