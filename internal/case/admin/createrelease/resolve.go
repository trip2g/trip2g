package createrelease

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertRelease(ctx context.Context, arg db.InsertReleaseParams) (db.Release, error)
	InsertReleaseNoteVersion(ctx context.Context, arg db.InsertReleaseNoteVersionParams) error
	ChangeLiveRelease(ctx context.Context, id int64) error
	LatestNoteViews() *appmodel.NoteViews
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	PrepareLiveNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

func normalizeInput(i *model.CreateReleaseInput) {
	i.Title = strings.TrimSpace(strings.ToLower(i.Title))
}

func Resolve(ctx context.Context, env Env, input model.CreateReleaseInput) (model.CreateReleaseOrErrorPayload, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	normalizeInput(&input)

	noteViews := env.LatestNoteViews()

	homeID, payloadErr := getHomeID(env, noteViews, input)
	if payloadErr != nil {
		return payloadErr, nil
	}

	releaseParams := db.InsertReleaseParams{
		Title:             input.Title,
		CreatedBy:         int64(token.ID),
		HomeNoteVersionID: homeID,
	}

	release, err := env.InsertRelease(ctx, releaseParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert release: %w", err)
	}

	err = env.ChangeLiveRelease(ctx, release.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to change live release: %w", err)
	}

	for _, view := range noteViews.List {
		rnvParams := db.InsertReleaseNoteVersionParams{
			NoteVersionID: view.VersionID,
			ReleaseID:     release.ID,
		}

		err := env.InsertReleaseNoteVersion(ctx, rnvParams)
		if err != nil {
			return nil, fmt.Errorf("failed to insert release note version: %w", err)
		}
	}

	_, err = env.PrepareLiveNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare live notes: %w", err)
	}

	payload := model.CreateReleasePayload{
		Release: &release,
	}

	return &payload, nil
}

func getHomeID(env Env, nvs *appmodel.NoteViews, input model.CreateReleaseInput) (sql.NullInt64, model.CreateReleaseOrErrorPayload) {
	if input.HomeNoteVersionID != nil {
		for _, view := range nvs.List {
			if view.VersionID == int64(*input.HomeNoteVersionID) {
				return sql.NullInt64{Int64: view.VersionID, Valid: true}, nil
			}
		}

		return sql.NullInt64{}, &model.ErrorPayload{Message: "home note version ID does not exist in latest note views"}
	}

	return sql.NullInt64{}, nil
}
