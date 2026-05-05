package main

import (
	"context"
	"fmt"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
)

func rawNoteChunksFromLatest(ctx context.Context, a *app) ([]noteloader.RawNoteChunk, error) {
	rows, err := a.GetAllLatestNoteChunksWithEmbeddings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest note chunks: %w", err)
	}
	res := make([]noteloader.RawNoteChunk, len(rows))
	for i, r := range rows {
		res[i] = noteloader.RawNoteChunk{
			VersionID:  r.VersionID,
			ChunkIndex: r.ChunkIndex,
			Content:    r.Content,
			Embedding:  r.Embedding,
			Path:       r.Path,
		}
	}
	return res, nil
}

func rawNoteChunksFromLive(ctx context.Context, a *app) ([]noteloader.RawNoteChunk, error) {
	rows, err := a.GetAllLiveNoteChunksWithEmbeddings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get live note chunks: %w", err)
	}
	res := make([]noteloader.RawNoteChunk, len(rows))
	for i, r := range rows {
		res[i] = noteloader.RawNoteChunk{
			VersionID:  r.VersionID,
			ChunkIndex: r.ChunkIndex,
			Content:    r.Content,
			Embedding:  r.Embedding,
			Path:       r.Path,
		}
	}
	return res, nil
}

type liveNoteLoaderEnv struct {
	*app
}

// cases can call Load inside the transaction, so we need to use the app from the context.
func (e *liveNoteLoaderEnv) env(ctx context.Context) *app {
	req, err := appreq.FromCtx(ctx)
	if err == nil {
		reqEnv, ok := req.Env.(*app)
		if ok {
			return reqEnv
		}
	}

	return e.app
}

func (e *liveNoteLoaderEnv) RawNotes(ctx context.Context) ([]noteloader.RawNote, error) {
	notes, err := e.env(ctx).AllLiveNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	res := make([]noteloader.RawNote, len(notes))
	for i, note := range notes {
		res[i] = noteloader.RawNote{
			Path:      note.Path,
			PathID:    note.PathID,
			VersionID: note.VersionID,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
			Embedding: note.Embedding,
		}
	}

	return res, nil
}

func (e *liveNoteLoaderEnv) RawAssets(ctx context.Context) ([]noteloader.RawAsset, error) {
	assets, err := e.env(ctx).AllLiveNoteAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get note assets: %w", err)
	}

	res := make([]noteloader.RawAsset, len(assets))
	for i, asset := range assets {
		res[i] = noteloader.RawAsset{
			VersionID:    asset.VersionID,
			Path:         asset.Path,
			NoteAsset:    asset.NoteAsset,
			AbsolutePath: asset.NoteAsset.AbsolutePath,
		}
	}

	return res, nil
}

func (e *liveNoteLoaderEnv) RawNoteChunks(ctx context.Context) ([]noteloader.RawNoteChunk, error) {
	return rawNoteChunksFromLive(ctx, e.env(ctx))
}

// ListAllSubgraphs proxies through env(ctx) to pick up the active write-transaction.
// Without this proxy the embedded *app.Queries hits the read-only pool, which cannot
// see uncommitted writes from the write connection. In practice: when a mutation calls
// PrepareLatestNotes / PrepareLiveNotes before its transaction commits, the read pool
// returns the pre-update value of require_signin and the noteloader enrichment silently
// stores stale flags — subgraph gating (e.g. sign-in wall) then has no effect until the
// next WAL checkpoint triggers a full reload.
func (e *liveNoteLoaderEnv) ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error) {
	return e.env(ctx).ListAllSubgraphs(ctx)
}

// just copy-paste the same code for latest notes loader
// because sqlc generates different structs for live and latest queries.
// please, fix it if you know better way to handle this.
func makeLiveNoteLoaderWrapper(a *app) *liveNoteLoaderEnv {
	return &liveNoteLoaderEnv{app: a}
}

type latestNoteLoaderEnv struct {
	*app
}

func (e *latestNoteLoaderEnv) env(ctx context.Context) *app {
	req, err := appreq.FromCtx(ctx)
	if err == nil {
		reqEnv, ok := req.Env.(*app)
		if ok {
			return reqEnv
		}
	}

	return e.app
}

// See liveNoteLoaderEnv.ListAllSubgraphs for rationale.
func (e *latestNoteLoaderEnv) ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error) {
	return e.env(ctx).ListAllSubgraphs(ctx)
}

