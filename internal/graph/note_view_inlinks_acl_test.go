package graph

// Regression test for the NoteView.inLinks ACL hole: the resolver resolved the
// inbound permalinks against the whole corpus (LatestNoteViews) and returned
// every hit unfiltered. NoteView exposes content and html, so a reader of note
// A got the full body of every note linking to A, including notes in subgraphs
// they cannot read. Every returned note must now pass CanReadNote — the same
// primitive as the note read path — and unreadable ones are dropped silently:
// this is a graph edge listing, not a search result.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// inLinksEnv implements just the Env surface InLinks touches. The embedded nil
// Env panics on any unexpected call.
type inLinksEnv struct {
	Env
	nvs     *appmodel.NoteViews
	canRead func(context.Context, *appmodel.NoteView) (bool, error)
}

func (e *inLinksEnv) LatestNoteViews() *appmodel.NoteViews { return e.nvs }

func (e *inLinksEnv) CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error) {
	return e.canRead(ctx, note)
}

// inLinksCtx carries env in the appreq the way r.env(ctx) expects to find it.
func inLinksCtx(env Env) context.Context {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")
	req := &appreq.Request{Req: fctx, Env: env}
	req.SetUserToken(&usertoken.Data{ID: 5, Role: "user"})
	req.StoreInContext()
	return fctx
}

// inLinksFixture: a target in subgraph "a" linked to from a readable note in
// "a" and from a secret note in "b".
func inLinksFixture() (*appmodel.NoteView, *appmodel.NoteViews) {
	target := aclNote(1, "a/target.md", "/a-target", "a")
	readable := aclNote(2, "a/two.md", "/a-two", "a")
	secret := aclNote(3, "b/secret.md", "/b-secret", "b")
	secret.Content = []byte("private body")

	target.InLinks = map[string]struct{}{
		readable.Permalink: {},
		secret.Permalink:   {},
	}

	return target, aclNvs(target, readable, secret)
}

func inLinksPermalinks(notes []appmodel.NoteView) []string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		out = append(out, n.Permalink)
	}
	return out
}

func TestNoteViewInLinks_DropsUnreadableNotes(t *testing.T) {
	target, nvs := inLinksFixture()
	env := &inLinksEnv{nvs: nvs, canRead: canReadSubgraphA}
	r := &noteViewResolver{&Resolver{DefaultEnv: env}}

	got, err := r.InLinks(inLinksCtx(env), target)

	require.NoError(t, err)
	require.Equal(t, []string{"/a-two"}, inLinksPermalinks(got),
		"a note in subgraph b must not leak through inLinks to a reader of subgraph a")
}

func TestNoteViewInLinks_ReadableNotesAreKept(t *testing.T) {
	target, nvs := inLinksFixture()
	env := &inLinksEnv{nvs: nvs, canRead: func(context.Context, *appmodel.NoteView) (bool, error) {
		return true, nil
	}}
	r := &noteViewResolver{&Resolver{DefaultEnv: env}}

	got, err := r.InLinks(inLinksCtx(env), target)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"/a-two", "/b-secret"}, inLinksPermalinks(got))
}

func TestNoteViewInLinks_ACLErrorFails(t *testing.T) {
	target, nvs := inLinksFixture()
	env := &inLinksEnv{nvs: nvs, canRead: func(context.Context, *appmodel.NoteView) (bool, error) {
		return true, errors.New("db down")
	}}
	r := &noteViewResolver{&Resolver{DefaultEnv: env}}

	_, err := r.InLinks(inLinksCtx(env), target)

	require.Error(t, err, "a failed ACL check must not fall through to returning the note")
}
