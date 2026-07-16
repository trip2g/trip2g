package materializenotefrontmatters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

func TestResolveUsesEffectiveMetadataAndRemovesDeletedKeys(t *testing.T) {
	env := &EnvMock{
		UpsertNoteVersionFrontmatterFunc: func(
			context.Context, db.UpsertNoteVersionFrontmatterParams,
		) error {
			return nil
		},
		DeleteNoteVersionFrontmatterKeysFunc: func(context.Context, int64) error {
			return nil
		},
		UpsertNoteVersionFrontmatterKeyFunc: func(
			context.Context, db.UpsertNoteVersionFrontmatterKeyParams,
		) error {
			return nil
		},
		InsertNoteVersionFrontmatterKeyFunc: func(
			context.Context, db.InsertNoteVersionFrontmatterKeyParams,
		) error {
			return nil
		},
		RefreshNoteVersionFrontmatterKeyVisibilityFunc: func(context.Context) error {
			return nil
		},
	}

	ctx := context.Background()
	note := &model.NoteView{
		VersionID: 42,
		RawMeta:   map[string]interface{}{"fleet_id": "codellm"},
	}

	require.NoError(t, Resolve(ctx, env, []*model.NoteView{note}))

	frontmatterCalls := env.UpsertNoteVersionFrontmatterCalls()
	require.Len(t, frontmatterCalls, 1)

	frontmatter := frontmatterCalls[0].UpsertNoteVersionFrontmatterParams
	require.Equal(t, int64(42), frontmatter.VersionID)
	require.JSONEq(t, `{"fleet_id":"codellm"}`, frontmatter.Data.(string))

	deletedCalls := env.DeleteNoteVersionFrontmatterKeysCalls()
	keyCalls := env.UpsertNoteVersionFrontmatterKeyCalls()
	linkCalls := env.InsertNoteVersionFrontmatterKeyCalls()
	refreshCalls := env.RefreshNoteVersionFrontmatterKeyVisibilityCalls()

	require.Len(t, deletedCalls, 1)
	require.Equal(t, int64(42), deletedCalls[0].N)
	require.Len(t, keyCalls, 1)
	require.Equal(t, "fleet_id", keyCalls[0].UpsertNoteVersionFrontmatterKeyParams.Value)
	require.Len(t, linkCalls, 1)
	require.Equal(t, "fleet_id", linkCalls[0].InsertNoteVersionFrontmatterKeyParams.KeyID)
	require.Len(t, refreshCalls, 1)
}

func TestResolveRemovesKeysFromVersionWhenEffectiveMetadataIsEmpty(t *testing.T) {
	env := &EnvMock{
		UpsertNoteVersionFrontmatterFunc: func(
			context.Context, db.UpsertNoteVersionFrontmatterParams,
		) error {
			return nil
		},
		DeleteNoteVersionFrontmatterKeysFunc: func(context.Context, int64) error {
			return nil
		},
		UpsertNoteVersionFrontmatterKeyFunc: func(
			context.Context, db.UpsertNoteVersionFrontmatterKeyParams,
		) error {
			return nil
		},
		InsertNoteVersionFrontmatterKeyFunc: func(
			context.Context, db.InsertNoteVersionFrontmatterKeyParams,
		) error {
			return nil
		},
		RefreshNoteVersionFrontmatterKeyVisibilityFunc: func(context.Context) error {
			return nil
		},
	}

	note := &model.NoteView{VersionID: 42, RawMeta: map[string]interface{}{}}
	require.NoError(t, Resolve(context.Background(), env, []*model.NoteView{note}))

	deleteCalls := env.DeleteNoteVersionFrontmatterKeysCalls()
	require.Len(t, deleteCalls, 1)
	require.Equal(t, int64(42), deleteCalls[0].N)
	require.Empty(t, env.UpsertNoteVersionFrontmatterKeyCalls())
	require.Empty(t, env.InsertNoteVersionFrontmatterKeyCalls())
	require.Len(t, env.RefreshNoteVersionFrontmatterKeyVisibilityCalls(), 1)
}

func TestResolveNormalizesNestedYAMLMetadata(t *testing.T) {
	env := &EnvMock{
		UpsertNoteVersionFrontmatterFunc: func(
			context.Context, db.UpsertNoteVersionFrontmatterParams,
		) error {
			return nil
		},
		DeleteNoteVersionFrontmatterKeysFunc: func(context.Context, int64) error {
			return nil
		},
		UpsertNoteVersionFrontmatterKeyFunc: func(
			context.Context, db.UpsertNoteVersionFrontmatterKeyParams,
		) error {
			return nil
		},
		InsertNoteVersionFrontmatterKeyFunc: func(
			context.Context, db.InsertNoteVersionFrontmatterKeyParams,
		) error {
			return nil
		},
		RefreshNoteVersionFrontmatterKeyVisibilityFunc: func(context.Context) error {
			return nil
		},
	}

	note := &model.NoteView{
		VersionID: 42,
		RawMeta: map[string]interface{}{
			"form": map[interface{}]interface{}{
				"name": "newsletter",
			},
		},
	}

	require.NoError(t, Resolve(context.Background(), env, []*model.NoteView{note}))
	calls := env.UpsertNoteVersionFrontmatterCalls()
	require.Len(t, calls, 1)
	require.JSONEq(t, `{"form":{"name":"newsletter"}}`, calls[0].UpsertNoteVersionFrontmatterParams.Data.(string))
}
