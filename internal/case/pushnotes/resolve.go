package pushnotes

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"

	appmodel "trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	InsertNote(ctx context.Context, update db.Note) error
	InsertSubgraph(ctx context.Context, name string) error
	PrepareNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

func Resolve(ctx context.Context, env Env, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	if len(input.Updates) == 0 {
		return &model.ErrorPayload{Message: "No updates provided"}, nil
	}

	for _, update := range input.Updates {
		note := db.Note{
			Path: update.Path,
			// TODO: remove it
			Content: update.Content, // + fmt.Sprintf("%d", time.Now().Unix()),
		}

		insertErr := env.InsertNote(ctx, note)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert note: %w", insertErr)
		}
	}

	nvs, err := env.PrepareNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	env.Logger().Info("insert subgraphs", "subgraphs", nvs.Subgraphs)

	for _, subgraph := range nvs.Subgraphs {
		insertErr := env.InsertSubgraph(ctx, subgraph.Name)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert subgraph: %w", insertErr)
		}
	}

	response := model.PushNotesPayload{
		Notes: nvs.List,
	}

	return &response, nil
}
