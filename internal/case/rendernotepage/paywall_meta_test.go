package rendernotepage_test

import (
	"strings"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// leakPartialRenderer feeds deriveMetaDescription a first paragraph with a
// recognizable marker, the way the real markdown renderer would.
type leakPartialRenderer struct {
	intro string
}

func (r leakPartialRenderer) Sections(int) []model.NoteViewSection  { return nil }
func (r leakPartialRenderer) Section(string) *model.NoteViewSection { return nil }
func (r leakPartialRenderer) Introduce() model.NoteViewSection {
	return model.NoteViewSection{ContentHTML: "<p>" + r.intro + "</p>"}
}
func (r leakPartialRenderer) HeadingBlocks(int) []model.NoteViewSection { return nil }
func (r leakPartialRenderer) FirstList() *model.NoteViewList            { return nil }
func (r leakPartialRenderer) Lists() []model.NoteViewList               { return nil }
func (r leakPartialRenderer) FirstImageURL() string                     { return "" }

const secretIntro = "SECRET-INTRO-MARKER the closed body starts right here"

// TestPaywalledNoteBodyNeverLeaksIntoMeta pins the fix for a content leak: the
// meta description is derived from the note's first paragraph, and it was built
// before the paywall / sign-in wall branch ran — so an anonymous GET of a closed
// note answered 200 with that paragraph in <meta name="description">, in
// og:description and once more through OGTagsSorted. The title is shown on
// purpose; the body is not.
func TestPaywalledNoteBodyNeverLeaksIntoMeta(t *testing.T) {
	cases := []struct {
		name  string
		setup func(note *model.NoteView)
	}{
		{
			name:  "paywall (non-free note)",
			setup: func(note *model.NoteView) { note.Free = false },
		},
		{
			name: "sign-in wall (require_signin subgraph)",
			setup: func(note *model.NoteView) {
				note.Subgraphs = map[string]*model.NoteSubgraph{
					"members": {Name: "members", RequireSignin: true},
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			note, views := cacheTestNote()
			note.PartialRenderer = leakPartialRenderer{intro: secretIntro}
			tc.setup(note)
			env, _, _ := cacheTestEnv(views, nil)

			ctx := newReqCtx(reqOpts{})
			runHandle(t, env, ctx, nil)

			html := string(body(t, ctx))
			require.NotContains(t, html, secretIntro,
				"closed note body must not reach an anonymous reader")
			require.Contains(t, html, note.Title, "the title is shown on purpose")
		})
	}

	t.Run("explicit description is still published", func(t *testing.T) {
		note, views := cacheTestNote()
		note.Free = false
		note.PartialRenderer = leakPartialRenderer{intro: secretIntro}
		note.Description = ptrTo("Teaser written for the paywall page")
		env, _, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{})
		runHandle(t, env, ctx, nil)

		html := string(body(t, ctx))
		require.NotContains(t, html, secretIntro)
		require.Contains(t, html, "Teaser written for the paywall page")
	})

	t.Run("free note still gets a derived description", func(t *testing.T) {
		note, views := cacheTestNote()
		note.PartialRenderer = leakPartialRenderer{intro: secretIntro}
		env, _, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{})
		runHandle(t, env, ctx, nil)

		html := string(body(t, ctx))
		require.Equal(t, 3, strings.Count(html, secretIntro),
			"public note: meta description, og:description and the twitter/og pass")
	})
}

func ptrTo[T any](v T) *T { return &v }
