package patreonjobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/patreon"
)

type Config struct {
	RefreshInterval time.Duration // How often to refresh data (default: 1 hour)
	ImmediatelyGap  time.Duration // How old synced_at must be to trigger immediate refresh (default: 10 minutes)
}

func DefaultConfig() Config {
	return Config{
		RefreshInterval: 1 * time.Hour,
		ImmediatelyGap:  10 * time.Minute,
	}
}

type Env interface {
	Logger() logger.Logger
	PublicURL() string

	UpdatePatreonCredentialsWebhookSecret(ctx context.Context, arg db.UpdatePatreonCredentialsWebhookSecretParams) error
	ClearPatreonCredentialsWebhookSecret(ctx context.Context, id int64) error

	PatreonClientByID(ctx context.Context, credentialsID int64) (patreon.Client, error)

	refreshpatreondata.Env
}

type PatreonJobs struct {
	env    Env
	config Config
	mu     sync.Mutex

	cancelMap map[int64]context.CancelFunc
	logger    logger.Logger
}

func New(ctx context.Context, env Env, config Config) (*PatreonJobs, error) {
	io := PatreonJobs{
		env:       env,
		config:    config,
		cancelMap: make(map[int64]context.CancelFunc),
		logger:    logger.WithPrefix(env.Logger(), "patreonjobs:"),
	}

	credentials, err := env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active Patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		err = io.RegisterWebhook(ctx, cred.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to register webhooks: %w", err)
		}

		// Check if credentials need immediate refresh based on config
		immediately := false
		if cred.SyncedAt == nil || time.Since(*cred.SyncedAt) > io.config.ImmediatelyGap {
			immediately = true
			var lastSync time.Time
			if cred.SyncedAt != nil {
				lastSync = *cred.SyncedAt
			}
			io.logger.Info("credentials need immediate refresh", "credentialsID", cred.ID, "lastSync", lastSync, "gap", io.config.ImmediatelyGap)
		}

		startErr := io.StartPatreonRefreshBackgroundJob(ctx, cred.ID, immediately)
		if startErr != nil {
			return nil, fmt.Errorf("failed to start Patreon refresh background job for credentials ID %d: %w", cred.ID, startErr)
		}
	}

	return &io, nil
}

func (io *PatreonJobs) Stop(ctx context.Context) {
	for id, cancel := range io.cancelMap {
		cancel()

		err := io.UnregisterWebhook(ctx, id)
		if err != nil {
			io.logger.Error("failed to unregister webhooks", "error", err)
		}
	}
}

func (io *PatreonJobs) StartPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64, immediately bool) error {
	io.logger.Info("starting Patreon refresh background job", "credentialsID", credentialsID)

	// Register webhook for this specific credential if PublicURL is configured
	if io.withWebhooks() {
		err := io.registerWebhookForCredentialID(ctx, credentialsID)
		if err != nil {
			io.logger.Error("failed to register webhook for credential", "credentialsID", credentialsID, "error", err)
			// Don't fail the job start if webhook registration fails
		}
	}

	ctx, cancel := context.WithCancel(ctx)

	io.mu.Lock()
	defer io.mu.Unlock()

	existingCancel, exists := io.cancelMap[credentialsID]
	if exists {
		existingCancel()
	}

	io.cancelMap[credentialsID] = cancel

	go func() {
		// Run immediately if requested
		if immediately {
			err := refreshpatreondata.Resolve(ctx, io.env, &credentialsID)
			if err != nil {
				io.logger.Error("failed to refresh Patreon data (immediate)", "credentialsID", credentialsID, "error", err)
			} else {
				io.logger.Info("successfully refreshed Patreon data (immediate)", "credentialsID", credentialsID)
			}
		}

		// Timer based on config
		ticker := time.NewTicker(io.config.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Call the refresh function
				err := refreshpatreondata.Resolve(ctx, io.env, &credentialsID)
				if err != nil {
					io.logger.Error("failed to refresh Patreon data", "credentialsID", credentialsID, "error", err)
				}
			case <-ctx.Done():
				io.logger.Info("Patreon refresh background job stopped", "credentialsID", credentialsID)
				return
			}
		}
	}()

	return nil
}

func (io *PatreonJobs) StopPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error {
	io.logger.Info("stopping Patreon refresh background job", "credentialsID", credentialsID)

	// Unregister webhook for this specific credential if PublicURL is configured
	if io.withWebhooks() {
		err := io.unregisterWebhookForCredentialID(ctx, credentialsID)
		if err != nil {
			io.logger.Error("failed to unregister webhook for credential", "credentialsID", credentialsID, "error", err)
			// Don't fail the job stop if webhook unregistration fails
		}
	}

	io.mu.Lock()
	defer io.mu.Unlock()

	cancelFunc, exists := io.cancelMap[credentialsID]
	if exists {
		cancelFunc()
		delete(io.cancelMap, credentialsID)
	}

	return nil
}

func (io *PatreonJobs) RegisterWebhook(ctx context.Context, credentialsID int64) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	client, err := io.env.PatreonClientByID(ctx, credentialsID)
	if err != nil {
		return fmt.Errorf("failed to get Patreon client: %w", err)
	}

	credentials, err := io.env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		if cred.WebhookSecret != nil && *cred.WebhookSecret != "" {
			io.logger.Info("webhook already registered for credentials", "credentialsID", cred.ID)
			continue
		}

		registerErr := io.registerWebhookForCredentials(ctx, credentialsID, publicURL, client)
		if registerErr != nil {
			return fmt.Errorf("failed to register webhook for credentials ID %d: %w", cred.ID, registerErr)
		}
	}

	return nil
}

