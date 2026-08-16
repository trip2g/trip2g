package createhatlink_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/createhatlink"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createhatlink_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	GenerateHotAuthTokenWithTTL(ctx context.Context, data appmodel.HotAuthToken, ttl time.Duration) (string, error)
	PublicURL() string
	AuditLogger() logger.Logger
}

type envMock = EnvMock

func newEnv() *envMock {
	return &envMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 42, Role: "admin"}, nil
		},
		GenerateHotAuthTokenWithTTLFunc: func(ctx context.Context, data appmodel.HotAuthToken, ttl time.Duration) (string, error) {
			return "jwt-token", nil
		},
		PublicURLFunc: func() string {
			return "https://example.com/"
		},
		AuditLoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name          string
		env           *envMock
		input         model.CreateHatLinkInput
		want          model.CreateHatLinkOrErrorPayload
		wantTTL       time.Duration
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name:  "mints a link with the default expiry",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "owner@example.com"},
			want: &model.CreateHatLinkPayload{
				URL: "https://example.com/_system/hat?token=jwt-token",
			},
			wantTTL: 5 * time.Minute,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.GenerateHotAuthTokenWithTTLCalls(), 1)
				require.Equal(t, appmodel.HotAuthToken{Email: "owner@example.com", Redirect: "/"},
					mockEnv.GenerateHotAuthTokenWithTTLCalls()[0].Data)
			},
		},
		{
			name:    "the link never provisions, whoever asks for it",
			env:     newEnv(),
			input:   model.CreateHatLinkInput{Email: "owner@example.com"},
			want:    &model.CreateHatLinkPayload{URL: "https://example.com/_system/hat?token=jwt-token"},
			wantTTL: 5 * time.Minute,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.False(t, mockEnv.GenerateHotAuthTokenWithTTLCalls()[0].Data.AdminEnter,
					"the admin API minted a token that creates a user and grants admin")
			},
		},
		{
			name:    "a redirect is carried into the token",
			env:     newEnv(),
			input:   model.CreateHatLinkInput{Email: "owner@example.com", RedirectURL: ptr("/admin/users")},
			want:    &model.CreateHatLinkPayload{URL: "https://example.com/_system/hat?token=jwt-token"},
			wantTTL: 5 * time.Minute,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, appmodel.HotAuthToken{Email: "owner@example.com", Redirect: "/admin/users"},
					mockEnv.GenerateHotAuthTokenWithTTLCalls()[0].Data)
			},
		},
		{
			name:  "an off-site redirect is a validation error",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "owner@example.com", RedirectURL: ptr("//evil.example.com/")},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "redirectUrl", Value: "must be a path on this site, starting with a single /"}},
			},
		},
		{
			name:  "an absolute redirect is a validation error",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "owner@example.com", RedirectURL: ptr("https://evil.example.com/")},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "redirectUrl", Value: "must be a path on this site, starting with a single /"}},
			},
		},
		{
			name:    "custom expiry within the cap is honoured",
			env:     newEnv(),
			input:   model.CreateHatLinkInput{Email: "owner@example.com", ExpiresInMinutes: ptr(int32(60))},
			want:    &model.CreateHatLinkPayload{URL: "https://example.com/_system/hat?token=jwt-token"},
			wantTTL: 60 * time.Minute,
		},
		{
			name:  "invalid email is a validation error",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "not-an-email"},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "email", Value: "must be a valid email address"}},
			},
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.GenerateHotAuthTokenWithTTLCalls())
			},
		},
		{
			name:  "empty email is a validation error",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: ""},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "email", Value: "cannot be blank"}},
			},
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.GenerateHotAuthTokenWithTTLCalls())
			},
		},
		{
			name:  "expiry above the cap is rejected",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "owner@example.com", ExpiresInMinutes: ptr(int32(61))},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "expiresInMinutes", Value: "must be between 1 and 60"}},
			},
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.GenerateHotAuthTokenWithTTLCalls())
			},
		},
		{
			name:  "zero expiry is rejected",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "owner@example.com", ExpiresInMinutes: ptr(int32(0))},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "expiresInMinutes", Value: "must be between 1 and 60"}},
			},
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.GenerateHotAuthTokenWithTTLCalls())
			},
		},
		{
			name:  "negative expiry is rejected",
			env:   newEnv(),
			input: model.CreateHatLinkInput{Email: "owner@example.com", ExpiresInMinutes: ptr(int32(-5))},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "expiresInMinutes", Value: "must be between 1 and 60"}},
			},
		},
		{
			name: "non-admin caller is rejected before anything is minted",
			env: func() *envMock {
				env := newEnv()
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				}
				return env
			}(),
			input:   model.CreateHatLinkInput{Email: "owner@example.com"},
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.GenerateHotAuthTokenWithTTLCalls())
				require.Empty(t, mockEnv.AuditLoggerCalls())
			},
		},
		{
			name: "missing public URL is a system error",
			env: func() *envMock {
				env := newEnv()
				env.PublicURLFunc = func() string { return "" }
				return env
			}(),
			input:   model.CreateHatLinkInput{Email: "owner@example.com"},
			wantErr: true,
		},
		{
			name: "token minting failure is a system error",
			env: func() *envMock {
				env := newEnv()
				env.GenerateHotAuthTokenWithTTLFunc = func(ctx context.Context, data appmodel.HotAuthToken, ttl time.Duration) (string, error) {
					return "", errors.New("boom")
				}
				return env
			}(),
			input:   model.CreateHatLinkInput{Email: "owner@example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			got, err := createhatlink.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if tt.afterCallback != nil {
					tt.afterCallback(t, tt.env)
				}
				return
			}

			require.NoError(t, err)

			if want, ok := tt.want.(*model.CreateHatLinkPayload); ok {
				payload, isPayload := got.(*model.CreateHatLinkPayload)
				require.True(t, isPayload, "expected CreateHatLinkPayload, got %T", got)
				require.Equal(t, want.URL, payload.URL)
				require.WithinRange(t, payload.ExpiresAt, before.Add(tt.wantTTL), time.Now().Add(tt.wantTTL))
				require.Equal(t, tt.wantTTL, tt.env.GenerateHotAuthTokenWithTTLCalls()[0].TTL)
			} else {
				require.Equal(t, tt.want, got)
			}

			if tt.afterCallback != nil {
				tt.afterCallback(t, tt.env)
			}
		})
	}
}

