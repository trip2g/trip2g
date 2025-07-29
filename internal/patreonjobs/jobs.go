package patreonjobs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/patreon"
)

type Env interface {
	Logger() logger.Logger
	PublicURL() string

	UpdatePatreonCredentialsWebhookSecret(ctx context.Context, arg db.UpdatePatreonCredentialsWebhookSecretParams) error
	ClearPatreonCredentialsWebhookSecret(ctx context.Context, id int64) error

	PatreonListCampaigns(token string) ([]patreon.Campaign, error)
	PatreonCreateWebhook(campaignID string, webhookURL string, triggers []string) (*patreon.Webhook, error)
	PatreonListWebhooks() ([]patreon.Webhook, error)
	PatreonDeleteWebhook(webhookID string) error

	refreshpatreondata.Env
}

type PatreonJobs struct {
	env Env
	mu  sync.Mutex

	cancelMap map[int64]context.CancelFunc
}

func New(ctx context.Context, env Env) (*PatreonJobs, error) {
	io := PatreonJobs{
		env:       env,
		cancelMap: make(map[int64]context.CancelFunc),
	}

	// Register webhooks if public URL is configured
	err := io.RegisterWebhook(ctx)
	if err != nil {
		env.Logger().Error("failed to register webhooks", "error", err)
		// Don't fail the initialization if webhook registration fails
	}

	credentials, err := env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active Patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		startErr := io.StartPatreonRefreshBackgroundJob(ctx, cred.ID)
		if startErr != nil {
			return nil, fmt.Errorf("failed to start Patreon refresh background job for credentials ID %d: %w", cred.ID, startErr)
		}
	}

	return &io, nil
}

func (io *PatreonJobs) Stop() {
	// Unregister webhooks
	ctx := context.Background()
	err := io.UnregisterWebhook(ctx)
	if err != nil {
		io.env.Logger().Error("failed to unregister webhooks", "error", err)
	}

	// Stop all background jobs
	for _, cancel := range io.cancelMap {
		cancel()
	}
}

func (io *PatreonJobs) StartPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error {
	io.env.Logger().Info("starting Patreon refresh background job", "credentialsID", credentialsID)

	// Register webhook for this specific credential if PublicURL is configured
	if io.withWebhooks() {
		err := io.registerWebhookForCredentialID(ctx, credentialsID)
		if err != nil {
			io.env.Logger().Error("failed to register webhook for credential", "credentialsID", credentialsID, "error", err)
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
		// 1 hour timer
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Call the refresh function
				err := refreshpatreondata.Resolve(ctx, io.env, &credentialsID)
				if err != nil {
					io.env.Logger().Error("failed to refresh Patreon data", "credentialsID", credentialsID, "error", err)
				}
			case <-ctx.Done():
				io.env.Logger().Info("Patreon refresh background job stopped", "credentialsID", credentialsID)
				return
			}
		}
	}()

	return nil
}

