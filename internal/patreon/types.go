package patreon

import (
	"encoding/json"
	"fmt"
)

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./types.go

type CampaignsResponse struct {
	Data     []Campaign        `json:"data"`
	Included []IncludedEntity  `json:"included"`
	Meta     Meta              `json:"meta"`
}

type Campaign struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    json.RawMessage `json:"attributes"`
	Relationships Relationships   `json:"relationships"`
}

// GetTiers returns the tiers relationship data for this campaign
func (c *Campaign) GetTiers() ([]Data, error) {
	return c.Relationships.Tiers.ParseDataArray()
}

type Tier struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes TierAttributes `json:"attributes"`
}

type TierAttributes struct {
	Title        string `json:"title"`
	AmountCents  int    `json:"amount_cents"`
	Description  string `json:"description"`
	Published    bool   `json:"published"`
	PatronCount  int    `json:"patron_count"`
	CreatedAt    string `json:"created_at"`
}

type WebhookRequest struct {
	Data WebhookData `json:"data"`
}

type WebhookData struct {
	Type          string               `json:"type"`
	Attributes    WebhookAttributes    `json:"attributes"`
	Relationships WebhookRelationships `json:"relationships"`
}

type WebhookAttributes struct {
	Triggers []string `json:"triggers"`
	URI      string   `json:"uri"`
}

type WebhookRelationships struct {
	Campaign RelationshipData `json:"campaign"`
}

type WebhooksResponse struct {
	Data []Webhook `json:"data"`
	Meta Meta      `json:"meta"`
}

type Webhook struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes WebhookAttributes `json:"attributes"`
}

type Meta struct {
	Count int `json:"count"`
}

type PatronsResponse struct {
	Data     []Patron         `json:"data"`
	Included []IncludedEntity `json:"included"`
	Links    Links            `json:"links"`
}

type Patron struct {
	ID            string           `json:"id"`
	Type          string           `json:"type"`
	Attributes    PatronAttributes `json:"attributes"`
	Relationships Relationships    `json:"relationships"`
}

// GetUser returns the user relationship data
func (p *Patron) GetUser() (*Data, error) {
	return p.Relationships.User.ParseData()
}

// GetCurrentlyEntitledTiers returns the currently entitled tiers relationship data
func (p *Patron) GetCurrentlyEntitledTiers() ([]Data, error) {
	return p.Relationships.CurrentlyEntitledTiers.ParseDataArray()
}

type PatronAttributes struct {
	// Members may restrict the sharing of their email address.
	Email string `json:"email"`
	// One of active_patron, declined_patron, former_patron. A null value indicates the member has never pledged. Can be null.
	PatronStatus   *string `json:"patron_status"`
	NextChargeDate string  `json:"next_charge_date"`
	LastChargeDate string  `json:"last_charge_date"`
}

type Relationships struct {
	User                   RelationshipData `json:"user"`
	CurrentlyEntitledTiers RelationshipData `json:"currently_entitled_tiers"`
	Tiers                  RelationshipData `json:"tiers"`
}

type RelationshipData struct {
	Data interface{} `json:"data"`
}

type Data struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	Raw  interface{} `json:"-"` // Raw holds the original data for additional processing
}

// ParseData converts interface{} data to Data struct for single relationship
func (r *RelationshipData) ParseData() (*Data, error) {
	if r.Data == nil {
		return nil, nil
	}

	// Handle single object case
	if dataMap, ok := r.Data.(map[string]interface{}); ok {
		data := &Data{
			Raw: dataMap,
		}
		if id, exists := dataMap["id"]; exists {
			if idStr, ok := id.(string); ok {
				data.ID = idStr
			}
		}
		if typ, exists := dataMap["type"]; exists {
			if typStr, ok := typ.(string); ok {
				data.Type = typStr
			}
		}
		return data, nil
	}

	return nil, fmt.Errorf("unexpected data format for single relationship")
}

// ParseDataArray converts interface{} data to []Data for array relationships
func (r *RelationshipData) ParseDataArray() ([]Data, error) {
	if r.Data == nil {
		return nil, nil
	}

	// Handle array case
	if dataArray, ok := r.Data.([]interface{}); ok {
		var result []Data
		for _, item := range dataArray {
			if dataMap, ok := item.(map[string]interface{}); ok {
				data := Data{
					Raw: dataMap,
				}
				if id, exists := dataMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						data.ID = idStr
					}
				}
				if typ, exists := dataMap["type"]; exists {
					if typStr, ok := typ.(string); ok {
						data.Type = typStr
					}
				}
				result = append(result, data)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("unexpected data format for array relationship")
}

type IncludedEntity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

type Links struct {
	Next string `json:"next"`
}

type ErrorResponse struct {
	Errors []ErrorDetail `json:"errors"`
}

type ErrorDetail struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}
