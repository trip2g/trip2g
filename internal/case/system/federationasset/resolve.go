// Package federationasset serves a note's attachment to a federation peer.
//
// A sibling of /_system/mcp rather than a tool on it: a tool answers in a
// JSON-RPC envelope, and an attachment is a stream of bytes of any size.
//
// The peer names the attachment the way its reader saw it — the note's path and
// the attachment's name as note_html lists it — and gets it only if it could
// read that note: the same view note_html reads, and the same subgraph grant.
// An attachment of a note outside the grant is refused exactly like a name the
// note does not have, so a refusal says nothing about what exists.
package federationasset

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

var (
	ErrAssetNotFound = errors.New("no such attachment in a note this pairing can read")
	ErrAssetTooLarge = errors.New("attachment exceeds the federated asset size limit")
)

type Env interface {
	mcp.FederationVerifyEnv
	LatestNoteViews() *model.NoteViews
	NoteAssetByID(ctx context.Context, id int64) (db.NoteAsset, error)
	StreamAssetObject(ctx context.Context, asset db.NoteAsset, offset, length int64) (io.ReadCloser, error)
	FederatedAssetMaxSize() int64
}

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg federationasset_test . Env

// Resolve finds the attachment params names, if caller may have it.
func Resolve(ctx context.Context, env Env, caller mcp.FederationCaller, params model.FederatedAssetParams) (db.NoteAsset, error) {
	note := env.LatestNoteViews().PathMap[params.NotePath]
	if note == nil {
		return db.NoteAsset{}, ErrAssetNotFound
	}

	canRead, err := canreadnote.ResolveWithSubgraphs(ctx, env, note, caller.AllowedSubgraphs)
	if err != nil {
		return db.NoteAsset{}, fmt.Errorf("check note access: %w", err)
	}
	if !canRead {
		return db.NoteAsset{}, ErrAssetNotFound
	}

	replace := note.AssetReplaces[params.Name]
	if replace == nil {
		return db.NoteAsset{}, ErrAssetNotFound
	}

	asset, err := env.NoteAssetByID(ctx, replace.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.NoteAsset{}, ErrAssetNotFound
	}
	if err != nil {
		return db.NoteAsset{}, fmt.Errorf("look up note asset: %w", err)
	}

	if asset.Size > env.FederatedAssetMaxSize() {
		return db.NoteAsset{}, ErrAssetTooLarge
	}

	return asset, nil
}
