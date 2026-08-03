package rendernotepage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

type introPartialRenderer struct {
	intro model.NoteViewSection
}

func (r introPartialRenderer) Sections(int) []model.NoteViewSection      { return nil }
func (r introPartialRenderer) Section(string) *model.NoteViewSection     { return nil }
func (r introPartialRenderer) Introduce() model.NoteViewSection          { return r.intro }
func (r introPartialRenderer) HeadingBlocks(int) []model.NoteViewSection { return nil }
func (r introPartialRenderer) FirstList() *model.NoteViewList            { return nil }
func (r introPartialRenderer) Lists() []model.NoteViewList               { return nil }
func (r introPartialRenderer) FirstImageURL() string                     { return "" }

func TestDeriveMetaDescription(t *testing.T) {
	t.Run("nil note", func(t *testing.T) {
		require.Empty(t, deriveMetaDescription(nil))
	})

	t.Run("no partial renderer", func(t *testing.T) {
		require.Empty(t, deriveMetaDescription(&model.NoteView{}))
	})

	t.Run("strips html and collapses whitespace", func(t *testing.T) {
		note := &model.NoteView{Free: true, PartialRenderer: introPartialRenderer{
			intro: model.NoteViewSection{ContentHTML: "<p>Hello   <strong>world</strong>\nagain</p>"},
		}}
		require.Equal(t, "Hello world again", deriveMetaDescription(note))
	})

	// The derived description reaches the <head> of the paywall / sign-in wall
	// pages too, so a note that is not anonymous-visible must never be summarized
	// from its body.
	t.Run("non-free note yields nothing", func(t *testing.T) {
		note := &model.NoteView{PartialRenderer: introPartialRenderer{
			intro: model.NoteViewSection{ContentHTML: "<p>Paid content</p>"},
		}}
		require.Empty(t, deriveMetaDescription(note))
	})

	t.Run("require_signin subgraph yields nothing", func(t *testing.T) {
		note := &model.NoteView{
			Free:      true,
			Subgraphs: map[string]*model.NoteSubgraph{"members": {RequireSignin: true}},
			PartialRenderer: introPartialRenderer{
				intro: model.NoteViewSection{ContentHTML: "<p>Members only</p>"},
			},
		}
		require.Empty(t, deriveMetaDescription(note))
	})

	t.Run("truncates to about 155 chars with ellipsis on word boundary", func(t *testing.T) {
		long := strings.Repeat("word ", 60) // 300 chars
		note := &model.NoteView{Free: true, PartialRenderer: introPartialRenderer{
			intro: model.NoteViewSection{ContentHTML: "<p>" + long + "</p>"},
		}}
		got := deriveMetaDescription(note)
		require.LessOrEqual(t, len([]rune(got)), metaDescriptionMaxLen+1) // +1 for the ellipsis
		require.True(t, strings.HasSuffix(got, "…"))
		// Truncation happens on a word boundary: dropping the ellipsis leaves only
		// whole "word" tokens, never a partial fragment.
		trimmed := strings.TrimSuffix(got, "…")
		for _, tok := range strings.Fields(trimmed) {
			require.Equal(t, "word", tok)
		}
	})
}
