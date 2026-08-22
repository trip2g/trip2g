package model

//go:generate go tool github.com/mailru/easyjson/easyjson -all ./federation.go

import (
	"context"
	"encoding/json"
	"time"
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
	RotateSecret(ctx context.Context, params MCPRotateSecretParams) (FederationResult, error)
	GrantedScope(ctx context.Context) (FederationScope, error)
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
	// PrevSecret is the key this pairing rotated away from, still accepted by
	// the peer inside its grace window. Empty outside that window, and outside
	// a rotation that has not been confirmed by a call yet. A request signs with
	// Secret and falls back to this one, which is what lets a rotation whose
	// response was lost heal on the next call instead of stranding the link.
	PrevSecret []byte
	Issuer     string
	Depth      int
}

// FederationAuthErrorCode is the JSON-RPC error a base returns when it cannot
// verify a federation JWT. It lives in the implementation-defined server range
// and exists so the caller can tell "this key is not the one you hold" from
// every other failure, which is what makes the fallback to PrevSecret precise
// rather than a blind retry. A peer too old to send it simply never triggers
// the fallback, and a peer too old to send it has no rotation either.
const FederationAuthErrorCode = -32001

// RotateSecretTool is dispatched on /_system/mcp but never appears in
// tools/list: that list is the contract third-party adapters mirror and the menu
// an LLM agent reads, and rotation is neither content nor something a model
// should reach for on its own initiative.
const RotateSecretTool = "rotate_secret"

// RotationGrace is how long a base keeps accepting the key a pairing rotated
// away from. It covers a response lost after the base committed and requests
// already in flight, and nothing longer: after the very first rotation the
// previous key is the one that travelled through a chat, so a window sized for
// an outage would keep alive exactly what rotation exists to kill. A successful
// call with the new key clears it sooner, which is the normal path.
const RotationGrace = 5 * time.Minute

// FederationScope is what a peer says this pairing may read there, as the
// pairing-description endpoint reports it.
type FederationScope struct {
	Version   int                   `json:"version"`
	KID       string                `json:"kid"`
	Subgraphs []FederationScopeItem `json:"subgraphs"`
	Rotation  bool                  `json:"rotation"`
}

type FederationScopeItem struct {
	Name             string `json:"name"`
	HumanDescription string `json:"human_description"`
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
