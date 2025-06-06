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
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	PrepareLiveNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

func normalizeInput(i *model.CreateReleaseInput) {
	i.Title = strings.TrimSpace(strings.ToLower(i.Title))
}

type Input = model.CreateReleaseInput
type Payload = model.CreateReleaseOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	normalizeInput(&input)

	noteViews := env.LatestNoteViews()

	homeID, payloadErr := getHomeID(noteViews, input)
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

func getHomeID(nvs *appmodel.NoteViews, input Input) (sql.NullInt64, Payload) {
	if input.HomeNoteVersionID != nil {
		for _, view := range nvs.List {
			if view.VersionID == *input.HomeNoteVersionID {
				return sql.NullInt64{Int64: view.VersionID, Valid: true}, nil
			}
		}

		return sql.NullInt64{}, &model.ErrorPayload{Message: "home note version ID does not exist in latest note views"}
	}

	return sql.NullInt64{}, nil
}
