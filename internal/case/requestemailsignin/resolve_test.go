package requestemailsignin

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg requestemailsignin . Env

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

func ptr(s string) *string { return &s }

// fullFlowEnv returns an EnvMock wired for a successful sign-in (no ban, ≤3 codes, happy path).
func fullFlowEnv(siteKey string) *EnvMock {
	return &EnvMock{
		TurnstileSiteKeyFunc:               func() string { return siteKey },
		IncrementAndCheckSigninCounterFunc: func() bool { return false },
		UserByEmailFunc: func(_ context.Context, _ string) (db.User, error) {
			return db.User{ID: 1}, nil
		},
		UserBanByUserIDFunc: func(_ context.Context, _ int64) (*db.UserBan, error) {
			return nil, nil
		},
		CountActiveSignInCodesFunc: func(_ context.Context, _ int64) (int64, error) {
			return 0, nil
		},
		MaxActiveSignInCodesFunc: func() int64 { return 3 },
		SMTPHostFunc:             func() string { return "smtp.example.com" },
		LogSignInCodesFunc:       func() bool { return false },
		CreateSignInCodeFunc: func(_ context.Context, _ int64) (string, error) {
			return "ABC123", nil
		},
		EnqueueRequestSignInEmailFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
	}
}

// TestCaptcha_BelowThreshold_NoToken verifies that when the counter is below the
// threshold no captcha is demanded and the normal flow completes successfully.
func TestCaptcha_BelowThreshold_NoToken(t *testing.T) {
	env := fullFlowEnv("site-key")
	// Counter below threshold — returns false.
	env.IncrementAndCheckSigninCounterFunc = func() bool { return false }

	input := Input{Email: "user@example.com"}

	result, err := Resolve(context.Background(), env, input, "1.2.3.4")
	require.NoError(t, err)

	payload, ok := result.(*model.RequestEmailSignInCodePayload)
	require.True(t, ok, "expected RequestEmailSignInCodePayload, got %T", result)
	require.True(t, payload.Success)
}

// TestCaptcha_AboveThreshold_NoToken verifies that when the counter exceeds the
// threshold and no token is provided, RequestCaptchaPayload is returned.
func TestCaptcha_AboveThreshold_NoToken(t *testing.T) {
	env := fullFlowEnv("site-key-abc")
	// Counter exceeded — returns true.
	env.IncrementAndCheckSigninCounterFunc = func() bool { return true }

	input := Input{Email: "user@example.com"}

	result, err := Resolve(context.Background(), env, input, "1.2.3.4")
	require.NoError(t, err)

	captchaPayload, ok := result.(*model.RequestCaptchaPayload)
	require.True(t, ok, "expected RequestCaptchaPayload, got %T", result)
	require.Equal(t, "site-key-abc", captchaPayload.SiteKey)
}

// TestCaptcha_AboveThreshold_ValidToken verifies that a valid captcha token bypasses
// the counter check and the normal flow proceeds.
func TestCaptcha_AboveThreshold_ValidToken(t *testing.T) {
	env := fullFlowEnv("site-key")
	env.VerifyCaptchaFunc = func(_ context.Context, _, _ string) error {
		return nil // valid token
	}

	token := "valid-token"
	input := Input{Email: "user@example.com", CaptchaToken: ptr(token)}

	result, err := Resolve(context.Background(), env, input, "1.2.3.4")
	require.NoError(t, err)

	payload, ok := result.(*model.RequestEmailSignInCodePayload)
	require.True(t, ok, "expected RequestEmailSignInCodePayload, got %T", result)
	require.True(t, payload.Success)
}

// TestCaptcha_AboveThreshold_InvalidToken verifies that an invalid captcha token
// causes an ErrorPayload with message "captcha_invalid" to be returned.
func TestCaptcha_AboveThreshold_InvalidToken(t *testing.T) {
	env := fullFlowEnv("site-key")
	env.VerifyCaptchaFunc = func(_ context.Context, _, _ string) error {
		return errors.New("turnstile rejected")
	}

	token := "bad-token"
	input := Input{Email: "user@example.com", CaptchaToken: ptr(token)}

	result, err := Resolve(context.Background(), env, input, "1.2.3.4")
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.Equal(t, "captcha_invalid", errPayload.Message)
}

