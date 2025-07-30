package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"trip2g/internal/appconfig"
	"trip2g/internal/patreon"
)

func logPrintf(format string, a ...any) {
	fmt.Printf(format, a...) //nolint:forbidigo // CLI output
}

func logPrintln(a ...any) {
	fmt.Println(a...) //nolint:forbidigo // CLI output
}

func main() {
	config, err := appconfig.Get()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if config.PatreonConfig.CreatorAccessToken == "" {
		log.Fatal("PATREON_CREATOR_ACCESS_TOKEN is required")
	}

	client, err := patreon.NewClient(config.PatreonConfig)
	if err != nil {
		log.Fatalf("failed to create patreon client: %v", err)
	}

	campaignID := testListCampaigns(client)
	testPatrons(client, campaignID)
	testWebhookWorkflow(client, campaignID)

	logPrintln("\n=== Patreon client test completed successfully! ===")
}

func testListCampaigns(client patreon.Client) string {
	logPrintln("=== Testing ListCampaigns ===")
	campaigns, err := client.ListCampaigns()
	if err != nil {
		log.Fatalf("failed to list campaigns: %v", err)
	}

	if len(campaigns) == 0 {
		log.Fatal("no campaigns found")
	}

	rawCampaigns, _ := json.Marshal(campaigns)

	if writeErr := os.WriteFile("./internal/case/refreshpatreondata/test_list_campaigns.json", rawCampaigns, 0600); writeErr != nil {
		log.Printf("Failed to write test campaigns file: %v", writeErr)
	}

	logPrintf("Found %d campaigns:\n", len(campaigns))
	for i, campaign := range campaigns {
		logPrintf("  %d. ID: %s, Type: %s\n", i+1, campaign.ID, campaign.Type)
		logPrintf("     Attributes: %s\n", string(campaign.Attributes))
		logPrintf("     Relationships: %+v\n", campaign.Relationships)

		// Test GetTiersWithAttributes method
		tiers, tiersErr := campaign.GetTiersWithAttributes()
		switch {
		case tiersErr != nil:
			logPrintf("     Error getting tiers: %v\n", tiersErr)
		case len(tiers) > 0:
			logPrintf("     Tiers (%d):\n", len(tiers))
			for j, tier := range tiers {
				logPrintf("       %d. ID: %v, Type: %v\n", j+1, tier["id"], tier["type"])
				if attributes, ok := tier["attributes"].(map[string]interface{}); ok {
					logPrintf("          Title: %v, Amount: %v cents\n",
						attributes["title"], attributes["amount_cents"])
				}
			}
		default:
			logPrintf("     No tiers found\n")
		}
	}

	// Return the first campaign ID for compatibility with existing tests
	return campaigns[0].ID
}

func testPatrons(client patreon.Client, campaignID string) {
	logPrintln("\n=== Testing ListPatrons ===")
	patronsResp, err := client.ListPatrons(campaignID)
	if err != nil {
		log.Fatalf("failed to list patrons: %v", err)
	}

	raw, _ := json.Marshal(patronsResp)

	if writeErr := os.WriteFile("./internal/case/refreshpatreondata/test_list_patrons.json", raw, 0600); writeErr != nil {
		log.Printf("Failed to write test patrons file: %v", writeErr)
	}

	logPrintf("Found %d patrons:\n", len(patronsResp.Data))
	for i, patron := range patronsResp.Data {
		logPrintf("  %d. ID: %s, Attributes: %+v\n",
			i+1, patron.ID, patron.Attributes)
	}

	if len(patronsResp.Included) > 0 {
		logPrintf("Included entities: %d\n", len(patronsResp.Included))
		for i, entity := range patronsResp.Included {
			logPrintf("  %d. Type: %s, ID: %s\n", i+1, entity.Type, entity.ID)
		}
	}
}

func testWebhookWorkflow(client patreon.Client, campaignID string) {
	logPrintln("\n=== Testing Webhook Workflow ===")

	listCurrentWebhooks(client)
	createTestWebhook(client, campaignID)
	actualTestWebhookID := listWebhooksAfterCreation(client)

	if actualTestWebhookID != "" {
		deleteTestWebhook(client, actualTestWebhookID)
		listWebhooksAfterDeletion(client)
	} else {
		logPrintln("\n4. Test webhook was not found, skipping deletion test")
	}

	// webhooks, err := client.ListWebhooks()
	// if err != nil {
	// 	panic(err)
	// }

	// for _, webhook := range webhooks {
	// 	fmt.Println("Remaining webhook ID:", webhook.ID, "URI:", webhook.Attributes.URI)
	// 	deleteTestWebhook(client, webhook.ID)
	// }
}

func listCurrentWebhooks(client patreon.Client) {
	logPrintln("1. Listing current webhooks...")
	webhooks, err := client.ListWebhooks()
	if err != nil {
		log.Fatalf("failed to list webhooks: %v", err)
	}
	logPrintf("Current webhooks: %d\n", len(webhooks))
	for i, webhook := range webhooks {
		logPrintf("  %d. ID: %s, URI: %s, Triggers: %+v\n", i+1, webhook.ID, webhook.Attributes.URI, webhook.Attributes.Triggers)
	}
}

func createTestWebhook(client patreon.Client, campaignID string) string {
	logPrintln("\n2. Adding test webhook...")
	testWebhookURL := "https://example.com/test-webhook"
	triggers := []string{
		"members:create",
		"members:update",
		"members:delete",
	}

	resp, err := client.CreateWebhook(campaignID, testWebhookURL, triggers)
	if err != nil {
		log.Printf("failed to create test webhook: %v", err)
		return ""
	}

	logPrintf("Test webhook created successfully: %+v\n", resp)
	return testWebhookURL
}

func listWebhooksAfterCreation(client patreon.Client) string {
	logPrintln("\n3. Listing webhooks after creation...")
	webhooksAfter, err := client.ListWebhooks()
	if err != nil {
		log.Fatalf("failed to list webhooks after creation: %v", err)
	}
	logPrintf("Webhooks after creation: %d\n", len(webhooksAfter))

	testWebhookURL := "https://example.com/test-webhook"
	var testWebhookID string
	for i, webhook := range webhooksAfter {
		logPrintf("  %d. ID: %s, URI: %s\n", i+1, webhook.ID, webhook.Attributes.URI)
		if webhook.Attributes.URI == testWebhookURL {
			testWebhookID = webhook.ID
		}
	}
	return testWebhookID
}

func deleteTestWebhook(client patreon.Client, testWebhookID string) {
	logPrintf("\n4. Deleting test webhook (ID: %s)...\n", testWebhookID)
	err := client.DeleteWebhook(testWebhookID)
	if err != nil {
		log.Printf("failed to delete test webhook: %v", err)
	} else {
		logPrintf("Test webhook deleted successfully\n")
	}
}

func listWebhooksAfterDeletion(client patreon.Client) {
	logPrintln("\n5. Listing webhooks after deletion...")
	webhooksFinal, err := client.ListWebhooks()
	if err != nil {
		log.Fatalf("failed to list webhooks after deletion: %v", err)
	}
	logPrintf("Final webhook count: %d\n", len(webhooksFinal))
	for i, webhook := range webhooksFinal {
		logPrintf("  %d. ID: %s, URI: %s\n", i+1, webhook.ID, webhook.Attributes.URI)
	}
}
