package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"trip2g/internal/appreq"
	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/getboostyuser"
	"trip2g/internal/case/getpatreonuser"
	"trip2g/internal/case/signinbytgauthtoken"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/vektah/gqlparser/gqlerror"
)

func (a *app) setTokenValidator() {
	a.tokenManager.AddValidator(func(ctx context.Context, data *usertoken.Data) error {
		ban, banErr := a.UserBanByUserID(ctx, int64(data.ID))
		if banErr != nil {
			return fmt.Errorf("failed to get user ban: %w", banErr)
		}

		if ban != nil {
			return gqlerror.Errorf("%s", ban.Reason)
		}

		return nil
	})
}

func (a *app) UpsertAPIKeyLogAction(ctx context.Context, name string) error {
	if txEnv := a.txEnvFromCtx(ctx); txEnv != nil {
		return txEnv.WriteQueries.UpsertAPIKeyLogAction(ctx, name)
	}
	return a.WriteQueries.UpsertAPIKeyLogAction(ctx, name)
}

func (a *app) UpsertAPIKeyLogIP(ctx context.Context, ip string) error {
	if txEnv := a.txEnvFromCtx(ctx); txEnv != nil {
		return txEnv.WriteQueries.UpsertAPIKeyLogIP(ctx, ip)
	}
	return a.WriteQueries.UpsertAPIKeyLogIP(ctx, ip)
}

func (a *app) CurrentUserToken(ctx context.Context) (*usertoken.Data, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	return req.UserToken()
}

var ErrNotAdmin = errors.New("unauthorized")

func (a *app) CanReadNote(ctx context.Context, note *model.NoteView) (bool, error) {
	return canreadnote.Resolve(ctx, a, note)
}

func (a *app) CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	data, err := req.UserToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	if data != nil {
		if !data.IsAdmin() {
			a.log.Warn("unauthorized access attempt", "user_id", data.ID, "role", data.Role)
			return nil, ErrNotAdmin
		}
		return data, nil
	}

	// TODO: refactor — using AdminActorUserID as a fallback is a stopgap; proper
	// actor identity should flow through the context without this field.
	// API key path: no user session, verify the key owner is actually an admin in DB.
	if req.AdminActorUserID != 0 {
		_, err = a.AdminByUserID(ctx, int64(req.AdminActorUserID))
		if db.IsNoFound(err) {
			return nil, ErrNotAdmin
		}
		if err != nil {
			return nil, fmt.Errorf("failed to verify admin: %w", err)
		}
		return &usertoken.Data{ID: req.AdminActorUserID, Role: "admin"}, nil
	}

	return nil, ErrNotAdmin
}

func (a *app) GenerateHotAuthToken(_ context.Context, data model.HotAuthToken) (string, error) {
	return a.hotAuthTokenManager.NewToken(data)
}

func (a *app) ParseHotAuthToken(_ context.Context, token string) (*model.HotAuthToken, error) {
	return a.hotAuthTokenManager.ParseToken(token)
}

func (a *app) GenerateTgAuthURL(_ context.Context, path string, data model.TgAuthToken) (string, error) {
	rawToken, err := a.tgAuthTokenManager.NewToken(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate Telegram auth token: %w", err)
	}

	publicURL := a.PublicURL()
	if publicURL == "" {
		a.log.Warn("GenerateTgAuthURL: empty public URL")
		publicURL = "https://example.com" // Fallback URL, must has a https scheme
	}

	u, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse public URL: %w", err)
	}

	u.Path = path

	query := u.Query()
	query.Set(signinbytgauthtoken.QueryParam, rawToken)

	u.RawQuery = query.Encode()

	return u.String(), nil
}

func (a *app) ParseTgAuthToken(ctx context.Context, token string) (*model.TgAuthToken, error) {
	return a.tgAuthTokenManager.ParseToken(token)
}

func (a *app) SetupUserToken(ctx context.Context, userID int64) (string, error) {
	role := "user"

	_, err := a.queries.AdminByUserID(ctx, userID)
	if err != nil {
		if !db.IsNoFound(err) {
			return "", fmt.Errorf("failed to get admin by user ID: %w", err)
		}
	} else {
		role = "admin"
	}

	data := usertoken.Data{
		ID:   int(userID),
		Role: role,
	}

	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return "", err
	}

	storeData, err := req.TokenManager.Store(req.Req, data)
	if err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	req.SetUserToken(&storeData.Data)

	return storeData.JWT, nil
}

func (a *app) ResetUserToken(ctx context.Context) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return err
	}

	err = req.TokenManager.Delete(req.Req)
	if err != nil {
		return fmt.Errorf("failed to reset token: %w", err)
	}

	return nil
}

func (a *app) Insecure() bool {
	return a.config.UserToken.Insecure
}

var ErrFailedGeneration = errors.New("failed to generate code")

// DevSignInCode is the fixed sign-in code used in dev mode (DevMode=true).
// Defined as a package-level const so the sign-in bypass path can reference
// it without duplicating the literal.
const DevSignInCode = "111111"

