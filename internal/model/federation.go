package model

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

type FederationNoteHTMLParams struct {
	KBID         string `json:"kb_id,omitempty"`
	PID          int64  `json:"pid,omitempty"`
	NoteID       string `json:"note_id,omitempty"`
	Path         string `json:"path,omitempty"`
	Href         string `json:"href,omitempty"`
	MatchID      string `json:"match_id,omitempty"`
	ContextWords int    `json:"context_words,omitempty"`
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
