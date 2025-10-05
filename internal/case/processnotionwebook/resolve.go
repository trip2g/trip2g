package processnotionwebhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"trip2g/internal/db"
)

type Env interface {
	NotionIntegration(ctx context.Context, id int64) (db.NotionIntegration, error)
	UpdateNotionIntegrationVerificationToken(ctx context.Context, params db.UpdateNotionIntegrationVerificationTokenParams) error
}

type Request struct {
	ID   string
	Body []byte
}

var ErrIntegrationNotEnabled = errors.New("notion integration not enabled")

func Resolve(ctx context.Context, env Env, req Request) error {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	integration, err := env.NotionIntegration(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get notion integration: %w", err)
	}

	if !integration.Enabled {
		return ErrIntegrationNotEnabled
	}

	skip, err := updateVerificationToken(ctx, env, req, &integration)
	if err != nil {
		return fmt.Errorf("failed to update verification token: %w", err)
	}

	if skip {
		return nil
	}

	return nil
}

func updateVerificationToken(ctx context.Context, env Env, req Request, ni *db.NotionIntegration) (bool, error) {
	var data struct {
		VerificationToken string `json:"verification_token"`
	}

	err := json.Unmarshal(req.Body, &data)
	if err != nil {
		return false, fmt.Errorf("failed to parse verification token: %w", err)
	}

	if data.VerificationToken != "" {
		params := db.UpdateNotionIntegrationVerificationTokenParams{
			ID: ni.ID,

			VerificationToken: sql.NullString{
				Valid:  true,
				String: data.VerificationToken,
			},
		}

		err = env.UpdateNotionIntegrationVerificationToken(ctx, params)
		if err != nil {
			return false, fmt.Errorf("failed to update verification token: %w", err)
		}

		return true, nil
	}

	return false, nil
}