func (io *PatreonJobs) registerWebhookForCredentials(ctx context.Context, credID int64, publicURL string, client patreon.Client) error {
	// Get campaigns for this credential
	campaigns, err := client.ListCampaigns()
	if err != nil {
		return fmt.Errorf("failed to list campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		io.logger.Warn("no campaigns found for credentials", "credentialsID", credID)
		return nil
	}

	// Use the first campaign (creators typically have one main campaign)
	campaign := campaigns[0]

	webhookURL := fmt.Sprintf("%s/api/patreon/webhook?credential_id=%d", publicURL, credID)
	triggers := []string{
		"members:create",
		"members:update",
		"members:delete",
	}

	webhook, err := client.CreateWebhook(campaign.ID, webhookURL, triggers)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	// Save the webhook secret to the database
	params := db.UpdatePatreonCredentialsWebhookSecretParams{
		WebhookSecret: &webhook.Attributes.Secret,
		ID:            credID,
	}

	err = io.env.UpdatePatreonCredentialsWebhookSecret(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to save webhook secret: %w", err)
	}

	io.logger.Info("webhook registered successfully",
		"credentialsID", credID,
		"campaignID", campaign.ID,
		"webhookURL", webhookURL,
	)

	return nil
}

func (io *PatreonJobs) UnregisterWebhook(ctx context.Context, credentialsID int64) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	client, err := io.env.PatreonClientByID(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to get Patreon client: %w", err)
	}

	// List all existing webhooks and delete only ones matching our public URL
	webhooks, err := client.ListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	expectedURLPrefix := fmt.Sprintf("%s/api/patreon/webhook", publicURL)

	for _, webhook := range webhooks {
		// Only delete webhooks that match our public URL
		if webhook.Attributes.URI == "" || len(webhook.Attributes.URI) < len(expectedURLPrefix) {
			continue
		}

		if webhook.Attributes.URI[:len(expectedURLPrefix)] == expectedURLPrefix {
			deleteErr := client.DeleteWebhook(webhook.ID)
			if deleteErr != nil {
				io.logger.Error("failed to delete webhook", "webhookID", webhook.ID, "error", deleteErr)
				// Continue with other webhooks even if one fails
			} else {
				io.logger.Info("webhook deleted successfully", "webhookID", webhook.ID, "uri", webhook.Attributes.URI)
			}
		}
	}

	// Clear all webhook secrets from the database
	credentials, err := io.env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		if cred.WebhookSecret != nil && *cred.WebhookSecret != "" {
			clearErr := io.env.ClearPatreonCredentialsWebhookSecret(ctx, cred.ID)
			if clearErr != nil {
				io.logger.Error("failed to clear webhook secret", "credentialsID", cred.ID, "error", clearErr)
			}
		}
	}

	return nil
}

func (io *PatreonJobs) withWebhooks() bool {
	return io.env.PublicURL() != ""
}

func (io *PatreonJobs) registerWebhookForCredentialID(ctx context.Context, credentialID int64) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	client, err := io.env.PatreonClientByID(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("failed to get Patreon client for credential ID %d: %w", credentialID, err)
	}

	// Check if webhook already exists via API (don't trust database)
	webhooks, err := client.ListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list existing webhooks: %w", err)
	}

	expectedURL := fmt.Sprintf("%s/api/patreon/webhook?credential_id=%d", publicURL, credentialID)
	for _, webhook := range webhooks {
		if webhook.Attributes.URI == expectedURL {
			io.logger.Info("webhook already exists for credential", "credentialID", credentialID, "webhookID", webhook.ID)

			params := db.UpdatePatreonCredentialsWebhookSecretParams{
				WebhookSecret: &webhook.Attributes.Secret,
				ID:            credentialID,
			}

			updateErr := io.env.UpdatePatreonCredentialsWebhookSecret(ctx, params)
			if updateErr != nil {
				io.logger.Error("failed to update webhook secret in database", "credentialID", credentialID, "error", updateErr)
			}

			return nil
		}
	}

	// Register the webhook using the existing logic
	return io.registerWebhookForCredentials(ctx, credentialID, publicURL, client)
}

func (io *PatreonJobs) unregisterWebhookForCredentialID(ctx context.Context, credentialID int64) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	client, err := io.env.PatreonClientByID(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("failed to get Patreon client for credential ID %d: %w", credentialID, err)
	}

	// List all webhooks and find the one matching this credential
	webhooks, err := client.ListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	expectedURL := fmt.Sprintf("%s/api/patreon/webhook?credential_id=%d", publicURL, credentialID)

	for _, webhook := range webhooks {
		if webhook.Attributes.URI == expectedURL {
			deleteErr := client.DeleteWebhook(webhook.ID)
			if deleteErr != nil {
				io.logger.Error("failed to delete webhook", "webhookID", webhook.ID, "error", deleteErr)
				// Continue to clear the secret even if deletion fails
			} else {
				io.logger.Info("webhook deleted successfully", "webhookID", webhook.ID, "credentialID", credentialID)
			}
			break
		}
	}

	// Clear the webhook secret from the database
	err = io.env.ClearPatreonCredentialsWebhookSecret(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("failed to clear webhook secret: %w", err)
	}

	return nil
}