func (e *latestNoteLoaderEnv) RawNotes(ctx context.Context) ([]noteloader.RawNote, error) {
	notes, err := e.env(ctx).AllLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	res := make([]noteloader.RawNote, len(notes))
	for i, note := range notes {
		res[i] = noteloader.RawNote{
			Path:      note.Path,
			PathID:    note.PathID,
			VersionID: note.VersionID,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
			Embedding: note.Embedding,
		}
	}

	return res, nil
}

func (e *latestNoteLoaderEnv) RawAssets(ctx context.Context) ([]noteloader.RawAsset, error) {
	assets, err := e.env(ctx).AllLatestNoteAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get note assets: %w", err)
	}

	res := make([]noteloader.RawAsset, len(assets))
	for i, asset := range assets {
		res[i] = noteloader.RawAsset{
			VersionID:    asset.VersionID,
			Path:         asset.Path,
			NoteAsset:    asset.NoteAsset,
			AbsolutePath: asset.NoteAsset.AbsolutePath,
		}
	}

	return res, nil
}

// LoadFrontmatterPatches loads patches using the context-aware env so that
// reads within an active GraphQL mutation transaction can see their own
// uncommitted inserts (SQLite read-your-own-writes within a transaction).
func (e *latestNoteLoaderEnv) LoadFrontmatterPatches(ctx context.Context) ([]frontmatterpatch.CompiledPatch, error) {
	return frontmatterpatch.NewLoader(e.env(ctx)).LoadFrontmatterPatches(ctx)
}

// LoadSiteConfig uses context-aware env for transaction visibility.
func (e *latestNoteLoaderEnv) LoadSiteConfig(ctx context.Context) (model.SiteConfig, error) {
	return e.env(ctx).SiteConfig(ctx), nil
}

func (e *latestNoteLoaderEnv) RawNoteChunks(ctx context.Context) ([]noteloader.RawNoteChunk, error) {
	return rawNoteChunksFromLatest(ctx, e.env(ctx))
}

func makeLatestNoteLoaderWrapper(a *app) *latestNoteLoaderEnv {
	return &latestNoteLoaderEnv{app: a}
}

type singleNoteLoaderEnv struct {
	*app
	versionID int64

	latestLoader *latestNoteLoaderEnv
}

func (e *singleNoteLoaderEnv) env(ctx context.Context) *app {
	req, err := appreq.FromCtx(ctx)
	if err == nil {
		reqEnv, ok := req.Env.(*app)
		if ok {
			return reqEnv
		}
	}

	return e.app
}

func (e *singleNoteLoaderEnv) RawNotes(ctx context.Context) ([]noteloader.RawNote, error) {
	note, err := e.env(ctx).NoteVersionByID(ctx, e.versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get note by version ID %d: %w", e.versionID, err)
	}

	// TODO: fix it. the layout can have dependency on multiple layout files. So we need to load all of them.
	if strings.HasPrefix(note.Path, "_layouts/") {
		return e.latestLoader.RawNotes(ctx)
	}

	return []noteloader.RawNote{
		{
			Path:      note.Path,
			PathID:    note.PathID,
			VersionID: note.VersionID,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
		},
	}, nil
}

func (e *singleNoteLoaderEnv) RawAssets(ctx context.Context) ([]noteloader.RawAsset, error) {
	assets, err := e.env(ctx).NoteAssetsByVersionID(ctx, e.versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get note assets by version ID %d: %w", e.versionID, err)
	}

	res := make([]noteloader.RawAsset, len(assets))
	for i, asset := range assets {
		res[i] = noteloader.RawAsset{
			VersionID:    asset.VersionID,
			Path:         asset.Path,
			NoteAsset:    asset.NoteAsset,
			AbsolutePath: asset.NoteAsset.AbsolutePath,
		}
	}

	return res, nil
}

func (e *singleNoteLoaderEnv) RawNoteChunks(_ context.Context) ([]noteloader.RawNoteChunk, error) {
	return nil, nil // single-note loader is used for preview rendering, not vector search
}

func makeSingleNoteLoaderWrapper(a *app, versionID int64) *singleNoteLoaderEnv {
	return &singleNoteLoaderEnv{
		app:          a,
		versionID:    versionID,
		latestLoader: makeLatestNoteLoaderWrapper(a),
	}
}
