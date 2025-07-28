package patreon

import (
	"encoding/json"
	"errors"
)

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./types.go

type CampaignsResponse struct {
	Data     []Campaign       `json:"data"`
	Included []IncludedEntity `json:"included"`
	Meta     Meta             `json:"meta"`
}

// ProcessIncluded processes the included data and links it to campaigns.
func (r *CampaignsResponse) ProcessIncluded() {
	// Create a map of included entities by ID for quick lookup
	includedMap := make(map[string]IncludedEntity)
	for _, entity := range r.Included {
		includedMap[entity.ID] = entity
	}

	// Process each campaign to link included data
	for i := range r.Data {
		campaign := &r.Data[i]
		processCampaignTiers(campaign, includedMap)
	}
}

func processTierItem(tierItem interface{}, includedMap map[string]IncludedEntity) {
	tierMap, tierMapOk := tierItem.(map[string]interface{})
	if !tierMapOk {
		return
	}

	tierID, tierIDExists := tierMap["id"]
	if !tierIDExists {
		return
	}

	tierIDStr, tierIDStrOk := tierID.(string)
	if !tierIDStrOk {
		return
	}

	includedEntity, includedExists := includedMap[tierIDStr]
	if !includedExists || includedEntity.Type != "tier" {
		return
	}

	// Store the included entity attributes in the tier data
	tierMap["attributes"] = includedEntity.Attributes
}

func processCampaignTiers(campaign *Campaign, includedMap map[string]IncludedEntity) {
	tierDataArray, tierOk := campaign.Relationships.Tiers.Data.([]interface{})
	if !tierOk {
		return
	}

	for j, tierItem := range tierDataArray {
		processTierItem(tierItem, includedMap)
		// Update the array with the modified tier item
		tierDataArray[j] = tierItem
	}
}

type Campaign struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    json.RawMessage `json:"attributes"`
	Relationships Relationships   `json:"relationships"`
}

// GetTiers returns the tiers relationship data for this campaign.
func (c *Campaign) GetTiers() ([]Data, error) {
	return c.Relationships.Tiers.ParseDataArray()
}

// GetTiersWithAttributes returns tier data with full attributes.
func (c *Campaign) GetTiersWithAttributes() ([]map[string]interface{}, error) {
	if tierDataArray, arrayOk := c.Relationships.Tiers.Data.([]interface{}); arrayOk {
		var tiers []map[string]interface{}
		for _, tierItem := range tierDataArray {
			if tierMap, tierMapOk := tierItem.(map[string]interface{}); tierMapOk {
				tiers = append(tiers, tierMap)
			}
		}
		return tiers, nil
	}
	return nil, nil
}

type Tier struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes TierAttributes `json:"attributes"`
}

type TierAttributes struct {
	Title       string `json:"title"`
	AmountCents int    `json:"amount_cents"`
	Description string `json:"description"`
	Published   bool   `json:"published"`
	PatronCount int    `json:"patron_count"`
	CreatedAt   string `json:"created_at"`
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
	Triggers                  []string `json:"triggers"`
	URI                       string   `json:"uri"`
	Paused                    bool     `json:"paused,omitempty"`
	LastAttemptedAt           *string  `json:"last_attempted_at,omitempty"`
	NumConsecutiveTimesFailed int      `json:"num_consecutive_times_failed,omitempty"`
	Secret                    string   `json:"secret,omitempty"`
}

type WebhookRelationships struct {
	Campaign RelationshipData `json:"campaign"`
}

type WebhooksResponse struct {
	Data []Webhook `json:"data"`
	Meta Meta      `json:"meta"`
}

type WebhookCreateResponse struct {
	Data  Webhook `json:"data"`
	Links Links   `json:"links"`
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

// GetUser returns the user relationship data.
func (p *Patron) GetUser() (*Data, error) {
	return p.Relationships.User.ParseData()
}

// GetCurrentlyEntitledTiers returns the currently entitled tiers relationship data.
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

func parseDataMap(dataMap map[string]interface{}) *Data {
	data := &Data{
		Raw: dataMap,
	}

	if id, idExists := dataMap["id"]; idExists {
		if idStr, idStrOk := id.(string); idStrOk {
			data.ID = idStr
		}
	}

	if typ, typExists := dataMap["type"]; typExists {
		if typStr, typStrOk := typ.(string); typStrOk {
			data.Type = typStr
		}
	}

	return data
}

// ParseData converts interface{} data to Data struct for single relationship.
func (r *RelationshipData) ParseData() (*Data, error) {
	if r.Data == nil {
		return nil, nil
	}

	// Handle single object case
	if dataMap, mapOk := r.Data.(map[string]interface{}); mapOk {
		return parseDataMap(dataMap), nil
	}

	return nil, errors.New("unexpected data format for single relationship")
}

// ParseDataArray converts interface{} data to []Data for array relationships.
func (r *RelationshipData) ParseDataArray() ([]Data, error) {
	if r.Data == nil {
		return nil, nil
	}

	// Handle array case
	if dataArray, arrayOk := r.Data.([]interface{}); arrayOk {
		var result []Data
		for _, item := range dataArray {
			if dataMap, dataMapOk := item.(map[string]interface{}); dataMapOk {
				data := parseDataMap(dataMap)
				result = append(result, *data)
			}
		}
		return result, nil
	}

	return nil, errors.New("unexpected data format for array relationship")
}

type IncludedEntity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

type Links struct {
	Next string `json:"next,omitempty"`
	Self string `json:"self,omitempty"`
}

type ErrorResponse struct {
	Errors []ErrorDetail `json:"errors"`
}

type ErrorDetail struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}