func generateSixDigitCode() (int64, error) {
	for range 100 {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			return 0, fmt.Errorf("failed to read random bytes: %w", err)
		}
		n := binary.BigEndian.Uint32(b[:]) % 1000000
		if n >= 100000 {
			return int64(n), nil
		}
	}

	return 0, ErrFailedGeneration
}

func generateEightCharCode() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func (a *app) CreateSignInCode(ctx context.Context, userID int64) (string, error) {
	code, err := generateSixDigitCode()
	if err != nil {
		return "", err
	}

	sCode := strconv.Itoa(int(code))
	if a.config.DevMode {
		sCode = DevSignInCode
	}

	err = appreq.CtxEnv(ctx, a).InsertSignInCode(ctx, db.InsertSignInCodeParams{
		UserID: userID,
		Code:   sCode,
	})
	if err != nil {
		return "", fmt.Errorf("failed to insert sign-in code: %w", err)
	}

	return sCode, nil
}

// DevSignInBypass reports whether a sign-in code should be accepted without the
// sign_in_codes row dance. True only in dev mode for the fixed dev code, so the
// parallel-sign-in delete-all race can't occur in tests. Zero prod effect.
func (a *app) DevSignInBypass(code string) bool {
	return a.config.DevMode && code == DevSignInCode
}

func (a *app) TryToAutoRegisterUser(ctx context.Context, email string) (*db.User, error) {
	user, err := getboostyuser.Resolve(ctx, a, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check Boosty user: %w", err)
	}

	if user != nil {
		return user, nil
	}

	user, err = getpatreonuser.Resolve(ctx, a, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check Patreon user: %w", err)
	}

	// etc

	return user, nil
}

// ShortAPITokenSecret returns the secret used for signing short API tokens.
func (a *app) ShortAPITokenSecret() string {
	return a.config.UserToken.Secret
}

// lookupAPIKeyRow resolves value to its ApiKey row, trying the sha256-hashed
// form first (new keys) then falling back to a plaintext match (old keys).
// found is false when no row matches; a genuine DB error is never swallowed.
func (a *app) lookupAPIKeyRow(ctx context.Context, value string) (db.ApiKey, bool, error) {
	hash := sha256.Sum256([]byte(value))
	hashedValue := hex.EncodeToString(hash[:])

	row, err := a.Queries.ApiKeyByValue(ctx, hashedValue)
	if err != nil && !db.IsNoFound(err) {
		return db.ApiKey{}, false, fmt.Errorf("resolve api key: %w", err)
	}
	if db.IsNoFound(err) {
		row, err = a.Queries.ApiKeyByValue(ctx, value)
		if err != nil && !db.IsNoFound(err) {
			return db.ApiKey{}, false, fmt.Errorf("resolve api key (plain): %w", err)
		}
		if db.IsNoFound(err) {
			return db.ApiKey{}, false, nil
		}
	}

	return row, true, nil
}

// ValidAPIKey reports whether plainKey is a currently valid API key. Unlike
// ResolveAPIKey it writes no api_key_log entry — asset downloads are too hot
// a path to log per-request.
func (a *app) ValidAPIKey(ctx context.Context, plainKey string) (bool, error) {
	_, found, err := a.lookupAPIKeyRow(ctx, plainKey)
	if err != nil {
		return false, err
	}
	return found, nil
}

// ResolveAPIKey resolves an API key by value (tries sha256 hash first, then plain),
// logs the action and IP, and returns the key record.
func (a *app) ResolveAPIKey(ctx context.Context, value, action string) (*db.ApiKey, error) {
	apiKey, found, err := a.lookupAPIKeyRow(ctx, value)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("invalid API key")
	}

	req, _ := appreq.FromCtx(ctx)
	ip := req.Req.RemoteIP().String()

	if logErr := a.WriteQueries.UpsertAPIKeyLogAction(ctx, action); logErr != nil {
		return nil, fmt.Errorf("log action: %w", logErr)
	}
	if logErr := a.WriteQueries.UpsertAPIKeyLogIP(ctx, ip); logErr != nil {
		return nil, fmt.Errorf("log ip: %w", logErr)
	}
	if logErr := a.WriteQueries.InsertAPIKeyLog(ctx, db.InsertAPIKeyLogParams{
		ApiKeyID: apiKey.ID,
		Action:   action,
		Ip:       ip,
	}); logErr != nil {
		return nil, fmt.Errorf("insert log: %w", logErr)
	}

	return &apiKey, nil
}

// UserID returns the current user ID from the request context, or nil if anonymous.
func (a *app) UserID(ctx context.Context) *int64 {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil
	}
	token, err := req.UserToken()
	if err != nil || token == nil || token.ID == 0 {
		return nil
	}
	id := int64(token.ID)
	return &id
}

// IsAdmin reports whether the current request's user token has admin role.
func (a *app) IsAdmin(ctx context.Context) bool {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return false
	}
	token, err := req.UserToken()
	if err != nil {
		return false
	}
	return token.IsAdmin()
}

// CurrentFederatedScope returns the inbound federation AllowedSubgraphs and
// ok=true when the request carries a federated-scoped identity.
func (a *app) CurrentFederatedScope(ctx context.Context) ([]string, bool) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, false
	}
	return req.FederatedScope()
}
