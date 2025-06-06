package makereleaselive

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	ReleaseByID(ctx context.Context, id int64) (db.Release, error)
	ChangeLiveRelease(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	PrepareLiveNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

type Input = model.MakeReleaseLiveInput
type Payload = model.MakeReleaseLiveOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	release, err := env.ReleaseByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	err = env.ChangeLiveRelease(ctx, release.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to change live release: %w", err)
	}

	_, err = env.PrepareLiveNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare live notes: %w", err)
	}

	payload := model.MakeReleaseLivePayload{
		Release: &release,
	}

	return &payload, nil
}
