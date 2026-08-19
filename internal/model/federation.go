package model

//go:generate go tool github.com/mailru/easyjson/easyjson -all ./federation.go

import (
	"context"
	"encoding/json"
)

type Federation interface {
	Search(ctx context.Context, params MCPSearchParams) (FederationResult, error)
	Similar(ctx context.Context, params MCPSimilarParams) (FederationResult, error)
	NoteHTML(ctx context.Context, params MCPNoteHTMLParams) (FederationResult, error)
	FederatedSearch(ctx context.Context, params MCPSearchParams) (FederationResult, error)
	FederatedSimilar(ctx context.Context, params MCPSimilarParams) (FederationResult, error)
	FederatedNoteHTML(ctx context.Context, params MCPNoteHTMLParams) (FederationResult, error)
	Expand(ctx context.Context, params MCPExpandParams) (FederationResult, error)
	FederatedExpand(ctx context.Context, params MCPExpandParams) (FederationResult, error)
	GraphQLRequest(ctx context.Context, params MCPGraphQLParams) (FederationResult, error)
	Instructions(ctx context.Context) (FederationResult, error)
	FederatedInstructions(ctx context.Context, params MCPInstructionsParams) (FederationResult, error)
}

type FederationClientFactory interface {
	FederationClient(ctx context.Context, kbID string) (Federation, error)
}

type FederationPeer struct {
	KBID   string
	KBURL  string
	KID    string
	Secret []byte
	Issuer string
	Depth  int
}

type FederationResult struct {
	Content           []FederationContent `json:"content"`
	StructuredContent json.RawMessage     `json:"structuredContent,omitempty"`
	IsError           bool                `json:"isError,omitempty"`
}

type FederationContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
