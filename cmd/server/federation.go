package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/federation"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
)

func (a *app) FederationSecretByKBURL(ctx context.Context, kbURL string) (db.FederationSecret, bool, error) {
	secret, err := a.Queries.FederationSecretByKBURL(ctx, &kbURL)
	if errors.Is(err, sql.ErrNoRows) {
		return db.FederationSecret{}, false, nil
	}
	if err != nil {
		return db.FederationSecret{}, false, err
	}
	return secret, true, nil
}

func (a *app) FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error) {
	secret, err := a.Queries.FederationSecretByKID(ctx, kid)
	if errors.Is(err, sql.ErrNoRows) {
		return db.FederationSecret{}, false, nil
	}
	if err != nil {
		return db.FederationSecret{}, false, err
	}
	return secret, true, nil
}

func (a *app) HasFederationSecretForKBURL(ctx context.Context, kbURL string) (bool, error) {
	rows, err := a.Queries.ListFederationSecrets(ctx)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.KbUrl != nil && *row.KbUrl == kbURL {
			return true, nil
		}
	}
	return false, nil
}

func (a *app) FederationClient(reqCtx context.Context, kbID string) (model.Federation, error) {
	nvs := a.LatestNoteViews()
	if nvs == nil {
		return nil, fmt.Errorf("federation kb %q not found", kbID)
	}

	depth := mcp.FederationDepthFromContext(reqCtx)

	for _, kb := range nvs.MCPFederationNotes {
		if kb == nil || kb.ID != kbID {
			continue
		}

		if kb.MaxDepth > 0 && depth >= kb.MaxDepth {
			return nil, fmt.Errorf("federation kb %q max depth %d exceeded", kbID, kb.MaxDepth)
		}

		peer := model.FederationPeer{
			KBID:   kb.ID,
			KBURL:  kb.URL,
			Issuer: a.PublicURL(),
			Depth:  depth,
		}

		dbCtx := a.ctx
		if dbCtx == nil {
			dbCtx = context.Background()
		}
		secretRow, ok, err := a.FederationSecretByKBURL(dbCtx, kb.URL)
		if err != nil {
			return nil, fmt.Errorf("get federation secret by kb url: %w", err)
		}
		if ok {
			secret, decErr := a.DecryptData(secretRow.SecretCrypt)
			if decErr != nil {
				return nil, decErr
			}
			peer.KID = secretRow.Kid
			peer.Secret = secret
		} else {
			configured, confErr := a.HasFederationSecretForKBURL(dbCtx, kb.URL)
			if confErr != nil {
				return nil, fmt.Errorf("check federation secret history by kb url: %w", confErr)
			}
			if configured {
				return nil, fmt.Errorf("no active federation secret for kb_id %q; the configured secret may be revoked", kbID)
			}
		}

		return federation.NewClient(peer, a.fedHTTPClient, a.config.DevMode), nil
	}

	return nil, fmt.Errorf("federation kb %q not found", kbID)
}

func (a *app) FederationMaxDepth() int {
	return a.config.MCPFederationMaxDepth
}

func (a *app) MCPMetrics() *metrics.MCPMetrics {
	return a.mcpMetrics
}
