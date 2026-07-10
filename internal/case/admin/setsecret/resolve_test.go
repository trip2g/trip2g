package setsecret_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/setsecret"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		value    string
		mockFunc func() *envMock
		wantErr  bool
	}{
		{
			name:  "success",
			key:   "change_webhooks:1:auth_token",
			value: "Bearer secret123",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.IsDevModeFunc = func() bool { return false }
				mock.WebhookByIDFunc = func(ctx context.Context, id int64) (db.ChangeWebhook, error) {
					require.Equal(t, int64(1), id)
					return db.ChangeWebhook{ID: 1, Url: "https://example.com/hook"}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.UpsertSecretFunc = func(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error) {
					require.Equal(t, "change_webhooks:1:auth_token", arg.Key)
					require.Equal(t, []byte("encrypted"), arg.ValueCrypt)
					require.Equal(t, int64(1), arg.CreatedBy)
					return db.Secret{Key: arg.Key}, nil
				}
				return mock
			},
		},
		{
			name:  "auth error",
			key:   "k",
			value: "v",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				}
				return mock
			},
			wantErr: true,
		},
		{
			name:  "encrypt error",
			key:   "k",
			value: "v",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return nil, errors.New("encrypt fail")
				}
				return mock
			},
			wantErr: true,
		},
		{
			name:  "db error",
			key:   "k",
			value: "v",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("enc"), nil
				}
				mock.UpsertSecretFunc = func(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error) {
					return db.Secret{}, errors.New("db error")
				}
				return mock
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errPayload, err := setsecret.Resolve(context.Background(), tt.mockFunc(), tt.key, tt.value)
			require.Nil(t, errPayload)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolveRejectsNonHTTPSWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		mockFunc func() *envMock
	}{
		{
			name: "http change webhook rejected",
			key:  "change_webhooks:7:auth_token",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.IsDevModeFunc = func() bool { return false }
				mock.WebhookByIDFunc = func(ctx context.Context, id int64) (db.ChangeWebhook, error) {
					require.Equal(t, int64(7), id)
					return db.ChangeWebhook{ID: 7, Url: "http://evil.example.com/hook"}, nil
				}
				return mock
			},
		},
		{
			name: "uppercase HTTP change webhook rejected",
			key:  "change_webhooks:7:auth_token",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.IsDevModeFunc = func() bool { return false }
				mock.WebhookByIDFunc = func(ctx context.Context, id int64) (db.ChangeWebhook, error) {
					return db.ChangeWebhook{ID: 7, Url: "HTTP://evil.example.com/hook"}, nil
				}
				return mock
			},
		},
		{
			name: "http cron webhook rejected",
			key:  "cron_webhooks:3:auth_token",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.IsDevModeFunc = func() bool { return false }
				mock.CronWebhookByIDFunc = func(ctx context.Context, id int64) (db.CronWebhook, error) {
					require.Equal(t, int64(3), id)
					return db.CronWebhook{ID: 3, Url: "http://evil.example.com/cron"}, nil
				}
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := tt.mockFunc()
			errPayload, err := setsecret.Resolve(context.Background(), mock, tt.key, "Bearer secret123")
			require.NoError(t, err)
			require.NotNil(t, errPayload)
			require.Len(t, errPayload.ByFields, 1)
			require.Equal(t, "key", errPayload.ByFields[0].Name)
			require.Contains(t, errPayload.ByFields[0].Value, "https is required")
			require.Empty(t, mock.UpsertSecretCalls(), "secret must not be stored for a non-HTTPS webhook")
		})
	}
}

func TestResolveDevModeAllowsHTTPWebhook(t *testing.T) {
	t.Parallel()

	mock := &envMock{}
	mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
		return &usertoken.Data{ID: 1}, nil
	}
	mock.IsDevModeFunc = func() bool { return true }
	mock.WebhookByIDFunc = func(ctx context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{ID: 7, Url: "http://fleet:9099/deliver"}, nil
	}
	mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
		return []byte("enc"), nil
	}
	mock.UpsertSecretFunc = func(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error) {
		return db.Secret{Key: arg.Key}, nil
	}

	errPayload, err := setsecret.Resolve(context.Background(), mock, "change_webhooks:7:auth_token", "v")
	require.NoError(t, err)
	require.Nil(t, errPayload)
	require.Len(t, mock.UpsertSecretCalls(), 1)
}
