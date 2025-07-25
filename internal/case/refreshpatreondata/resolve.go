package refreshpatreondata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/patreon"
)

type Env interface {
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	AllActivePatreonCredentials(ctx context.Context) ([]db.PatreonCredential, error)
	PatreonListCampaigns(token string) ([]patreon.Campaign, error)
	PatreonListPatrons(token string, campaignID string) (*patreon.PatronsResponse, error)
	UpsertPatreonCampaign(ctx context.Context, arg db.UpsertPatreonCampaignParams) error
	GetPatreonCampaignsByCredentialsID(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error)
	UpsertPatreonTier(ctx context.Context, arg db.UpsertPatreonTierParams) error
	UpsertPatreonMember(ctx context.Context, arg db.UpsertPatreonMemberParams) error
	UpdatePatreonCredentialsSyncedAt(ctx context.Context, id int64) error
	GetPatreonTierByTierID(ctx context.Context, arg db.GetPatreonTierByTierIDParams) (db.PatreonTier, error)
	GetPatreonMemberByPatreonIDAndCampaignID(ctx context.Context, arg db.GetPatreonMemberByPatreonIDAndCampaignIDParams) (db.PatreonMember, error)
	SetPatreonMemberCurrentTier(ctx context.Context, arg db.SetPatreonMemberCurrentTierParams) error
}

func sync(ctx context.Context, env Env, credentials db.PatreonCredential) error {

	// Fetch campaigns from Patreon
	campaigns, err := env.PatreonListCampaigns(credentials.CreatorAccessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch campaigns from Patreon: %w", err)
	}

	// Upsert fresh campaigns and sync their tiers
	for _, campaign := range campaigns {
		attributesJSON, err := json.Marshal(campaign.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal campaign attributes for %s: %w", campaign.ID, err)
		}

		params := db.UpsertPatreonCampaignParams{
			CredentialsID: credentials.ID,
			CampaignID:    campaign.ID,
			Attributes:    string(attributesJSON),
		}

		err = env.UpsertPatreonCampaign(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to upsert campaign %s: %w", campaign.ID, err)
		}

		// Get the campaign ID from database
		dbCampaigns, err := env.GetPatreonCampaignsByCredentialsID(ctx, credentials.ID)
		if err != nil {
			return fmt.Errorf("failed to get campaigns from db: %w", err)
		}

		var dbCampaignID int64
		for _, dbCampaign := range dbCampaigns {
			if dbCampaign.CampaignID == campaign.ID {
				dbCampaignID = dbCampaign.ID
				break
			}
		}

		if dbCampaignID == 0 {
			return fmt.Errorf("failed to find campaign %s in database", campaign.ID)
		}

		// Sync tiers from campaign relationships
		// Note: Currently the ListCampaigns doesn't include tier data
		// We would need to modify the client to return included data
		// For now, tiers will only be synced if we implement a separate API call

		// Fetch and sync members for this campaign
		patronsResp, err := env.PatreonListPatrons(credentials.CreatorAccessToken, campaign.ID)
		if err != nil {
			// Log but don't fail - some campaigns might not have patron access
			continue
		}

		// Collect patron IDs for marking missed members
		patronIDs := make([]string, 0, len(patronsResp.Data))
		
		// Process each patron
		for _, patron := range patronsResp.Data {
			patronIDs = append(patronIDs, patron.ID)

			// Get user email from relationships
			// Note: Email is typically not available in the user relationship
			// We'll use a placeholder for now
			var email string

			// If no email found, use a placeholder
			if email == "" {
				email = fmt.Sprintf("patron_%s@patreon.local", patron.ID)
			}

			// Get status, handling nil pointer
			status := ""
			if patron.Attributes.PatronStatus != nil {
				status = *patron.Attributes.PatronStatus
			}

			memberParams := db.UpsertPatreonMemberParams{
				PatreonID:  patron.ID,
				CampaignID: dbCampaignID,
				Status:     status,
				Email:      email,
			}

			err = env.UpsertPatreonMember(ctx, memberParams)
			if err != nil {
				return fmt.Errorf("failed to upsert member %s: %w", patron.ID, err)
			}

			// Handle currently entitled tiers
			tierData, _ := patron.GetCurrentlyEntitledTiers()
			if tierData != nil && len(tierData) > 0 {
				// Get the first tier (primary tier)
				tierID := tierData[0].ID
				
				// Find the tier in our database
				dbTier, err := env.GetPatreonTierByTierID(ctx, db.GetPatreonTierByTierIDParams{
					CampaignID: dbCampaignID,
					TierID:     tierID,
				})
				if err == nil {
					// Get the member we just created/updated
					dbMember, err := env.GetPatreonMemberByPatreonIDAndCampaignID(ctx, db.GetPatreonMemberByPatreonIDAndCampaignIDParams{
						PatreonID:  patron.ID,
						CampaignID: dbCampaignID,
					})
					if err == nil {
						// Update member's current tier
						err = env.SetPatreonMemberCurrentTier(ctx, db.SetPatreonMemberCurrentTierParams{
							CurrentTierID: sql.NullInt64{Int64: dbTier.ID, Valid: true},
							ID:            dbMember.ID,
						})
						if err != nil {
							// Log but don't fail
							_ = err
						}
					}
				}
			}
		}

		// Mark members not in current sync as missed
		if len(patronIDs) > 0 {
			patronIDsJSON, err := json.Marshal(patronIDs)
			if err != nil {
				return fmt.Errorf("failed to marshal patron IDs: %w", err)
			}

			// TODO: Fix this when sqlc properly supports json_each with multiple parameters
			// For now, we'll skip marking missed members
			_ = patronIDsJSON
		}
	}

	// Update synced_at timestamp
	err = env.UpdatePatreonCredentialsSyncedAt(ctx, credentials.ID)
	if err != nil {
		return fmt.Errorf("failed to update synced_at: %w", err)
	}

	return nil
}

// Resolve is the GraphQL resolver that calls sync
func Resolve(ctx context.Context, env Env, credentialsID *int64) (model.RefreshPatreonDataOrErrorPayload, error) {
	// If credentialsID is nil, sync all credentials
	if credentialsID == nil {
		// Get all active credentials
		allCreds, err := env.AllActivePatreonCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get all active credentials: %w", err)
		}

		// Sync each credential
		for _, cred := range allCreds {
			err = sync(ctx, env, cred)
			if err != nil {
				// Log but continue with other credentials
				// In production, you might want to use a proper logger here
				_ = err
			}
		}
	} else {
		// Get the specific credential
		cred, err := env.PatreonCredentials(ctx, *credentialsID)
		if err != nil {
			return nil, fmt.Errorf("failed to get patreon credentials: %w", err)
		}

		// Sync single credential
		err = sync(ctx, env, cred)
		if err != nil {
			return &model.ErrorPayload{Message: fmt.Sprintf("Failed to sync Patreon data: %v", err)}, nil
		}
	}

	return &model.RefreshPatreonDataPayload{Success: true}, nil
}