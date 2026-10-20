package canreadnote

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/model"
)

// Reader answers Resolve for one viewer over many notes.
type Reader func(note *model.NoteView) (bool, error)

// Class names what a Reader admits: two readers of the same Class read the
// same notes, so a result built from one serves the other.
type Class string

// NewReader looks the viewer up once, so checking a whole vault costs one
// subgraph lookup instead of one per note. Scoped tokens and federated
// identities go through Resolve, which reads no DB for them.
func NewReader(ctx context.Context, env Env) (Reader, Class, error) {
	allowed, federated := env.CurrentFederatedScope(ctx)
	if appreq.Scoped(ctx) || federated {
		class := "federated:" + classList(allowed)
		if appreq.Scoped(ctx) {
			class = "scoped:" + classList(appreq.WebhookReadPatterns(ctx))
		}
		return func(note *model.NoteView) (bool, error) {
			return Resolve(ctx, env, note)
		}, Class(class), nil
	}

	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current user token: %w", err)
	}

	if userToken.IsAdmin() {
		return func(*model.NoteView) (bool, error) {
			return true, nil
		}, "admin", nil
	}

	if userToken == nil {
		return func(note *model.NoteView) (bool, error) {
			return note.IsAnonymouslyReadable(), nil
		}, "guest", nil
	}

	userSubgraphs, err := env.ListActiveUserSubgraphs(ctx, int64(userToken.ID))
	if err != nil {
		return nil, "", fmt.Errorf("failed to list user subgraphs: %w", err)
	}

	return func(note *model.NoteView) (bool, error) {
		if strings.HasSuffix(note.Path, ".html") {
			return false, nil
		}
		if note.IsAnonymouslyReadable() {
			return true, nil
		}
		return subgraphsGrant(note, userSubgraphs), nil
	}, Class("user:" + classList(userSubgraphs)), nil
}

// classList is names as a set: sorted, without repeats.
func classList(names []string) string {
	sorted := slices.Compact(slices.Sorted(slices.Values(names)))
	return strings.Join(sorted, "\n")
}
