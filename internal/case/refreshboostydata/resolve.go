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
	GetBoostyTierIDByCredentialsAndBoostyID(ctx context.Context, arg db.GetBoostyTierIDByCredentialsAndBoostyIDParams) (int64, error)

	// Boosty member operations
	UpsertBoostyMember(ctx context.Context, arg db.UpsertBoostyMemberParams) error
	MarkBoostyMembersAsMissed(ctx context.Context, boostyIDs []int64) error

	// Sync tracking
	UpdateBoostyCredentialsSyncedAt(ctx context.Context, id int64) error

	Logger() logger.Logger
}

func processBoostyTier(ctx context.Context, env Env, credentialsID int64, subscriber boosty.Subscriber, tierBoostyIDs map[int64]bool) error {
	if subscriber.Level.ID <= 0 || subscriber.Level.Deleted {
		return nil
	}

	tierID := int64(subscriber.Level.ID)
	if tierBoostyIDs[tierID] {
		return nil // Already processed
	}

	tierBoostyIDs[tierID] = true

	levelData, err := json.Marshal(subscriber.Level)
	if err != nil {
		return fmt.Errorf("failed to marshal tier data for tier %d: %w", tierID, err)
	}

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

	return nil
}

func determineSubscriberStatus(subscriber boosty.Subscriber) string {
	status := "inactive"
	if subscriber.Subscribed {
		status = "active"
	}
	if subscriber.IsBlackListed {
		status = "blacklisted"
	}
	return status
}

func processBoostyMember(ctx context.Context, env Env, credentialsID int64, subscriber boosty.Subscriber) error {
	status := determineSubscriberStatus(subscriber)

	subscriberData, err := json.Marshal(subscriber)
	if err != nil {
		return fmt.Errorf("failed to marshal subscriber data for %d: %w", subscriber.ID, err)
	}

	memberParams := db.UpsertBoostyMemberParams{
		CredentialsID: credentialsID,
		BoostyID:      int64(subscriber.ID),
		Email:         subscriber.Email,
		Status:        status,
		Data:          string(subscriberData),
	}

	// Set current_tier_id if the subscriber has a valid tier
	if subscriber.Level.ID > 0 && !subscriber.Level.Deleted {
		tierParams := db.GetBoostyTierIDByCredentialsAndBoostyIDParams{
			CredentialsID: credentialsID,
			BoostyID:      int64(subscriber.Level.ID),
		}
		tierID, tierErr := env.GetBoostyTierIDByCredentialsAndBoostyID(ctx, tierParams)
		if tierErr != nil {
			// If tier not found, log but continue without setting current_tier_id
			env.Logger().Debug("tier not found for member", "member_id", subscriber.ID, "tier_boosty_id", subscriber.Level.ID, "error", tierErr)
		} else {
			memberParams.CurrentTierID = &tierID
		}
	}

	err = env.UpsertBoostyMember(ctx, memberParams)
	if err != nil {
		return fmt.Errorf("failed to upsert member %d: %w", subscriber.ID, err)
	}

	return nil
}

func markMissedEntities(ctx context.Context, env Env, credentialsID int64, tierBoostyIDs map[int64]bool, memberBoostyIDs []int64) error {
	// Mark tiers not in current sync as missed
	if len(tierBoostyIDs) > 0 {
		tierIDsList := make([]int64, 0, len(tierBoostyIDs))
		for tierID := range tierBoostyIDs {
			tierIDsList = append(tierIDsList, tierID)
		}

		markTiersParams := db.MarkBoostyTiersAsMissedParams{
			CredentialsID: credentialsID,
			BoostyIds:     tierIDsList,
		}
		err := env.MarkBoostyTiersAsMissed(ctx, markTiersParams)
		if err != nil {
			return fmt.Errorf("failed to mark missed tiers: %w", err)
		}
	}

	// Mark members not in current sync as missed
	if len(memberBoostyIDs) > 0 {
		err := env.MarkBoostyMembersAsMissed(ctx, memberBoostyIDs)
		if err != nil {
			return fmt.Errorf("failed to mark missed members: %w", err)
		}
	}

	return nil
}

func syncBoostyData(ctx context.Context, env Env, credentialsID int64) error {
	client, err := env.BoostyClientByCredentialsID(ctx, credentialsID)
	if err != nil {
		return fmt.Errorf("failed to get Boosty client: %w", err)
	}

	subscribers, err := client.Subscribers()
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	env.Logger().Debug("fetched subscribers from Boosty", "count", len(subscribers))

	tierBoostyIDs := make(map[int64]bool)
	memberBoostyIDs := make([]int64, 0, len(subscribers))

	for _, subscriber := range subscribers {
		memberBoostyIDs = append(memberBoostyIDs, int64(subscriber.ID))

		err = processBoostyTier(ctx, env, credentialsID, subscriber, tierBoostyIDs)
		if err != nil {
			return err
		}

		err = processBoostyMember(ctx, env, credentialsID, subscriber)
		if err != nil {
			return err
		}
	}

	return markMissedEntities(ctx, env, credentialsID, tierBoostyIDs, memberBoostyIDs)
}

func Resolve(ctx context.Context, env Env, credentialsID int64) error {
	err := syncBoostyData(ctx, env, credentialsID)
	if err != nil {
		return err
	}

	// Update synced_at timestamp after successful sync
	err = env.UpdateBoostyCredentialsSyncedAt(ctx, credentialsID)
	if err != nil {
		// Log the error but don't fail the entire operation
		env.Logger().Error("failed to update synced_at timestamp", "credentials_id", credentialsID, "error", err)
	}

	return nil
}
