package templateviews_test

import (
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

// linkFixture builds one target note linked from (and linking to) three notes:
// one the viewer can read, one closed, one closed but in a teaser subgraph.
func linkFixture(t *testing.T) (*model.NoteViews, *model.NoteView) {
	t.Helper()

	nvs := model.NewNoteViews()

	open := &model.NoteView{PathID: 2, Path: "docs/open.md", Permalink: "/docs/open"}
	closed := &model.NoteView{
		PathID: 3, Path: "docs/closed.md", Permalink: "/docs/closed",
		SubgraphNames: []string{"vault"},
		Subgraphs:     map[string]*model.NoteSubgraph{"vault": {Name: "vault"}},
	}
	teased := &model.NoteView{
		PathID: 4, Path: "docs/teased.md", Permalink: "/docs/teased",
		SubgraphNames: []string{"shop"},
		Subgraphs:     map[string]*model.NoteSubgraph{"shop": {Name: "shop", Teaser: true}},
	}
	target := &model.NoteView{
		PathID: 1, Path: "docs/article.md", Permalink: "/docs/article",
		InLinks: map[string]struct{}{
			"/docs/open":   {},
			"/docs/closed": {},
			"/docs/teased": {},
		},
		ResolvedLinks: map[string]string{
			"open":   "/docs/open",
			"closed": "/docs/closed",
			"teased": "/docs/teased",
		},
	}

	for _, nv := range []*model.NoteView{target, open, closed, teased} {
		nvs.Map[nv.Permalink] = nv
		nvs.PathMap[nv.Path] = nv
	}

	return nvs, target
}

// canReadOnlyOpen is the viewer predicate: everything but the two subgraph
// notes is readable.
func canReadOnlyOpen(nv *model.NoteView) bool {
	return nv.Path == "docs/open.md" || nv.Path == "docs/article.md"
}

func pathsOf(notes []*templateviews.Note) []string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		out = append(out, n.Path())
	}
	return out
}

// Default is a wall: a backlink the viewer cannot read is absent entirely —
// no title, no URL. A teaser subgraph is the exception.
func TestNVS_BackLinks_WallsUnreadableNotes(t *testing.T) {
	nvs, target := linkFixture(t)

	wrapper := templateviews.NewNVS(nvs, "live").WithAccess(canReadOnlyOpen)
	backlinks := wrapper.BackLinks(wrapper.ByPath(target.Path))

	require.ElementsMatch(t, []string{"docs/open.md", "docs/teased.md"}, pathsOf(backlinks))
}

func TestNVS_OutLinks_WallsUnreadableNotes(t *testing.T) {
	nvs, target := linkFixture(t)

	wrapper := templateviews.NewNVS(nvs, "live").WithAccess(canReadOnlyOpen)
	outlinks := wrapper.OutLinks(wrapper.ByPath(target.Path))

	require.ElementsMatch(t, []string{"docs/open.md", "docs/teased.md"}, pathsOf(outlinks))
}

// Wall wins across subgraphs: a note that also sits in a non-teaser subgraph
// stays hidden even though one of its subgraphs is a teaser.
func TestNVS_BackLinks_WallWinsOverTeaser(t *testing.T) {
	nvs, target := linkFixture(t)

	teased := nvs.PathMap["docs/teased.md"]
	teased.SubgraphNames = append(teased.SubgraphNames, "vault")
	teased.Subgraphs["vault"] = &model.NoteSubgraph{Name: "vault"}

	wrapper := templateviews.NewNVS(nvs, "live").WithAccess(canReadOnlyOpen)
	backlinks := wrapper.BackLinks(wrapper.ByPath(target.Path))

	require.Equal(t, []string{"docs/open.md"}, pathsOf(backlinks))
}

// No predicate bound (preview render, smoke render, tests): nothing is filtered.
func TestNVS_BackLinks_NoAccessPredicateKeepsEverything(t *testing.T) {
	nvs, target := linkFixture(t)

	wrapper := templateviews.NewNVS(nvs, "live")
	backlinks := wrapper.BackLinks(wrapper.ByPath(target.Path))

	require.Len(t, backlinks, 3)
}
