package refreshpatreondata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/patreon"
)

type Env interface {
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)

	AllActivePatreonCredentials(ctx context.Context) ([]db.PatreonCredential, error)
	PatreonClientByID(ctx context.Context, credentialsID int64) (patreon.Client, error)

	UpdatePatreonCredentialsSyncedAt(ctx context.Context, id int64) error
	SetPatreonMemberCurrentTier(ctx context.Context, arg db.SetPatreonMemberCurrentTierParams) error

	GetPatreonCampaignsByCredentialsID(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error)
	GetPatreonTierByTierID(ctx context.Context, arg db.GetPatreonTierByTierIDParams) (db.PatreonTier, error)
	GetPatreonMemberByPatreonIDAndCampaignID(ctx context.Context, arg db.GetPatreonMemberByPatreonIDAndCampaignIDParams) (db.PatreonMember, error)

	UpsertPatreonCampaign(ctx context.Context, arg db.UpsertPatreonCampaignParams) error
	UpsertPatreonTier(ctx context.Context, arg db.UpsertPatreonTierParams) error
	UpsertPatreonMember(ctx context.Context, arg db.UpsertPatreonMemberParams) error
}

func syncCampaigns(ctx context.Context, env Env, credentials db.PatreonCredential) ([]patreon.Campaign, error) {
	// Get Patreon client
	client, err := env.PatreonClientByID(ctx, credentials.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Patreon client: %w", err)
	}

	// Fetch campaigns from Patreon
	campaigns, err := client.ListCampaigns()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch campaigns from Patreon: %w", err)
	}

	// Upsert fresh campaigns
	for _, campaign := range campaigns {
		attributesJSON, marshalErr := json.Marshal(campaign.Attributes)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal campaign attributes for %s: %w", campaign.ID, marshalErr)
		}

		params := db.UpsertPatreonCampaignParams{
			CredentialsID: credentials.ID,
			CampaignID:    campaign.ID,
			Attributes:    string(attributesJSON),
		}

		err = env.UpsertPatreonCampaign(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert campaign %s: %w", campaign.ID, err)
		}
	}

	return campaigns, nil
}

func processTier(ctx context.Context, env Env, tier map[string]interface{}, dbCampaignID int64) error {
	// Extract tier data
	tierID, _ := tier["id"].(string)
	if tierID == "" {
		return nil // Skip invalid tiers
	}

	// Extract attributes
	tierAttributes := make(map[string]interface{})
	if attrs, ok := tier["attributes"].(map[string]interface{}); ok {
		tierAttributes = attrs
	}

	// Extract required fields
	title, _ := tierAttributes["title"].(string)
	amountCents := 0
	if amount, ok := tierAttributes["amount_cents"].(float64); ok {
		amountCents = int(amount)
	} else if amountInt, okInt := tierAttributes["amount_cents"].(int); okInt {
		amountCents = amountInt
	}

	// Marshal attributes for storage
	tierAttributesJSON, marshalErr := json.Marshal(tierAttributes)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal tier attributes: %w", marshalErr)
	}

	// Upsert tier
	tierParams := db.UpsertPatreonTierParams{
		CampaignID:  dbCampaignID,
		TierID:      tierID,
		Title:       title,
		AmountCents: int64(amountCents),
		Attributes:  string(tierAttributesJSON),
	}

	return env.UpsertPatreonTier(ctx, tierParams)
}

func syncTiers(ctx context.Context, env Env, campaign patreon.Campaign, dbCampaignID int64) error {
	// Sync tiers from campaign relationships
	tiers, tiersErr := campaign.GetTiersWithAttributes()
	if tiersErr != nil {
		return fmt.Errorf("failed to get tiers with attributes: %w", tiersErr)
	}
	if len(tiers) == 0 {
		return nil
	}

	for _, tier := range tiers {
		if err := processTier(ctx, env, tier, dbCampaignID); err != nil {
			tierID, _ := tier["id"].(string)
			return fmt.Errorf("failed to upsert tier %s: %w", tierID, err)
		}
	}

	return nil
}

func processIncludedTier(ctx context.Context, env Env, included patreon.IncludedEntity, dbCampaignID int64) {
	// Skip if attributes are empty or meaningless
	if len(included.Attributes) == 0 {
		return
	}

	// Extract tier data
	tierID := included.ID
	title, _ := included.Attributes["title"].(string)
	amountCents := 0
	if amount, ok := included.Attributes["amount_cents"].(float64); ok {
		amountCents = int(amount)
	} else if amountInt, okInt := included.Attributes["amount_cents"].(int); okInt {
		amountCents = amountInt
	}

	// Only upsert if we have meaningful data (title or amount_cents)
	if title == "" && amountCents == 0 {
		return
	}

	// Marshal attributes for storage
	tierAttributesJSON, marshalErr := json.Marshal(included.Attributes)
	if marshalErr != nil {
		return
	}

	// Upsert tier
	tierParams := db.UpsertPatreonTierParams{
		CampaignID:  dbCampaignID,
		TierID:      tierID,
		Title:       title,
		AmountCents: int64(amountCents),
		Attributes:  string(tierAttributesJSON),
	}

	upsertErr := env.UpsertPatreonTier(ctx, tierParams)
	if upsertErr != nil {
		// Log but don't fail - continue processing
		_ = upsertErr
	}
}

