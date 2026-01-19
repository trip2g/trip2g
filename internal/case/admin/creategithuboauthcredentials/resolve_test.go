package creategithuboauthcredentials_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/creategithuboauthcredentials"
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
		input    model.CreateGitHubOAuthCredentialsInput
		mockFunc func() *envMock
		want     model.CreateGitHubOAuthCredentialsOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success - creates credentials and sets active",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted-" + string(plaintext)), nil
				}
				mock.DeactivateAllGitHubOAuthCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertGitHubOAuthCredentialsFunc = func(ctx context.Context, arg db.InsertGitHubOAuthCredentialsParams) (db.GithubOauthCredential, error) {
					require.True(t, arg.Active, "new credentials should be active")
					return db.GithubOauthCredential{
						ID:        1,
						Name:      arg.Name,
						ClientID:  arg.ClientID,
						Active:    arg.Active,
						CreatedBy: arg.CreatedBy,
						CreatedAt: time.Now(),
					}, nil
				}
				return mock
			},
			want: &model.CreateGitHubOAuthCredentialsPayload{
				Credentials: &db.GithubOauthCredential{
					ID:        1,
					Name:      "Production",
					ClientID:  "Iv1.abcdefghij12345",
					Active:    true,
					CreatedBy: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "deactivates existing credentials before creating new",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "New Production",
				ClientID:     "Iv1.newclientid1234",
				ClientSecret: "new-secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				deactivateCalled := false
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllGitHubOAuthCredentialsFunc = func(ctx context.Context) error {
					deactivateCalled = true
					return nil
				}
				mock.InsertGitHubOAuthCredentialsFunc = func(ctx context.Context, arg db.InsertGitHubOAuthCredentialsParams) (db.GithubOauthCredential, error) {
					require.True(t, deactivateCalled, "should deactivate existing before insert")
					require.True(t, arg.Active, "new credentials should be active")
					return db.GithubOauthCredential{
						ID:        2,
						Name:      arg.Name,
						ClientID:  arg.ClientID,
						Active:    true,
						CreatedBy: 1,
						CreatedAt: time.Now(),
					}, nil
				}
				return mock
			},
			want: &model.CreateGitHubOAuthCredentialsPayload{
				Credentials: &db.GithubOauthCredential{
					ID:        2,
					Name:      "New Production",
					ClientID:  "Iv1.newclientid1234",
					Active:    true,
					CreatedBy: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty name",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
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
			name: "validation error - client id too short",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "short",
				ClientSecret: "secret-key-12345678901234567890",
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
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
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
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
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
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
					return errors.New("invalid credentials: The OAuth client was not found")
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "Invalid credentials: invalid credentials: The OAuth client was not found",
			},
			wantErr: false,
		},
		{
			name: "encryption error",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
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
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllGitHubOAuthCredentialsFunc = func(ctx context.Context) error {
					return errors.New("database error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unique constraint violation",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllGitHubOAuthCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertGitHubOAuthCredentialsFunc = func(ctx context.Context, arg db.InsertGitHubOAuthCredentialsParams) (db.GithubOauthCredential, error) {
					return db.GithubOauthCredential{}, errors.New("UNIQUE constraint failed: github_oauth_credentials.client_id")
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "GitHub OAuth credentials with this client ID already exist",
			},
			wantErr: false,
		},
		{
			name: "database error",
			input: model.CreateGitHubOAuthCredentialsInput{
				Name:         "Production",
				ClientID:     "Iv1.abcdefghij12345",
				ClientSecret: "secret-key-12345678901234567890",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ValidateGitHubOAuthCredentialsFunc = func(ctx context.Context, clientID, clientSecret string) error {
					return nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.DeactivateAllGitHubOAuthCredentialsFunc = func(ctx context.Context) error {
					return nil
				}
				mock.InsertGitHubOAuthCredentialsFunc = func(ctx context.Context, arg db.InsertGitHubOAuthCredentialsParams) (db.GithubOauthCredential, error) {
					return db.GithubOauthCredential{}, errors.New("database error")
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
			got, err := creategithuboauthcredentials.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			switch want := tt.want.(type) {
			case *model.CreateGitHubOAuthCredentialsPayload:
				gotPayload, ok := got.(*model.CreateGitHubOAuthCredentialsPayload)
				require.True(t, ok, "expected CreateGitHubOAuthCredentialsPayload")
				require.Equal(t, want.Credentials.ID, gotPayload.Credentials.ID)
				require.Equal(t, want.Credentials.Name, gotPayload.Credentials.Name)
				require.Equal(t, want.Credentials.ClientID, gotPayload.Credentials.ClientID)
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