// TestMaxActiveSignInCodes_LimitEnforced verifies that when the active code count
// exceeds the configured limit, too_many_sign_in_codes is returned.
func TestMaxActiveSignInCodes_LimitEnforced(t *testing.T) {
	env := fullFlowEnv("")
	env.MaxActiveSignInCodesFunc = func() int64 { return 3 }
	env.CountActiveSignInCodesFunc = func(_ context.Context, _ int64) (int64, error) {
		return 4, nil // exceeds limit of 3
	}

	result, err := Resolve(context.Background(), env, Input{Email: "user@example.com"}, "")
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.Equal(t, "too_many_sign_in_codes", errPayload.Message)
}

// TestMaxActiveSignInCodes_BelowLimit verifies that when the active code count
// is below the configured limit, sign-in proceeds successfully.
func TestMaxActiveSignInCodes_BelowLimit(t *testing.T) {
	env := fullFlowEnv("")
	env.MaxActiveSignInCodesFunc = func() int64 { return 3 }
	env.CountActiveSignInCodesFunc = func(_ context.Context, _ int64) (int64, error) {
		return 2, nil // within limit of 3
	}

	result, err := Resolve(context.Background(), env, Input{Email: "user@example.com"}, "")
	require.NoError(t, err)

	payload, ok := result.(*model.RequestEmailSignInCodePayload)
	require.True(t, ok, "expected RequestEmailSignInCodePayload, got %T", result)
	require.True(t, payload.Success)
}

// At count == limit, creating one more code would exceed the configured
// maximum. The boundary therefore has to reject before CreateSignInCode.
func TestMaxActiveSignInCodes_AtLimitRejected(t *testing.T) {
	env := fullFlowEnv("")
	env.MaxActiveSignInCodesFunc = func() int64 { return 3 }
	env.CountActiveSignInCodesFunc = func(_ context.Context, _ int64) (int64, error) {
		return 3, nil
	}

	result, err := Resolve(context.Background(), env, Input{Email: "user@example.com"}, "")
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "count == limit must return ErrorPayload, got %T", result)
	require.Equal(t, "too_many_sign_in_codes", errPayload.Message)
	require.Empty(t, env.CreateSignInCodeCalls(), "must not create a code once the limit is reached")
}

// TestNoSMTP_LogSignInCodesDisabled_ReturnsError verifies that when no SMTP is
// configured and code logging is disabled, no code can ever be delivered, so
// Resolve returns an ErrorPayload without creating or queueing a code.
func TestNoSMTP_LogSignInCodesDisabled_ReturnsError(t *testing.T) {
	env := fullFlowEnv("")
	env.SMTPHostFunc = func() string { return "" }
	env.LogSignInCodesFunc = func() bool { return false }

	result, err := Resolve(context.Background(), env, Input{Email: "user@example.com"}, "")
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.Equal(t, "Email delivery isn't configured on this server, so no code can be sent. Sign in with Google or GitHub instead, or ask the site admin.", errPayload.Message)
	require.Empty(t, env.EnqueueRequestSignInEmailCalls(), "must not queue a code that can't be delivered")
}

// TestNoSMTP_LogSignInCodesEnabled_ProceedsNormally verifies that when SMTP is
// not configured but code logging is enabled, the code is still logged
// server-side, so Resolve proceeds to the normal success path.
func TestNoSMTP_LogSignInCodesEnabled_ProceedsNormally(t *testing.T) {
	env := fullFlowEnv("")
	env.SMTPHostFunc = func() string { return "" }
	env.LogSignInCodesFunc = func() bool { return true }

	result, err := Resolve(context.Background(), env, Input{Email: "user@example.com"}, "")
	require.NoError(t, err)

	payload, ok := result.(*model.RequestEmailSignInCodePayload)
	require.True(t, ok, "expected RequestEmailSignInCodePayload, got %T", result)
	require.True(t, payload.Success)
}

// TestSMTPConfigured_ProceedsNormally verifies that when SMTP is configured,
// the normal success path is unaffected regardless of LogSignInCodes.
func TestSMTPConfigured_ProceedsNormally(t *testing.T) {
	env := fullFlowEnv("")
	env.SMTPHostFunc = func() string { return "smtp.example.com" }
	env.LogSignInCodesFunc = func() bool { return false }

	result, err := Resolve(context.Background(), env, Input{Email: "user@example.com"}, "")
	require.NoError(t, err)

	payload, ok := result.(*model.RequestEmailSignInCodePayload)
	require.True(t, ok, "expected RequestEmailSignInCodePayload, got %T", result)
	require.True(t, payload.Success)
}
