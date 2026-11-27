package graph

import (
	"context"
	"html/template"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/canreadnote"
	"trip2g/internal/embedguard"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

const embeddedSecret = "zzsecretzz"

type readableHTMLEnv struct {
	Env
	nvs  *appmodel.NoteViews
	user *usertoken.Data
}

func (e *readableHTMLEnv) LatestNoteViews() *appmodel.NoteViews { return e.nvs }

func (e *readableHTMLEnv) CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error) {
	return canreadnote.Resolve(ctx, e, note)
}

func (e *readableHTMLEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return e.user, nil
}

func (e *readableHTMLEnv) ListActiveUserSubgraphs(context.Context, int64) ([]string, error) {
	return []string{"paid"}, nil
}

func (e *readableHTMLEnv) CurrentFederatedScope(context.Context) ([]string, bool) {
	return nil, false
}

func readableHTMLVault() (*appmodel.NoteView, *appmodel.NoteView, *appmodel.NoteViews) {
	closed := aclNote(2, "closed.md", "/closed", "paid")
	closed.Title = "Closed"
	closed.HTML = template.HTML("<p>" + embeddedSecret + "</p>")

	parent := aclNote(1, "docs/parent.md", "/docs-parent")
	parent.Free = true
	parent.HTML = template.HTML("<p>intro</p>" + embedguard.Wrap(2, string(closed.HTML)) + "<p>outro</p>")

	return parent, closed, aclNvs(parent, closed)
}

func requestCtx(env Env, scoped []string) context.Context {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")

	req := &appreq.Request{Req: fctx, Env: env}
	if scoped != nil {
		req.WebhookScoped = true
		req.WebhookReadPatterns = scoped
	}

	req.StoreInContext()

	return fctx
}

func TestPublicNoteHTML_AnonymousDoesNotGetAClosedEmbed(t *testing.T) {
	parent, _, nvs := readableHTMLVault()
	env := &readableHTMLEnv{nvs: nvs}
	r := &publicNoteResolver{&Resolver{DefaultEnv: env}}

	got, err := r.HTML(requestCtx(env, nil), model.ConvertNoteToPublic(parent))

	require.NoError(t, err)
	require.Contains(t, got, "intro")
	require.Contains(t, got, "outro")
	require.NotContains(t, got, embeddedSecret)
	require.Contains(t, got, `<a href="/closed">Closed</a>`)
	require.NotContains(t, got, "<!--t2e:")
}

func TestPublicNoteHTML_SubscriberStillGetsTheEmbedInline(t *testing.T) {
	parent, _, nvs := readableHTMLVault()
	env := &readableHTMLEnv{nvs: nvs, user: &usertoken.Data{ID: 5, Role: "user"}}
	r := &publicNoteResolver{&Resolver{DefaultEnv: env}}

	got, err := r.HTML(requestCtx(env, nil), model.ConvertNoteToPublic(parent))

	require.NoError(t, err)
	require.Contains(t, got, embeddedSecret)
	require.NotContains(t, got, "<!--t2e:")
}

func TestPublicNoteHTML_PathScopedTokenDoesNotGetAnEmbedOutsideItsPattern(t *testing.T) {
	parent, closed, nvs := readableHTMLVault()
	closed.Free = true
	env := &readableHTMLEnv{nvs: nvs}
	r := &publicNoteResolver{&Resolver{DefaultEnv: env}}

	got, err := r.HTML(requestCtx(env, []string{"docs/*.md"}), model.ConvertNoteToPublic(parent))

	require.NoError(t, err)
	require.Contains(t, got, "intro")
	require.NotContains(t, got, embeddedSecret, "a free note outside the token's read patterns is still outside them")
}

func TestSearchDocumentHTML_StripsAClosedEmbed(t *testing.T) {
	parent, _, nvs := readableHTMLVault()
	env := &readableHTMLEnv{nvs: nvs}
	ctx := requestCtx(env, nil)

	doc, err := (&searchResultResolver{&Resolver{DefaultEnv: env}}).Document(ctx, &appmodel.SearchResult{NoteView: parent})
	require.NoError(t, err)

	note, ok := doc.(*model.PublicNote)
	require.True(t, ok)

	got, err := (&publicNoteResolver{&Resolver{DefaultEnv: env}}).HTML(ctx, note)
	require.NoError(t, err)
	require.NotContains(t, got, embeddedSecret)
}

func TestNoteViewHTML_AnonymousDoesNotGetAClosedEmbed(t *testing.T) {
	parent, _, nvs := readableHTMLVault()
	env := &readableHTMLEnv{nvs: nvs}
	r := &noteViewResolver{&Resolver{DefaultEnv: env}}

	got, err := r.HTML(requestCtx(env, nil), parent)

	require.NoError(t, err)
	require.NotContains(t, got, embeddedSecret)
	require.Contains(t, got, "intro")
}
