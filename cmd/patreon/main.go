package main

import (
	"fmt"
	"log"

	"trip2g/internal/appconfig"
	"trip2g/internal/patreon"
)

func logPrint(a ...any) {
	fmt.Print(a...) //nolint:forbidigo // CLI output
}

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

	campaignID := testCampaignID(client)
	testPatrons(client, campaignID)
	testWebhookWorkflow(client, campaignID)

	logPrintln("\n=== Patreon client test completed successfully! ===")
}

func testCampaignID(client *patreon.Client) string {
	logPrintln("=== Testing CampaignID ===")
	campaignID, err := client.CampaignID()
	if err != nil {
		log.Fatalf("failed to get campaign ID: %v", err)
	}
	logPrintf("Campaign ID: %s\n", campaignID)
	return campaignID
}

func testPatrons(client *patreon.Client, campaignID string) {
	logPrintln("\n=== Testing ListPatrons ===")
	patronsResp, err := client.ListPatrons(campaignID)
	if err != nil {
		log.Fatalf("failed to list patrons: %v", err)
	}

	logPrintf("Found %d patrons:\n", len(patronsResp.Data))
	for i, patron := range patronsResp.Data {
		logPrintf("  %d. ID: %s, Status: %s, Email: %s\n",
			i+1, patron.ID, patron.Attributes.PatronStatus,
			patron.Attributes.Email)
	}

	if len(patronsResp.Included) > 0 {
		logPrintf("Included entities: %d\n", len(patronsResp.Included))
		for i, entity := range patronsResp.Included {
			logPrintf("  %d. Type: %s, ID: %s\n", i+1, entity.Type, entity.ID)
		}
	}
}

func testWebhookWorkflow(client *patreon.Client, campaignID string) {
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
}

func listCurrentWebhooks(client *patreon.Client) {
	logPrintln("1. Listing current webhooks...")
	webhooks, err := client.ListWebhooks()
	if err != nil {
		log.Fatalf("failed to list webhooks: %v", err)
	}
	logPrintf("Current webhooks: %d\n", len(webhooks))
	for i, webhook := range webhooks {
		logPrintf("  %d. ID: %s, URI: %s\n", i+1, webhook.ID, webhook.Attributes.URI)
	}
}

func createTestWebhook(client *patreon.Client, campaignID string) string {
	logPrintln("\n2. Adding test webhook...")
	testWebhookURL := "https://example.com/test-webhook"
	triggers := []string{
		"members:create",
		"members:update",
		"members:delete",
	}

	err := client.CreateWebhook(campaignID, testWebhookURL, triggers)
	if err != nil {
		log.Printf("failed to create test webhook: %v", err)
		return ""
	}

	logPrintf("Test webhook created successfully\n")
	return testWebhookURL
}

func listWebhooksAfterCreation(client *patreon.Client) string {
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

func deleteTestWebhook(client *patreon.Client, testWebhookID string) {
	logPrintf("\n4. Deleting test webhook (ID: %s)...\n", testWebhookID)
	err := client.DeleteWebhook(testWebhookID)
	if err != nil {
		log.Printf("failed to delete test webhook: %v", err)
	} else {
		logPrintf("Test webhook deleted successfully\n")
	}
}

func listWebhooksAfterDeletion(client *patreon.Client) {
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
