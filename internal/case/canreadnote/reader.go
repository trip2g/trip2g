package canreadnote

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/model"
)

// Reader answers Resolve for one viewer over many notes.
type Reader func(note *model.NoteView) (bool, error)

// NewReader looks the viewer up once, so checking a whole vault costs one
// subgraph lookup instead of one per note. Scoped tokens and federated
// identities go through Resolve, which reads no DB for them.
func NewReader(ctx context.Context, env Env) (Reader, error) {
	_, federated := env.CurrentFederatedScope(ctx)
	if appreq.Scoped(ctx) || federated {
		return func(note *model.NoteView) (bool, error) {
			return Resolve(ctx, env, note)
		}, nil
	}

	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	if userToken.IsAdmin() {
		return func(*model.NoteView) (bool, error) {
			return true, nil
		}, nil
	}

	if userToken == nil {
		return func(note *model.NoteView) (bool, error) {
			return note.IsAnonymouslyReadable(), nil
		}, nil
	}

	userSubgraphs, err := env.ListActiveUserSubgraphs(ctx, int64(userToken.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to list user subgraphs: %w", err)
	}

	return func(note *model.NoteView) (bool, error) {
		if strings.HasSuffix(note.Path, ".html") {
			return false, nil
		}
		if note.IsAnonymouslyReadable() {
			return true, nil
		}
		return subgraphsGrant(note, userSubgraphs), nil
	}, nil
}
