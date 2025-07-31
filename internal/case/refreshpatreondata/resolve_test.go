package refreshpatreondata

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/patreon"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

// mockPatreonClient is a mock implementation of patreon.Client.
type mockPatreonClient struct {
	listCampaignsFunc func() ([]patreon.Campaign, error)
	listPatronsFunc   func(campaignID string, nextPageURL ...string) (*patreon.PatronsResponse, error)
	listWebhooksFunc  func() ([]patreon.Webhook, error)
	createWebhookFunc func(campaignID string, webhookURL string, triggers []string) (*patreon.Webhook, error)
	deleteWebhookFunc func(webhookID string) error
}

func (m *mockPatreonClient) ListCampaigns() ([]patreon.Campaign, error) {
	if m.listCampaignsFunc != nil {
		return m.listCampaignsFunc()
	}
	return nil, nil
}

func (m *mockPatreonClient) ListPatrons(campaignID string, nextPageURL ...string) (*patreon.PatronsResponse, error) {
	if m.listPatronsFunc != nil {
		return m.listPatronsFunc(campaignID, nextPageURL...)
	}
	return nil, nil
}

func (m *mockPatreonClient) ListWebhooks() ([]patreon.Webhook, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc()
	}
	return nil, nil
}

func (m *mockPatreonClient) CreateWebhook(campaignID string, webhookURL string, triggers []string) (*patreon.Webhook, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(campaignID, webhookURL, triggers)
	}
	return nil, nil
}

func (m *mockPatreonClient) DeleteWebhook(webhookID string) error {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(webhookID)
	}
	return nil
}

func TestSyncTiers(t *testing.T) {
	// Load test campaign data
	campaignData, err := os.ReadFile("test_list_campaigns.json")
	require.NoError(t, err)

	var campaigns []patreon.Campaign
	err = json.Unmarshal(campaignData, &campaigns)
	require.NoError(t, err)
	require.Len(t, campaigns, 1)

	campaign := campaigns[0]

	// Process included data to ensure tiers have attributes
	resp := &patreon.CampaignsResponse{
		Data:     campaigns,
		Included: []patreon.IncludedEntity{}, // Empty for this test
	}
	resp.ProcessIncluded()

	ctx := context.Background()
	mockEnv := &EnvMock{}
	dbCampaignID := int64(1)

	// Track upserted tiers
	var upsertedTiers []db.UpsertPatreonTierParams
	mockEnv.UpsertPatreonTierFunc = func(ctx context.Context, arg db.UpsertPatreonTierParams) error {
		upsertedTiers = append(upsertedTiers, arg)
		return nil
	}

	// Test syncTiers
	err = syncTiers(ctx, mockEnv, campaign, dbCampaignID)
	require.NoError(t, err)

	// Verify tiers were upserted correctly
	require.Len(t, upsertedTiers, 2)

	// Check first tier (Free)
	freeTier := upsertedTiers[0]
	require.Equal(t, "17291845", freeTier.TierID)
	require.Equal(t, "Free", freeTier.Title)
	require.Equal(t, int64(0), freeTier.AmountCents)
	require.Contains(t, freeTier.Attributes, `"title":"Free"`)
	require.Contains(t, freeTier.Attributes, `"amount_cents":0`)

	// Check second tier (demo)
	demoTier := upsertedTiers[1]
	require.Equal(t, "26443756", demoTier.TierID)
	require.Equal(t, "demo", demoTier.Title)
	require.Equal(t, int64(100), demoTier.AmountCents)
	require.Contains(t, demoTier.Attributes, `"title":"demo"`)
	require.Contains(t, demoTier.Attributes, `"amount_cents":100`)
}