// TestResolve_AuditsWithoutLeakingTheLink pins the audit entry: it must record
// who minted the link and for whom, but never the token or the URL itself.
func TestResolve_AuditsWithoutLeakingTheLink(t *testing.T) {
	recorder := &recordingLogger{}
	env := newEnv()
	env.AuditLoggerFunc = func() logger.Logger { return recorder }

	got, err := createhatlink.Resolve(context.Background(), env, model.CreateHatLinkInput{
		Email:       "owner@example.com",
		RedirectURL: ptr("/admin/users"),
	})
	require.NoError(t, err)
	require.IsType(t, &model.CreateHatLinkPayload{}, got)

	require.Len(t, recorder.infos, 1)
	entry := recorder.infos[0]
	require.NotContains(t, entry.args, "jwt-token")
	require.NotContains(t, entry.args, "https://example.com/_system/hat?token=jwt-token")
	require.Contains(t, entry.args, "owner@example.com")
	require.Contains(t, entry.args, 42)
	require.Contains(t, entry.args, "/admin/users")
}

type logEntry struct {
	msg  string
	args []interface{}
}

type recordingLogger struct {
	infos []logEntry
}

func (l *recordingLogger) Info(msg string, args ...interface{}) {
	l.infos = append(l.infos, logEntry{msg: msg, args: args})
}
func (l *recordingLogger) Error(string, ...interface{}) {}
func (l *recordingLogger) Debug(string, ...interface{}) {}
func (l *recordingLogger) Warn(string, ...interface{})  {}
