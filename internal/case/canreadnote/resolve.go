package canreadnote

import (
	"context"
	"fmt"
	"strings"
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

	// System files (_footer.md, _layouts/) and HTML templates are internal —
	// never expose to non-admin users.
	if note.IsSystem() || strings.HasSuffix(note.Path, ".html") {
		return false, nil
	}

	if userToken != nil {
		userSubgraphs, err = env.ListActiveUserSubgraphs(ctx, int64(userToken.ID))
		if err != nil {
			return false, fmt.Errorf("failed to list user subgraphs: %w", err)
		}
	} else {
		return note.Free, nil
	}

	// Notes in require_signin subgraphs: any authenticated user can read.
	// The sign-in wall already blocked guests in rendernotepage.
	if anySubgraphRequiresSignin(note) {
		return true, nil
	}

	// if user has no active subscriptions, they can't see anything
	if len(userSubgraphs) == 0 {
		return false, nil
	}

	// notes without subgraphs (general knowledge) are accessible to all users with active subscriptions
	if len(note.SubgraphNames) == 0 {
		return true, nil
	}

	// check if user has access to any of the note's subgraphs
	for _, noteSubgraph := range note.SubgraphNames {
		for _, userSubgraph := range userSubgraphs {
			if noteSubgraph == userSubgraph {
				return true, nil
			}
		}
	}

	return false, nil
}

// anySubgraphRequiresSignin returns true if any of the note's subgraphs
// has RequireSignin=true (auth-only access, no subscription needed).
func anySubgraphRequiresSignin(note *model.NoteView) bool {
	for _, sg := range note.Subgraphs {
		if sg != nil && sg.RequireSignin {
			return true
		}
	}
	return false
}