func TestSyncMembers(t *testing.T) {
	// Load test patrons data
	patronsData, err := os.ReadFile("test_list_patrons.json")
	require.NoError(t, err)

	var patronsResp patreon.PatronsResponse
	err = json.Unmarshal(patronsData, &patronsResp)
	require.NoError(t, err)

	ctx := context.Background()
	mockEnv := &EnvMock{}
	credentials := db.PatreonCredential{
		ID:                 1,
		CreatorAccessToken: "test-token",
	}
	campaignID := "7287952"
	dbCampaignID := int64(1)

	// Track upserted data
	var upsertedTiers []db.UpsertPatreonTierParams
	var upsertedMembers []db.UpsertPatreonMemberParams

	mockEnv.UpsertPatreonTierFunc = func(ctx context.Context, arg db.UpsertPatreonTierParams) error {
		upsertedTiers = append(upsertedTiers, arg)
		return nil
	}

	mockEnv.UpsertPatreonMemberFunc = func(ctx context.Context, arg db.UpsertPatreonMemberParams) error {
		upsertedMembers = append(upsertedMembers, arg)
		return nil
	}

	// Mock tier lookup for setting current tier
	mockEnv.GetPatreonTierByTierIDFunc = func(ctx context.Context, arg db.GetPatreonTierByTierIDParams) (db.PatreonTier, error) {
		return db.PatreonTier{
			ID:     1,
			TierID: "26443756",
		}, nil
	}

	// Mock member lookup for setting current tier
	mockEnv.GetPatreonMemberByPatreonIDAndCampaignIDFunc = func(ctx context.Context, arg db.GetPatreonMemberByPatreonIDAndCampaignIDParams) (db.PatreonMember, error) {
		return db.PatreonMember{
			ID:         1,
			PatreonID:  "80d9b143-c1ee-4370-bfd7-46997cd5bc7b",
			CampaignID: dbCampaignID,
		}, nil
	}

	mockEnv.SetPatreonMemberCurrentTierFunc = func(ctx context.Context, arg db.SetPatreonMemberCurrentTierParams) error {
		return nil
	}

	// Create a mock Patreon client
	mockPatreonClient := &mockPatreonClient{
		listPatronsFunc: func(campaignID string, nextPageURL ...string) (*patreon.PatronsResponse, error) {
			return &patronsResp, nil
		},
	}

	// Mock PatreonClientByID to return our mock client
	mockEnv.PatreonClientByIDFunc = func(ctx context.Context, credentialsID int64) (patreon.Client, error) {
		return mockPatreonClient, nil
	}

	// Test syncMembers
	err = syncMembers(ctx, mockEnv, credentials, campaignID, dbCampaignID)
	require.NoError(t, err)

	// Verify member was upserted
	require.Len(t, upsertedMembers, 1)
	member := upsertedMembers[0]
	require.Equal(t, "80d9b143-c1ee-4370-bfd7-46997cd5bc7b", member.PatreonID)
	require.Equal(t, dbCampaignID, member.CampaignID)
	require.Equal(t, "active_patron", member.Status)
	require.Equal(t, "sdevalex@gmail.com", member.Email)

	// Verify that tier from included data was NOT processed because it has empty attributes
	require.Empty(t, upsertedTiers, "Should not upsert tiers with empty attributes")
}

func TestSyncCampaigns(t *testing.T) {
	// Load test campaign data
	campaignData, err := os.ReadFile("test_list_campaigns.json")
	require.NoError(t, err)

	var campaigns []patreon.Campaign
	err = json.Unmarshal(campaignData, &campaigns)
	require.NoError(t, err)

	ctx := context.Background()
	mockEnv := &EnvMock{}
	credentials := db.PatreonCredential{
		ID:                 1,
		CreatorAccessToken: "test-token",
	}

	// Create a mock Patreon client
	mockPatreonClient := &mockPatreonClient{
		listCampaignsFunc: func() ([]patreon.Campaign, error) {
			return campaigns, nil
		},
	}

	// Mock PatreonClientByID to return our mock client
	mockEnv.PatreonClientByIDFunc = func(ctx context.Context, credentialsID int64) (patreon.Client, error) {
		return mockPatreonClient, nil
	}

	// Track upserted campaigns
	var upsertedCampaigns []db.UpsertPatreonCampaignParams
	mockEnv.UpsertPatreonCampaignFunc = func(ctx context.Context, arg db.UpsertPatreonCampaignParams) error {
		upsertedCampaigns = append(upsertedCampaigns, arg)
		return nil
	}

	// Test syncCampaigns
	resultCampaigns, err := syncCampaigns(ctx, mockEnv, credentials)
	require.NoError(t, err)
	require.Len(t, resultCampaigns, 1)

	// Verify campaign was upserted
	require.Len(t, upsertedCampaigns, 1)
	campaign := upsertedCampaigns[0]
	require.Equal(t, credentials.ID, campaign.CredentialsID)
	require.Equal(t, "7287952", campaign.CampaignID)
	require.Contains(t, campaign.Attributes, `"patron_count":1`)
}

