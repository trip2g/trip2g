package model

import (
	"context"
	"errors"
	"io"
)

// ErrFederatedAssetTooLarge is a peer offering an attachment over this side's
// limit, or streaming one without saying how large it is.
var ErrFederatedAssetTooLarge = errors.New("federated asset exceeds the size limit")

// FederatedAssetPath is where a base serves its notes' attachments to a
// federation caller, relative to the base's MCP URL (a KB-note's url).
const FederatedAssetPath = "/federation/asset"

// FederatedAssetParams names an attachment the way a reader of the note sees
// it: the note's path, and the attachment's name exactly as note_html lists it
// under assets — {"skills/reel.md", "files/attach.zip"}. Describe asks for the
// attachment's NoteAssetInfo instead of its bytes.
type FederatedAssetParams struct {
	NotePath string `json:"note_path"`
	Name     string `json:"name"`
	Describe bool   `json:"describe,omitempty"`
}

// NoteAssetInfo is one attachment of a note as an MCP reader sees it. Name is
// the attachment's path as written in the note's markdown, and the only form
// federated_asset accepts.
type NoteAssetInfo struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Sha256 string `json:"sha256"`
}

// FederatedAsset is a peer's attachment on its way through. Body streams it and
// must be closed; Size is what the peer declared, and the body never exceeds it.
type FederatedAsset struct {
	Size int64
	Body io.ReadCloser
}

type FederationAssetFetcher interface {
	DescribeAsset(ctx context.Context, params FederatedAssetParams) (NoteAssetInfo, error)
	FetchAsset(ctx context.Context, params FederatedAssetParams) (*FederatedAsset, error)
}

type FederationAssetFetcherFactory interface {
	FederationAssetFetcher(ctx context.Context, kbID string) (FederationAssetFetcher, error)
}
