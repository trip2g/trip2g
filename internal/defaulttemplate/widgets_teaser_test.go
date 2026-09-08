package defaulttemplate

import (
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

// The rendered widgets, not just NVS: the docs promise a reader never sees the
// title or the address of a note they cannot read, unless every subgraph the
// note belongs to is a teaser. See docs/en/user/default-template.md.
func teaserWidgetCtx(t *testing.T) *Ctx {
	t.Helper()

	nvs := model.NewNoteViews()

	open := &model.NoteView{Path: "open.md", Title: "Open Note", Permalink: "/open"}
	closed := &model.NoteView{
		Path: "closed.md", Title: "Closed Note", Permalink: "/closed",
		SubgraphNames: []string{"vault"},
		Subgraphs:     map[string]*model.NoteSubgraph{"vault": {Name: "vault"}},
	}
	teased := &model.NoteView{
		Path: "teased.md", Title: "Teased Note", Permalink: "/teased",
		SubgraphNames: []string{"shop"},
		Subgraphs:     map[string]*model.NoteSubgraph{"shop": {Name: "shop", Teaser: true}},
	}
	target := &model.NoteView{
		Path: "target.md", Title: "Target", Permalink: "/target",
		InLinks: map[string]struct{}{"/open": {}, "/closed": {}, "/teased": {}},
		ResolvedLinks: map[string]string{
			"open": "/open", "closed": "/closed", "teased": "/teased",
		},
	}

	for _, nv := range []*model.NoteView{target, open, closed, teased} {
		nvs.Map[nv.Permalink] = nv
		nvs.PathMap[nv.Path] = nv
	}

	wrapper := templateviews.NewNVS(nvs, "live").WithAccess(func(nv *model.NoteView) bool {
		return nv.Path == "open.md" || nv.Path == "target.md"
	})

	return &Ctx{Note: wrapper.ByPath("target.md"), Notes: wrapper}
}

func TestInLinksWidget_HidesUnreadableNotes(t *testing.T) {
	html := InLinksWidget(teaserWidgetCtx(t))

	require.Contains(t, html, "Open Note")
	require.Contains(t, html, "Teased Note")
	require.NotContains(t, html, "Closed Note", "a closed note must not appear by title")
	require.NotContains(t, html, "/closed", "a closed note must not appear by address either")
}

func TestOutLinksWidget_HidesUnreadableNotes(t *testing.T) {
	html := OutLinksWidget(teaserWidgetCtx(t))

	require.Contains(t, html, "Open Note")
	require.Contains(t, html, "Teased Note")
	require.NotContains(t, html, "Closed Note", "a closed note must not appear by title")
	require.NotContains(t, html, "/closed", "a closed note must not appear by address either")
}