func (io *PatreonJobs) StopPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error {
	io.env.Logger().Info("stopping Patreon refresh background job", "credentialsID", credentialsID)

	// Unregister webhook for this specific credential if PublicURL is configured
	if io.withWebhooks() {
		err := io.unregisterWebhookForCredentialID(ctx, credentialsID)
		if err != nil {
			io.env.Logger().Error("failed to unregister webhook for credential", "credentialsID", credentialsID, "error", err)
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

func (io *PatreonJobs) RegisterWebhook(ctx context.Context) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	credentials, err := io.env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		if cred.WebhookSecret.Valid && cred.WebhookSecret.String != "" {
			io.env.Logger().Info("webhook already registered for credentials", "credentialsID", cred.ID)
			continue
		}

		registerErr := io.registerWebhookForCredentials(ctx, cred, publicURL)
		if registerErr != nil {
			io.env.Logger().Error("failed to register webhook for credentials", "credentialsID", cred.ID, "error", registerErr)
			// Continue with other credentials even if one fails
		}
	}

	return nil
}

func (io *PatreonJobs) registerWebhookForCredentials(ctx context.Context, cred db.PatreonCredential, publicURL string) error {
	// Get campaigns for this credential
	campaigns, err := io.env.PatreonListCampaigns(cred.CreatorAccessToken)
	if err != nil {
		return fmt.Errorf("failed to list campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		io.env.Logger().Warn("no campaigns found for credentials", "credentialsID", cred.ID)
		return nil
	}

	// Use the first campaign (creators typically have one main campaign)
	campaign := campaigns[0]

	webhookURL := fmt.Sprintf("%s/api/patreon/webhook?credential_id=%d", publicURL, cred.ID)
	triggers := []string{
		"members:create",
		"members:update",
		"members:delete",
	}

	webhook, err := io.env.PatreonCreateWebhook(campaign.ID, webhookURL, triggers)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	// Save the webhook secret to the database
	params := db.UpdatePatreonCredentialsWebhookSecretParams{
		WebhookSecret: sql.NullString{String: webhook.Attributes.Secret, Valid: true},
		ID:            cred.ID,
	}

	err = io.env.UpdatePatreonCredentialsWebhookSecret(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to save webhook secret: %w", err)
	}

	io.env.Logger().Info("webhook registered successfully",
		"credentialsID", cred.ID,
		"campaignID", campaign.ID,
		"webhookURL", webhookURL,
	)

	return nil
}

func (io *PatreonJobs) UnregisterWebhook(ctx context.Context) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	// List all existing webhooks and delete only ones matching our public URL
	webhooks, err := io.env.PatreonListWebhooks()
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
			deleteErr := io.env.PatreonDeleteWebhook(webhook.ID)
			if deleteErr != nil {
				io.env.Logger().Error("failed to delete webhook", "webhookID", webhook.ID, "error", deleteErr)
				// Continue with other webhooks even if one fails
			} else {
				io.env.Logger().Info("webhook deleted successfully", "webhookID", webhook.ID, "uri", webhook.Attributes.URI)
			}
		}
	}

	// Clear all webhook secrets from the database
	credentials, err := io.env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		if cred.WebhookSecret.Valid && cred.WebhookSecret.String != "" {
			clearErr := io.env.ClearPatreonCredentialsWebhookSecret(ctx, cred.ID)
			if clearErr != nil {
				io.env.Logger().Error("failed to clear webhook secret", "credentialsID", cred.ID, "error", clearErr)
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

	// Get the credential to access the token
	credential, err := io.env.PatreonCredentials(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("failed to get patreon credentials: %w", err)
	}

	// Check if webhook already exists via API (don't trust database)
	webhooks, err := io.env.PatreonListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list existing webhooks: %w", err)
	}

	expectedURL := fmt.Sprintf("%s/api/patreon/webhook?credential_id=%d", publicURL, credentialID)
	for _, webhook := range webhooks {
		if webhook.Attributes.URI == expectedURL {
			io.env.Logger().Info("webhook already exists for credential", "credentialID", credentialID, "webhookID", webhook.ID)

			// Update database with the webhook secret if it's missing
			if !credential.WebhookSecret.Valid || credential.WebhookSecret.String == "" {
				params := db.UpdatePatreonCredentialsWebhookSecretParams{
					WebhookSecret: sql.NullString{String: webhook.Attributes.Secret, Valid: true},
					ID:            credentialID,
				}
				updateErr := io.env.UpdatePatreonCredentialsWebhookSecret(ctx, params)
				if updateErr != nil {
					io.env.Logger().Error("failed to update webhook secret in database", "credentialID", credentialID, "error", updateErr)
				}
			}
			return nil
		}
	}

	// Register the webhook using the existing logic
	return io.registerWebhookForCredentials(ctx, credential, publicURL)
}

func (io *PatreonJobs) unregisterWebhookForCredentialID(ctx context.Context, credentialID int64) error {
	if !io.withWebhooks() {
		return nil
	}

	publicURL := io.env.PublicURL()

	// List all webhooks and find the one matching this credential
	webhooks, err := io.env.PatreonListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	expectedURL := fmt.Sprintf("%s/api/patreon/webhook?credential_id=%d", publicURL, credentialID)

	for _, webhook := range webhooks {
		if webhook.Attributes.URI == expectedURL {
			deleteErr := io.env.PatreonDeleteWebhook(webhook.ID)
			if deleteErr != nil {
				io.env.Logger().Error("failed to delete webhook", "webhookID", webhook.ID, "error", deleteErr)
				// Continue to clear the secret even if deletion fails
			} else {
				io.env.Logger().Info("webhook deleted successfully", "webhookID", webhook.ID, "credentialID", credentialID)
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
