package canreadnote

import (
	"context"

	"trip2g/internal/model"
)

type ResolveWithSubgraphsEnv interface{}

func ResolveWithSubgraphs(ctx context.Context, env ResolveWithSubgraphsEnv, note *model.NoteView, allowed []string) (bool, error) {
	_ = ctx
	_ = env

	if note.Free {
		return true, nil
	}

	if len(allowed) == 0 {
		return false, nil
	}

	if anySubgraphRequiresSignin(note) {
		return true, nil
	}

	if len(note.SubgraphNames) == 0 {
		return true, nil
	}

	for _, noteSubgraph := range note.SubgraphNames {
		for _, allowedSubgraph := range allowed {
			if noteSubgraph == allowedSubgraph {
				return true, nil
			}
		}
	}

	return false, nil
}
