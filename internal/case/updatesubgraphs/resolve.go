package updatesubgraphs

import (
	"context"
	"fmt"

	"trip2g/internal/model"
)

type Env interface {
	InsertSubgraph(ctx context.Context, name string) error
	LatestNoteViews() *model.NoteViews
}

func Resolve(ctx context.Context, env Env) error {
	nvs := env.LatestNoteViews()

	for _, subgraph := range nvs.Subgraphs {
		err := env.InsertSubgraph(ctx, subgraph.Name)
		if err != nil {
			return fmt.Errorf("failed to insert subgraph %q: %w", subgraph.Name, err)
		}
	}

	return nil
}
