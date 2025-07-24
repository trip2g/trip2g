package patreon

import "encoding/json"

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./types.go

type CampaignsResponse struct {
	Data []Campaign `json:"data"`
	Meta Meta       `json:"meta"`
}

type Campaign struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes json.RawMessage `json:"attributes"`
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

type PatronAttributes struct {
	Email        string `json:"email"`
	PatronStatus string `json:"patron_status"`
}

type Relationships struct {
	User                   RelationshipData `json:"user"`
	CurrentlyEntitledTiers RelationshipData `json:"currently_entitled_tiers"`
}

type RelationshipData struct {
	Data interface{} `json:"data"`
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
