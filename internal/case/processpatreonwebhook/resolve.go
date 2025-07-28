package processpatreonwebhook

import (
	"context"
	"fmt"

	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/valyala/fasthttp"
)

type Env interface {
	Logger() logger.Logger
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	refreshpatreondata.Env
}

type Response struct {
	Success bool `json:"success"`
}

func Resolve(reqCtx *fasthttp.RequestCtx, env Env, credentialID int64) (*Response, error) {
	ctx := context.Background()

	// Verify that the credentials exist
	credentials, err := env.PatreonCredentials(ctx, credentialID)
	if err != nil {
		if db.IsNoFound(err) {
			env.Logger().Warn("credentials not found", "credential_id", credentialID)
			return nil, fmt.Errorf("credentials not found")
		}
		return nil, fmt.Errorf("failed to get patreon credentials: %w", err)
	}

	env.Logger().Info("processing patreon webhook",
		"credential_id", credentialID,
	)

	// Call refreshpatreondata to sync the data
	err = refreshpatreondata.Resolve(ctx, env, &credentials.ID)
	if err != nil {
		env.Logger().Error("failed to refresh patreon data", "error", err, "credential_id", credentialID)
		return nil, fmt.Errorf("failed to refresh patreon data: %w", err)
	}

	return &Response{Success: true}, nil
}
