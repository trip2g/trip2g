package createoidccredentials_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/createoidccredentials"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.CreateOIDCCredentialsInput
		mockFunc func() *envMock
		want     model.CreateOIDCCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success - creates credentials and sets active",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted-" + string(plaintext)), nil
				}
				mock.DeactivateAllOIDCCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertOIDCCredentialsFunc = func(ctx context.Context, arg db.InsertOIDCCredentialsParams) (db.OidcCredential, error) {
					require.True(t, arg.Active, "new credentials should be active")
					require.Equal(t, "openid email profile", arg.Scopes, "default scopes applied")
					return db.OidcCredential{
						ID:        1,
						Name:      arg.Name,
						Issuer:    arg.Issuer,
						ClientID:  arg.ClientID,
						Scopes:    arg.Scopes,
						Active:    arg.Active,
						CreatedBy: arg.CreatedBy,
						CreatedAt: time.Now(),
					}, nil
				}
				return mock
			},
			want: &model.CreateOIDCCredentialsPayload{
				Credentials: &db.OidcCredential{
					ID:        1,
					Name:      "Production",
					Issuer:    "https://accounts.example.com",
					ClientID:  "1234567890-client-id",
					Scopes:    "openid email profile",
					Active:    true,
					CreatedBy: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "success - custom optional fields",
			input: model.CreateOIDCCredentialsInput{
				Name:               "Custom",
				Issuer:             "https://accounts.example.com",
				ClientID:           "1234567890-client-id",
				ClientSecret:       "secret-key-12345",
				Scopes:             ptr("openid email"),
				AutoProvision:      ptrBool(true),
				AllowedEmailDomain: ptr("example.com"),
				RequiredGroup:      ptr("admins"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllOIDCCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertOIDCCredentialsFunc = func(ctx context.Context, arg db.InsertOIDCCredentialsParams) (db.OidcCredential, error) {
					require.Equal(t, "openid email", arg.Scopes)
					require.True(t, arg.AutoProvision)
					require.Equal(t, "example.com", arg.AllowedEmailDomain)
					require.Equal(t, "admins", arg.RequiredGroup)
					return db.OidcCredential{
						ID:        3,
						Name:      arg.Name,
						Issuer:    arg.Issuer,
						ClientID:  arg.ClientID,
						Scopes:    arg.Scopes,
						Active:    true,
						CreatedBy: 1,
						CreatedAt: time.Now(),
					}, nil
				}
				return mock
			},
			want: &model.CreateOIDCCredentialsPayload{
				Credentials: &db.OidcCredential{
					ID:        3,
					Name:      "Custom",
					Issuer:    "https://accounts.example.com",
					ClientID:  "1234567890-client-id",
					Scopes:    "openid email",
					Active:    true,
					CreatedBy: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "deactivates existing credentials before creating new",
			input: model.CreateOIDCCredentialsInput{
				Name:         "New Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "new-client-id-12345",
				ClientSecret: "new-secret-key-123",
			},
			mockFunc: func() *envMock {
				deactivateCalled := false
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllOIDCCredentialsFunc = func(ctx context.Context) error {
					deactivateCalled = true
					return nil
				}
				mock.InsertOIDCCredentialsFunc = func(ctx context.Context, arg db.InsertOIDCCredentialsParams) (db.OidcCredential, error) {
					require.True(t, deactivateCalled, "should deactivate existing before insert")
					require.True(t, arg.Active, "new credentials should be active")
					return db.OidcCredential{
						ID:        2,
						Name:      arg.Name,
						Issuer:    arg.Issuer,
						ClientID:  arg.ClientID,
						Active:    true,
						CreatedBy: 1,
						CreatedAt: time.Now(),
					}, nil
				}
				return mock
			},
			want: &model.CreateOIDCCredentialsPayload{
				Credentials: &db.OidcCredential{
					ID:        2,
					Name:      "New Production",
					Issuer:    "https://accounts.example.com",
					ClientID:  "new-client-id-12345",
					Active:    true,
					CreatedBy: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty name",
			input: model.CreateOIDCCredentialsInput{
				Name:         "",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "name", Value: "cannot be blank"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - invalid issuer",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "not-a-url",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "issuer", Value: "must be a valid URL"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - client id too short",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "short",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "clientId", Value: "the length must be between 10 and 200"}},
			},
			wantErr: false,
		},
		{
			name: "validation error - client secret too short",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "short",
			},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "clientSecret", Value: "the length must be between 10 and 200"}},
			},
			wantErr: false,
		},
		{
			name: "authorization error",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
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
			name: "credential validation error",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return errors.New("discovery failed: issuer unreachable")
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "Invalid credentials: discovery failed: issuer unreachable",
			},
			wantErr: false,
		},
		{
			name: "encryption error",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return nil, errors.New("encryption failed")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "deactivate error",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllOIDCCredentialsFunc = func(ctx context.Context) error {
					return errors.New("database error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unique constraint violation",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllOIDCCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertOIDCCredentialsFunc = func(ctx context.Context, arg db.InsertOIDCCredentialsParams) (db.OidcCredential, error) {
					return db.OidcCredential{}, errors.New("UNIQUE constraint failed: oidc_credentials.client_id")
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "OIDC credentials with this client ID already exist",
			},
			wantErr: false,
		},
		{
			name: "database error",
			input: model.CreateOIDCCredentialsInput{
				Name:         "Production",
				Issuer:       "https://accounts.example.com",
				ClientID:     "1234567890-client-id",
				ClientSecret: "secret-key-12345",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateOIDCCredentialsFunc = func(ctx context.Context, issuer, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllOIDCCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertOIDCCredentialsFunc = func(ctx context.Context, arg db.InsertOIDCCredentialsParams) (db.OidcCredential, error) {
					return db.OidcCredential{}, errors.New("database error")
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
			got, err := createoidccredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			switch want := tt.want.(type) {
			case *model.CreateOIDCCredentialsPayload:
				gotPayload, ok := got.(*model.CreateOIDCCredentialsPayload)
				require.True(t, ok, "expected CreateOIDCCredentialsPayload")
				require.Equal(t, want.Credentials.ID, gotPayload.Credentials.ID)
				require.Equal(t, want.Credentials.Name, gotPayload.Credentials.Name)
				require.Equal(t, want.Credentials.Issuer, gotPayload.Credentials.Issuer)
				require.Equal(t, want.Credentials.ClientID, gotPayload.Credentials.ClientID)
				require.Equal(t, want.Credentials.Scopes, gotPayload.Credentials.Scopes)
				require.Equal(t, want.Credentials.Active, gotPayload.Credentials.Active)
				require.Equal(t, want.Credentials.CreatedBy, gotPayload.Credentials.CreatedBy)
			case *model.ErrorPayload:
				gotPayload, ok := got.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload")
				require.Equal(t, want.Message, gotPayload.Message)
				require.Equal(t, want.ByFields, gotPayload.ByFields)
			default:
				t.Errorf("unexpected payload type: %T", tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }
func ptrBool(b bool) *bool { return &b }