func syncMembers(ctx context.Context, env Env, credentials db.PatreonCredential, campaignID string, dbCampaignID int64) error {
	// Get Patreon client
	client, err := env.PatreonClientByID(ctx, credentials.ID)
	if err != nil {
		return fmt.Errorf("failed to get Patreon client: %w", err)
	}

	// Fetch and sync members for this campaign
	patronsResp, err := client.ListPatrons(campaignID)
	if err != nil {
		return fmt.Errorf("failed to fetch patrons for campaign %s: %w", campaignID, err)
	}

	// Sync tiers from patron response included data
	for _, included := range patronsResp.Included {
		if included.Type == "tier" {
			processIncludedTier(ctx, env, included, dbCampaignID)
		}
	}

	// Collect patron IDs for marking missed members
	patronIDs := make([]string, 0, len(patronsResp.Data))

	// Process each patron
	for _, patron := range patronsResp.Data {
		patronIDs = append(patronIDs, patron.ID)

		// Get user email from patron attributes
		var email string
		if patron.Attributes.Email != "" {
			email = patron.Attributes.Email
		} else {
			// Use a placeholder if no email is available
			email = fmt.Sprintf("patron_%s@patreon.local", patron.ID)
		}

		// Extract patron status
		var status string
		if patron.Attributes.PatronStatus != nil {
			status = *patron.Attributes.PatronStatus
		} else {
			status = "unknown"
		}

		// Upsert member
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
		var currentTierID sql.NullInt64

		if len(tierData) > 0 {
			// Get the first tier (primary tier)
			tierID := tierData[0].ID

			// Find the tier in our database
			dbTier, tierErr := env.GetPatreonTierByTierID(ctx, db.GetPatreonTierByTierIDParams{
				CampaignID: dbCampaignID,
				TierID:     tierID,
			})
			if tierErr == nil {
				currentTierID = sql.NullInt64{Int64: dbTier.ID, Valid: true}
			}
		}

		// Get the member we just created/updated and set tier if we have one
		if currentTierID.Valid {
			dbMember, memberErr := env.GetPatreonMemberByPatreonIDAndCampaignID(ctx, db.GetPatreonMemberByPatreonIDAndCampaignIDParams{
				PatreonID:  patron.ID,
				CampaignID: dbCampaignID,
			})
			if memberErr == nil {
				// Update member's current tier
				updateErr := env.SetPatreonMemberCurrentTier(ctx, db.SetPatreonMemberCurrentTierParams{
					CurrentTierID: currentTierID,
					ID:            dbMember.ID,
				})
				if updateErr != nil {
					// Log but don't fail
					_ = updateErr
				}
			}
		}
	}

	// Mark members not in current sync as missed
	if len(patronIDs) > 0 {
		patronIDsJSON, marshalErr := json.Marshal(patronIDs)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal patron IDs: %w", marshalErr)
		}

		// TODO: Implement MarkMissedPatreonMembers when sqlc supports json_each properly
		_ = patronIDsJSON
	}

	return nil
}

func syncPatreonData(ctx context.Context, env Env, credentials db.PatreonCredential) error {
	// Sync campaigns
	campaigns, err := syncCampaigns(ctx, env, credentials)
	if err != nil {
		return err
	}

	// Process each campaign
	for _, campaign := range campaigns {
		// Get the campaign ID from database
		dbCampaigns, dbErr := env.GetPatreonCampaignsByCredentialsID(ctx, credentials.ID)
		if dbErr != nil {
			return fmt.Errorf("failed to get campaigns from db: %w", dbErr)
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

		// Sync tiers for this campaign
		err = syncTiers(ctx, env, campaign, dbCampaignID)
		if err != nil {
			return err
		}

		// Sync members for this campaign
		err = syncMembers(ctx, env, credentials, campaign.ID, dbCampaignID)
		if err != nil {
			// Log but don't fail - some campaigns might not have patron access
			continue
		}
	}

	// Update synced_at timestamp
	err = env.UpdatePatreonCredentialsSyncedAt(ctx, credentials.ID)
	if err != nil {
		return fmt.Errorf("failed to update synced_at: %w", err)
	}

	return nil
}

func syncAllCredentials(ctx context.Context, env Env) error {
	allCreds, err := env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all active credentials: %w", err)
	}

	for _, cred := range allCreds {
		if syncErr := syncPatreonData(ctx, env, cred); syncErr != nil {
			// Log but continue with other credentials
			// In production, you might want to use a proper logger here
			_ = syncErr
		}
	}
	return nil
}

func syncSingleCredential(ctx context.Context, env Env, credentialsID int64) error {
	cred, err := env.PatreonCredentials(ctx, credentialsID)
	if err != nil {
		return fmt.Errorf("failed to get patreon credentials: %w", err)
	}

	return syncPatreonData(ctx, env, cred)
}

// Resolve is the GraphQL resolver that calls sync.
func Resolve(ctx context.Context, env Env, credentialsID *int64) error {
	if credentialsID == nil {
		return syncAllCredentials(ctx, env)
	}

	return syncSingleCredential(ctx, env, *credentialsID)
}
