package canreadnote

import (
	"context"
	"fmt"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
}

// Resolve determines if the current user has access to read the given note.
// Yes, it's not optimized for performance, but it's simple and works well enough for now.
func Resolve(ctx context.Context, env Env, note *model.NoteView) (bool, error) {
	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user token: %w", err)
	}

	var userSubgraphs []string

	if userToken != nil && userToken.IsAdmin() {
		return true, nil
	}

	if userToken != nil {
		userSubgraphs, err = env.ListActiveUserSubgraphs(ctx, int64(userToken.ID))
		if err != nil {
			return false, fmt.Errorf("failed to list user subgraphs: %w", err)
		}
	} else {
		return note.Free, nil
	}

	// non-subgraph notes are opened if the user have a subscription to
	// a graph with show_unsubgraph_notes_for_paid_users
	hasAccess := len(note.Subgraphs) == 0
	if hasAccess {
		// TODO: check show_unsubgraph_notes_for_paid_users
		// it's work for claude.
		hasAccess = true
	}

	// check if the user has access to the subgraph
	if !hasAccess {
		for _, ps := range note.SubgraphNames {
			for _, us := range userSubgraphs {
				if ps == us {
					hasAccess = true
					break
				}
			}
		}
	}

	return hasAccess, nil
}
