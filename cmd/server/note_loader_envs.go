package main

import (
	"context"
	"fmt"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/noteloader"
)

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
		res[i] = noteloader.RawNote(note)
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
			VersionID: asset.VersionID,
			Path:      asset.Path,
			NoteAsset: asset.NoteAsset,
		}
	}

	return res, nil
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

func (e *latestNoteLoaderEnv) RawNotes(ctx context.Context) ([]noteloader.RawNote, error) {
	notes, err := e.env(ctx).AllLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	res := make([]noteloader.RawNote, len(notes))
	for i, note := range notes {
		res[i] = noteloader.RawNote(note)
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
			VersionID: asset.VersionID,
			Path:      asset.Path,
			NoteAsset: asset.NoteAsset,
		}

		if asset.NoteAsset.FileName != "hardreset_loop.png" {
			continue
		}
	}

	return res, nil
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
			VersionID: asset.VersionID,
			Path:      asset.Path,
			NoteAsset: asset.NoteAsset,
		}

		if asset.NoteAsset.FileName != "hardreset_loop.png" {
			continue
		}
	}

	return res, nil
}

func makeSingleNoteLoaderWrapper(a *app, versionID int64) *singleNoteLoaderEnv {
	return &singleNoteLoaderEnv{
		app:          a,
		versionID:    versionID,
		latestLoader: makeLatestNoteLoaderWrapper(a),
	}
}
