package refreshboostytoken

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"trip2g/internal/boosty"
	"trip2g/internal/db"
)

type Env interface {
	BoostyCredentials(ctx context.Context, id int64) (db.BoostyCredential, error)
	UpdateBoostyCredentialsTokens(ctx context.Context, arg db.UpdateBoostyCredentialsTokensParams) (db.BoostyCredential, error)
	BoostyClientByCredentialsID(ctx context.Context, credentialID int64) (boosty.Client, error)
}

func Resolve(ctx context.Context, env Env, credentialID int64) error {
	// Get the current credential
	cred, err := env.BoostyCredentials(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("failed to get boosty credential: %w", err)
	}

	// Get client and refresh token
	client, err := env.BoostyClientByCredentialsID(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("failed to get boosty client: %w", err)
	}

	result, err := client.RefreshToken()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Parse the current auth data to update it
	var authData boosty.AuthData
	err = json.Unmarshal([]byte(cred.AuthData), &authData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal auth data: %w", err)
	}

	// Update auth data with new tokens
	authData.AccessToken = result.AccessToken
	authData.RefreshToken = result.RefreshToken

	// Serialize updated auth data
	updatedAuthData, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("failed to marshal updated auth data: %w", err)
	}

	// Calculate expires_at (current time + expires_in seconds)
	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	// Update credentials in database
	_, err = env.UpdateBoostyCredentialsTokens(ctx, db.UpdateBoostyCredentialsTokensParams{
		AuthData: string(updatedAuthData),
		ExpiresAt: sql.NullTime{
			Time:  expiresAt,
			Valid: true,
		},
		ID: credentialID,
	})
	if err != nil {
		return fmt.Errorf("failed to update boosty credentials: %w", err)
	}

	return nil
}
