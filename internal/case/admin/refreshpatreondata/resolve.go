package refreshpatreondata

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/patreon"
)

type Env interface {
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	PatreonListCampaigns(token string) ([]patreon.Campaign, error)
	UpsertPatreonCampaign(ctx context.Context, arg db.UpsertPatreonCampaignParams) error
}

// Payload is an alias for RefreshPatreonDataOrErrorPayload for cleaner code.
type Payload = model.RefreshPatreonDataOrErrorPayload

func Resolve(ctx context.Context, env Env, credentialsID int64) (Payload, error) {
	// Get the credentials
	credentials, err := env.PatreonCredentials(ctx, credentialsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get patreon credentials: %w", err)
	}

	// Fetch campaigns from Patreon
	campaigns, err := env.PatreonListCampaigns(credentials.CreatorAccessToken)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Failed to fetch campaigns from Patreon: %v", err)}, nil
	}

	// The upsert operation will handle keeping campaigns fresh
	// missed_at will remain null for campaigns that are upserted

	// Upsert fresh campaigns
	for _, campaign := range campaigns {
		attributesJSON, err := json.Marshal(campaign.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal campaign attributes for %s: %w", campaign.ID, err)
		}

		params := db.UpsertPatreonCampaignParams{
			CredentialsID: credentialsID,
			CampaignID:    campaign.ID,
			Attributes:    string(attributesJSON),
		}

		err = env.UpsertPatreonCampaign(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert campaign %s: %w", campaign.ID, err)
		}
	}

	// Define payload as separate variable
	payload := model.RefreshPatreonDataPayload{
		Success: true,
	}

	return &payload, nil
}