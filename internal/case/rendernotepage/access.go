package rendernotepage

import (
	"context"

	"trip2g/internal/appreq"
	"trip2g/internal/case/canreadnote"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// noteAccess answers "may this viewer read that note" for one page render.
//
// It runs the one access mechanism (canreadnote.Resolve) but pins the request's
// identity and remembers both the viewer's subgraph list and every verdict, so
// a page listing many backlinks costs one lookup instead of one per link.
// See docs/en/user/subgraphs.md, "Teaser subgraphs".
type noteAccess struct {
	env       Env
	ctx       context.Context //nolint:containedctx // pinned to the render this predicate serves
	userToken *usertoken.Data

	subgraphs       []string
	subgraphsLoaded bool
	verdicts        map[string]bool
}

func newNoteAccess(ctx context.Context, env Env, userToken *usertoken.Data) *noteAccess {
	return &noteAccess{
		env:       env,
		ctx:       ctx,
		userToken: userToken,
		verdicts:  map[string]bool{},
	}
}

// canRead is the templateviews.NoteAccess predicate. A lookup failure denies:
// a widget list is not the place to leak on error.
func (a *noteAccess) canRead(note *model.NoteView) bool {
	if note == nil {
		return false
	}
	if verdict, ok := a.verdicts[note.Path]; ok {
		return verdict
	}

	verdict, err := canreadnote.Resolve(a.ctx, a, note)
	if err != nil {
		a.env.Logger().Error("note access check failed", "path", note.Path, "error", err)
		verdict = false
	}

	a.verdicts[note.Path] = verdict
	return verdict
}

// CurrentUserToken, CurrentFederatedScope and ListActiveUserSubgraphs make this
// a canreadnote.Env bound to a single request.
func (a *noteAccess) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return a.userToken, nil
}

func (a *noteAccess) CurrentFederatedScope(ctx context.Context) ([]string, bool) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, false
	}
	return req.FederatedScope()
}

func (a *noteAccess) ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error) {
	if a.subgraphsLoaded {
		return a.subgraphs, nil
	}

	subgraphs, err := a.env.ListActiveUserSubgraphs(ctx, userID)
	if err != nil {
		return nil, err
	}

	a.subgraphs = subgraphs
	a.subgraphsLoaded = true
	return subgraphs, nil
}

var _ canreadnote.Env = (*noteAccess)(nil)
