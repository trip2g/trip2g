package assetindex_test

import (
	"testing"

	"trip2g/internal/assetindex"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type testEnv struct {
	nvs     *model.NoteViews
	layouts *model.Layouts
}

func (e *testEnv) LiveNoteViews() *model.NoteViews { return e.nvs }
func (e *testEnv) Layouts() *model.Layouts         { return e.layouts }

func noteWithAsset(path, hash string, free bool) *model.NoteView {
	return &model.NoteView{
		Path: path,
		Free: free,
		AssetReplaces: map[string]*model.NoteAssetReplace{
			"img.png": {ID: 1, Hash: hash, URL: model.NoteAssetURLPath(hash, "img.png")},
		},
	}
}

func views(notes ...*model.NoteView) *model.NoteViews {
	return &model.NoteViews{List: notes}
}

func TestAssetOwnership_PublicViaFreeNote(t *testing.T) {
	env := &testEnv{nvs: views(noteWithAsset("post.md", "h1", true))}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1")
	require.True(t, ok)
	require.True(t, own.Public)
	require.Len(t, own.Notes, 1)
}

func TestAssetOwnership_PrivateViaPaidNote(t *testing.T) {
	env := &testEnv{nvs: views(noteWithAsset("post.md", "h1", false))}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1")
	require.True(t, ok)
	require.False(t, own.Public)
	require.Len(t, own.Notes, 1)
}

func TestAssetOwnership_FreeNoteInSigninSubgraphIsNotPublic(t *testing.T) {
	note := noteWithAsset("post.md", "h1", true)
	note.Subgraphs = map[string]*model.NoteSubgraph{"members": {RequireSignin: true}}
	idx := assetindex.New(&testEnv{nvs: views(note)})

	own, ok := idx.AssetOwnership("h1")
	require.True(t, ok)
	require.False(t, own.Public, "sign-in-walled note must not make its assets public")
}

func TestAssetOwnership_SharedAssetPublicIfAnyOwnerIsFree(t *testing.T) {
	env := &testEnv{nvs: views(
		noteWithAsset("paid.md", "h1", false),
		noteWithAsset("free.md", "h1", true),
	)}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1")
	require.True(t, ok)
	require.True(t, own.Public)
	require.Len(t, own.Notes, 2)
}

func TestAssetOwnership_LayoutAssetIsPublic(t *testing.T) {
	env := &testEnv{
		nvs: views(),
		layouts: &model.Layouts{AssetReplaces: map[string]*model.NoteAssetReplace{
			"logo.svg": {ID: 2, Hash: "h2"},
		}},
	}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h2")
	require.True(t, ok)
	require.True(t, own.Public)
}

func TestAssetOwnership_UnknownHash(t *testing.T) {
	idx := assetindex.New(&testEnv{nvs: views()})

	_, ok := idx.AssetOwnership("nope")
	require.False(t, ok)
}

func TestInvalidate_RebuildsFromCurrentSnapshot(t *testing.T) {
	env := &testEnv{nvs: views(noteWithAsset("post.md", "h1", true))}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1")
	require.True(t, ok)
	require.True(t, own.Public)

	// Note flips free -> paid on reload; without invalidation the stale index
	// would keep serving the asset publicly.
	env.nvs = views(noteWithAsset("post.md", "h1", false))

	own, _ = idx.AssetOwnership("h1")
	require.True(t, own.Public, "stale until invalidated (documents the contract)")

	idx.InvalidateAssetIndex()

	own, ok = idx.AssetOwnership("h1")
	require.True(t, ok)
	require.False(t, own.Public, "after invalidation the paid flag must win")
}
