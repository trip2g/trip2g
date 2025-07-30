package refreshboostydata

import (
	"context"
	"encoding/json"
	"fmt"
	"trip2g/internal/boosty"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Env interface {
	BoostyClientByCredentialsID(ctx context.Context, credentialsID int64) (boosty.Client, error)

	// Boosty tier operations
	UpsertBoostyTier(ctx context.Context, arg db.UpsertBoostyTierParams) error
	MarkBoostyTiersAsMissed(ctx context.Context, arg db.MarkBoostyTiersAsMissedParams) error

	// Boosty member operations
	UpsertBoostyMember(ctx context.Context, arg db.UpsertBoostyMemberParams) error
	MarkBoostyMembersAsMissed(ctx context.Context, boostyIDs []int64) error

	Logger() logger.Logger
}

func syncBoostyData(ctx context.Context, env Env, credentialsID int64) error {
	// Get Boosty client
	client, err := env.BoostyClientByCredentialsID(ctx, credentialsID)
	if err != nil {
		return fmt.Errorf("failed to get Boosty client: %w", err)
	}

	// Fetch all subscribers
	subscribers, err := client.Subscribers()
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	env.Logger().Debug("fetched subscribers from Boosty", "count", len(subscribers))

	// Collect unique tiers and member IDs
	tierBoostyIDs := make(map[int64]bool)
	memberBoostyIDs := make([]int64, 0, len(subscribers))

	// Process each subscriber
	for _, subscriber := range subscribers {
		// Track member ID
		memberBoostyIDs = append(memberBoostyIDs, int64(subscriber.ID))

		// Process tier if present
		if subscriber.Level.ID > 0 && !subscriber.Level.Deleted {
			tierID := int64(subscriber.Level.ID)

			// Only process each tier once
			if !tierBoostyIDs[tierID] {
				tierBoostyIDs[tierID] = true

				// Marshal level data
				levelData, marshalErr := json.Marshal(subscriber.Level)
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal tier data for tier %d: %w", tierID, marshalErr)
				}

				// Upsert tier
				tierParams := db.UpsertBoostyTierParams{
					CredentialsID: credentialsID,
					BoostyID:      int64(subscriber.Level.ID),
					Name:          subscriber.Level.Name,
					Data:          string(levelData),
				}

				err = env.UpsertBoostyTier(ctx, tierParams)
				if err != nil {
					return fmt.Errorf("failed to upsert tier %d: %w", tierID, err)
				}
			}
		}

		// Determine member status based on available fields
		// Status is derived from Subscribed boolean as there's no Status field in the API response
		status := "inactive"
		if subscriber.Subscribed {
			status = "active"
		}
		if subscriber.IsBlackListed {
			status = "blacklisted"
		}

		// Marshal subscriber data
		subscriberData, marshalErr := json.Marshal(subscriber)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal subscriber data for %d: %w", subscriber.ID, marshalErr)
		}

		// Upsert member
		memberParams := db.UpsertBoostyMemberParams{
			CredentialsID: credentialsID,
			BoostyID:      int64(subscriber.ID),
			Email:         subscriber.Email,
			Status:        status,
			Data:          string(subscriberData),
		}

		err = env.UpsertBoostyMember(ctx, memberParams)
		if err != nil {
			return fmt.Errorf("failed to upsert member %d: %w", subscriber.ID, err)
		}
	}

	// Mark tiers not in current sync as missed
	tierIDsList := make([]int64, 0, len(tierBoostyIDs))
	for tierID := range tierBoostyIDs {
		tierIDsList = append(tierIDsList, tierID)
	}

	if len(tierIDsList) > 0 {
		markTiersParams := db.MarkBoostyTiersAsMissedParams{
			CredentialsID: credentialsID,
			BoostyIds:     tierIDsList,
		}
		err = env.MarkBoostyTiersAsMissed(ctx, markTiersParams)
		if err != nil {
			return fmt.Errorf("failed to mark missed tiers: %w", err)
		}
	}

	// Mark members not in current sync as missed
	if len(memberBoostyIDs) > 0 {
		err = env.MarkBoostyMembersAsMissed(ctx, memberBoostyIDs)
		if err != nil {
			return fmt.Errorf("failed to mark missed members: %w", err)
		}
	}

	return nil
}

func Resolve(ctx context.Context, env Env, credentialsID int64) error {
	return syncBoostyData(ctx, env, credentialsID)
}
