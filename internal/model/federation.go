package model

//go:generate go tool github.com/mailru/easyjson/easyjson -all ./federation.go

import (
	"context"
	"encoding/json"
)

type Federation interface {
	Search(ctx context.Context, params FederationSearchParams) (FederationResult, error)
	Similar(ctx context.Context, params FederationSimilarParams) (FederationResult, error)
	NoteHTML(ctx context.Context, params FederationNoteHTMLParams) (FederationResult, error)
	FederatedSearch(ctx context.Context, params FederationSearchParams) (FederationResult, error)
	FederatedSimilar(ctx context.Context, params FederationSimilarParams) (FederationResult, error)
	FederatedNoteHTML(ctx context.Context, params FederationNoteHTMLParams) (FederationResult, error)
	Expand(ctx context.Context, params FederationExpandParams) (FederationResult, error)
	FederatedExpand(ctx context.Context, params FederationExpandParams) (FederationResult, error)
	GraphQLRequest(ctx context.Context, params FederationGraphQLParams) (FederationResult, error)
	Instructions(ctx context.Context) (FederationResult, error)
	FederatedInstructions(ctx context.Context, params FederationInstructionsParams) (FederationResult, error)
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

type FederationSearchParams struct {
	Query string   `json:"query"`
	KBID  string   `json:"kb_id,omitempty"`
	KBIDs []string `json:"kb_ids,omitempty"`
}

type FederationSimilarParams struct {
	KBID   string `json:"kb_id,omitempty"`
	PID    int64  `json:"pid,omitempty"`
	NoteID string `json:"note_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type FederationInstructionsParams struct {
	KBID string `json:"kb_id,omitempty"`
}

type FederationGraphQLParams struct {
	KBID      string         `json:"kb_id,omitempty"`
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type FederationNoteHTMLParams struct {
	KBID         string `json:"kb_id,omitempty"`
	PID          int64  `json:"pid,omitempty"`
	NoteID       string `json:"note_id,omitempty"`
	Path         string `json:"path,omitempty"`
	Href         string `json:"href,omitempty"`
	MatchID      string `json:"match_id,omitempty"`
	ContextWords int    `json:"context_words,omitempty"`
}

type FederationExpandParams struct {
	KBID    string   `json:"kb_id,omitempty"`
	PID     int64    `json:"pid,omitempty"`
	NoteID  string   `json:"note_id,omitempty"`
	Path    string   `json:"path,omitempty"`
	Href    string   `json:"href,omitempty"`
	TocPath []string `json:"toc_path,omitempty"`
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
