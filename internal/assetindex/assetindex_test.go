package assetindex_test

import (
	"testing"

	"trip2g/internal/assetindex"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type testEnv struct {
	nvs        *model.NoteViews
	layouts    *model.Layouts
	generation uint64
}

func (e *testEnv) LiveNoteViews() *model.NoteViews { return e.nvs }
func (e *testEnv) Layouts() *model.Layouts         { return e.layouts }
func (e *testEnv) AssetIndexGeneration() uint64    { return e.generation }

func noteWithAsset(path, fileName string, free bool) *model.NoteView {
	const hash = "h1"
	return &model.NoteView{
		Path: path,
		Free: free,
		AssetReplaces: map[string]*model.NoteAssetReplace{
			"img.png": {ID: 1, Hash: hash, FileName: fileName, URL: model.NoteAssetURLPath(hash, fileName)},
		},
	}
}

func views(notes ...*model.NoteView) *model.NoteViews {
	return &model.NoteViews{List: notes}
}

func TestAssetOwnership_PublicViaFreeNote(t *testing.T) {
	env := &testEnv{nvs: views(noteWithAsset("post.md", "img.png", true))}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1", "img.png")
	require.True(t, ok)
	require.True(t, own.Public)
	require.Len(t, own.Notes, 1)
}

func TestAssetOwnership_PrivateViaPaidNote(t *testing.T) {
	env := &testEnv{nvs: views(noteWithAsset("post.md", "img.png", false))}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1", "img.png")
	require.True(t, ok)
	require.False(t, own.Public)
	require.Len(t, own.Notes, 1)
}

func TestAssetOwnership_FreeNoteInSigninSubgraphIsNotPublic(t *testing.T) {
	note := noteWithAsset("post.md", "img.png", true)
	note.Subgraphs = map[string]*model.NoteSubgraph{"members": {RequireSignin: true}}
	idx := assetindex.New(&testEnv{nvs: views(note)})

	own, ok := idx.AssetOwnership("h1", "img.png")
	require.True(t, ok)
	require.False(t, own.Public, "sign-in-walled note must not make its assets public")
}

func TestAssetOwnership_SharedAssetPublicIfAnyOwnerIsFree(t *testing.T) {
	env := &testEnv{nvs: views(
		noteWithAsset("paid.md", "img.png", false),
		noteWithAsset("free.md", "img.png", true),
	)}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1", "img.png")
	require.True(t, ok)
	require.True(t, own.Public)
	require.Len(t, own.Notes, 2)
}

func TestAssetOwnership_LayoutAssetIsPublic(t *testing.T) {
	env := &testEnv{
		nvs: views(),
		layouts: &model.Layouts{AssetReplaces: map[string]*model.NoteAssetReplace{
			"logo.svg": {ID: 2, Hash: "h2", FileName: "logo.svg"},
		}},
	}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h2", "logo.svg")
	require.True(t, ok)
	require.True(t, own.Public)
}

func TestAssetOwnership_UnknownHash(t *testing.T) {
	idx := assetindex.New(&testEnv{nvs: views()})

	_, ok := idx.AssetOwnership("nope", "nope.png")
	require.False(t, ok)
}

// TestAssetOwnership_SameHashDifferentFileNameIsIndependent pins the fix for
// a hash-only-keyed leak: two distinct note_assets rows can share a sha256
// (identical bytes) while having different file names and different owning
// notes. Ownership must be resolved per (hash, fileName), so a public row's
// publicness must not bleed onto a private sibling row that merely shares
// the hash.
func TestAssetOwnership_SameHashDifferentFileNameIsIndependent(t *testing.T) {
	publicNote := noteWithAsset("free.md", "public.png", true)
	privateNote := &model.NoteView{
		Path: "paid.md",
		Free: false,
		AssetReplaces: map[string]*model.NoteAssetReplace{
			"secret.png": {ID: 2, Hash: "h1", FileName: "private.png", URL: model.NoteAssetURLPath("h1", "private.png")},
		},
	}
	idx := assetindex.New(&testEnv{nvs: views(publicNote, privateNote)})

	pubOwn, ok := idx.AssetOwnership("h1", "public.png")
	require.True(t, ok)
	require.True(t, pubOwn.Public)
	require.Len(t, pubOwn.Notes, 1)

	privOwn, ok := idx.AssetOwnership("h1", "private.png")
	require.True(t, ok)
	require.False(t, privOwn.Public, "same-hash sibling with a different filename must stay private")
	require.Len(t, privOwn.Notes, 1)
	require.Equal(t, "paid.md", privOwn.Notes[0].Path)
}

// TestAssetOwnership_StaleGenerationTriggersRebuild pins the TOCTOU fix: the
// index must never trust a cached publicness decision built before the
// current generation — it self-checks env.AssetIndexGeneration() on every
// call instead of relying on an explicit invalidation call.
func TestAssetOwnership_StaleGenerationTriggersRebuild(t *testing.T) {
	env := &testEnv{nvs: views(noteWithAsset("post.md", "img.png", true)), generation: 1}
	idx := assetindex.New(env)

	own, ok := idx.AssetOwnership("h1", "img.png")
	require.True(t, ok)
	require.True(t, own.Public)

	// Simulate a reload that flips the note free -> paid, publishing a new
	// snapshot AND bumping the generation atomically (as noteloader.Loader
	// does) — no explicit invalidation call.
	env.nvs = views(noteWithAsset("post.md", "img.png", false))
	env.generation = 2

	own, ok = idx.AssetOwnership("h1", "img.png")
	require.True(t, ok)
	require.False(t, own.Public, "a bumped generation must force a rebuild even without an invalidation call")
}

// TestAssetOwnership_SameGenerationServesCachedIndex documents that the
// index does NOT rebuild on every call — only when the generation changes —
// so a stable snapshot doesn't pay the rebuild cost per request.
func TestAssetOwnership_SameGenerationServesCachedIndex(t *testing.T) {
	calls := 0
	env := &countingEnv{testEnv: testEnv{nvs: views(noteWithAsset("post.md", "img.png", true)), generation: 1}, calls: &calls}
	idx := assetindex.New(env)

	_, _ = idx.AssetOwnership("h1", "img.png")
	_, _ = idx.AssetOwnership("h1", "img.png")
	_, _ = idx.AssetOwnership("h1", "img.png")

	require.Equal(t, 1, calls, "LiveNoteViews should only be read once per generation (build happens once)")
}

type countingEnv struct {
	testEnv
	calls *int
}

func (e *countingEnv) LiveNoteViews() *model.NoteViews {
	*e.calls++
	return e.testEnv.nvs
}