func TestFullSyncIntegration(t *testing.T) {
	// This test demonstrates the issue where syncMembers overwrites good tier data
	// Load test data
	campaignData, err := os.ReadFile("test_list_campaigns.json")
	require.NoError(t, err)
	patronsData, err := os.ReadFile("test_list_patrons.json")
	require.NoError(t, err)

	var campaigns []patreon.Campaign
	err = json.Unmarshal(campaignData, &campaigns)
	require.NoError(t, err)

	var patronsResp patreon.PatronsResponse
	err = json.Unmarshal(patronsData, &patronsResp)
	require.NoError(t, err)

	ctx := context.Background()
	mockEnv := &EnvMock{}
	credentials := db.PatreonCredential{
		ID:                 1,
		CreatorAccessToken: "test-token",
	}

	// Track all upserted tiers to see the overwrite issue
	var allTierUpserts []db.UpsertPatreonTierParams

	// Create a mock Patreon client
	mockClient := &mockPatreonClient{
		listCampaignsFunc: func() ([]patreon.Campaign, error) {
			return campaigns, nil
		},
		listPatronsFunc: func(campaignID string, nextPageURL ...string) (*patreon.PatronsResponse, error) {
			return &patronsResp, nil
		},
	}

	// Mock PatreonClientByID to return our mock client
	mockEnv.PatreonClientByIDFunc = func(ctx context.Context, credentialsID int64) (patreon.Client, error) {
		return mockClient, nil
	}

	mockEnv.UpsertPatreonCampaignFunc = func(ctx context.Context, arg db.UpsertPatreonCampaignParams) error {
		return nil
	}

	mockEnv.UpsertPatreonTierFunc = func(ctx context.Context, arg db.UpsertPatreonTierParams) error {
		allTierUpserts = append(allTierUpserts, arg)
		return nil
	}

	mockEnv.UpsertPatreonMemberFunc = func(ctx context.Context, arg db.UpsertPatreonMemberParams) error {
		return nil
	}

	mockEnv.GetPatreonCampaignsByCredentialsIDFunc = func(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error) {
		return []db.PatreonCampaign{{
			ID:         1,
			CampaignID: "7287952",
		}}, nil
	}

	mockEnv.GetPatreonTierByTierIDFunc = func(ctx context.Context, arg db.GetPatreonTierByTierIDParams) (db.PatreonTier, error) {
		return db.PatreonTier{ID: 1}, nil
	}

	mockEnv.GetPatreonMemberByPatreonIDAndCampaignIDFunc = func(ctx context.Context, arg db.GetPatreonMemberByPatreonIDAndCampaignIDParams) (db.PatreonMember, error) {
		return db.PatreonMember{ID: 1}, nil
	}

	mockEnv.SetPatreonMemberCurrentTierFunc = func(ctx context.Context, arg db.SetPatreonMemberCurrentTierParams) error {
		return nil
	}

	mockEnv.UpdatePatreonCredentialsSyncedAtFunc = func(ctx context.Context, id int64) error {
		return nil
	}

	// Run full sync
	err = syncPatreonData(ctx, mockEnv, credentials)
	require.NoError(t, err)

	// Verify the fix: tiers should only be upserted once (from syncTiers with good data)
	// syncMembers should skip empty tier data
	require.GreaterOrEqual(t, len(allTierUpserts), 2, "Expected at least 2 tier upserts for different tiers")

	// Find tier 26443756 upserts
	var tier26443756Upserts []db.UpsertPatreonTierParams
	for _, upsert := range allTierUpserts {
		if upsert.TierID == "26443756" {
			tier26443756Upserts = append(tier26443756Upserts, upsert)
		}
	}

	require.Len(t, tier26443756Upserts, 1, "Expected tier 26443756 to be upserted only once (syncMembers should skip empty data)")

	// Upsert should have good data (from syncTiers only)
	upsert := tier26443756Upserts[0]
	require.Equal(t, "demo", upsert.Title)
	require.Equal(t, int64(100), upsert.AmountCents)

	t.Log("Fix verified: empty tier data from included section is now skipped")
}

func TestSyncMembersSkipsEmptyTiers(t *testing.T) {
	// This test verifies that syncMembers skips tiers with empty attributes
	patronsData, err := os.ReadFile("test_list_patrons.json")
	require.NoError(t, err)

	var patronsResp patreon.PatronsResponse
	err = json.Unmarshal(patronsData, &patronsResp)
	require.NoError(t, err)

	ctx := context.Background()
	mockEnv := &EnvMock{}
	credentials := db.PatreonCredential{
		ID:                 1,
		CreatorAccessToken: "test-token",
	}
	campaignID := "7287952"
	dbCampaignID := int64(1)

	// Track upserted tiers - should be empty because test data has empty tier attributes
	var upsertedTiers []db.UpsertPatreonTierParams

	// Create a mock Patreon client
	mockPatreonClientForTiers := &mockPatreonClient{
		listPatronsFunc: func(campaignID string, nextPageURL ...string) (*patreon.PatronsResponse, error) {
			return &patronsResp, nil
		},
	}

	// Mock PatreonClientByID to return our mock client
	mockEnv.PatreonClientByIDFunc = func(ctx context.Context, credentialsID int64) (patreon.Client, error) {
		return mockPatreonClientForTiers, nil
	}

	mockEnv.UpsertPatreonTierFunc = func(ctx context.Context, arg db.UpsertPatreonTierParams) error {
		upsertedTiers = append(upsertedTiers, arg)
		return nil
	}

	mockEnv.UpsertPatreonMemberFunc = func(ctx context.Context, arg db.UpsertPatreonMemberParams) error {
		return nil
	}

	mockEnv.GetPatreonTierByTierIDFunc = func(ctx context.Context, arg db.GetPatreonTierByTierIDParams) (db.PatreonTier, error) {
		return db.PatreonTier{ID: 1}, nil
	}

	mockEnv.GetPatreonMemberByPatreonIDAndCampaignIDFunc = func(ctx context.Context, arg db.GetPatreonMemberByPatreonIDAndCampaignIDParams) (db.PatreonMember, error) {
		return db.PatreonMember{ID: 1}, nil
	}

	mockEnv.SetPatreonMemberCurrentTierFunc = func(ctx context.Context, arg db.SetPatreonMemberCurrentTierParams) error {
		return nil
	}

	// Test syncMembers
	err = syncMembers(ctx, mockEnv, credentials, campaignID, dbCampaignID)
	require.NoError(t, err)

	// Verify that no tiers were upserted because the included tier has empty attributes
	require.Empty(t, upsertedTiers, "Should not upsert tiers with empty attributes")
}
