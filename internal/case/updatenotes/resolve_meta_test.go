package updatenotes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/updatenotes"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func TestResolve_Meta(t *testing.T) {
	const stored = "---\ntitle: Hello # keep\nstatus: draft\n---\nBody\n"
	storedHash := hashContent(stored)
	wrongHash := "badhash=="

	tests := []struct {
		name        string
		patterns    []string
		meta        model.NoteChangeMetaInput
		wantContent string
		wantPayload model.UpdateNotesOrErrorPayload
	}{
		{
			name:        "set replaces and adds keys",
			meta:        model.NoteChangeMetaInput{Path: "note.md", Set: map[string]any{"status": "done", "done": true}},
			wantContent: "---\ntitle: Hello # keep\nstatus: done\ndone: true\n---\nBody\n",
		},
		{
			name:        "unset removes a key",
			meta:        model.NoteChangeMetaInput{Path: "note.md", Unset: []string{"status", "absent"}},
			wantContent: "---\ntitle: Hello # keep\n---\nBody\n",
		},
		{
			name:        "leading slash is normalized",
			meta:        model.NoteChangeMetaInput{Path: "/note.md", Set: map[string]any{"status": "done"}},
			wantContent: "---\ntitle: Hello # keep\nstatus: done\n---\nBody\n",
		},
		{
			name:        "matching expectedHash applies",
			meta:        model.NoteChangeMetaInput{Path: "note.md", Unset: []string{"status"}, ExpectedHash: &storedHash},
			wantContent: "---\ntitle: Hello # keep\n---\nBody\n",
		},
		{
			name:        "wrong expectedHash is a mismatch",
			meta:        model.NoteChangeMetaInput{Path: "note.md", Unset: []string{"status"}, ExpectedHash: &wrongHash},
			wantPayload: model.UpdateNotesHashMismatchPayload{Path: "note.md", ActualHash: storedHash},
		},
		{
			name:        "missing note",
			meta:        model.NoteChangeMetaInput{Path: "missing.md", Unset: []string{"status"}},
			wantPayload: &model.ErrorPayload{Message: "note not found: missing.md"},
		},
		{
			name:        "nothing to change",
			meta:        model.NoteChangeMetaInput{Path: "note.md"},
			wantPayload: &model.ErrorPayload{Message: "meta note.md: set or unset must name at least one key"},
		},
		{
			name:        "path outside write scope",
			patterns:    []string{"boards/**"},
			meta:        model.NoteChangeMetaInput{Path: "note.md", Unset: []string{"status"}},
			wantPayload: &model.ErrorPayload{Message: "write denied for path: note.md"},
		},
		{
			name:        "path escaping the vault",
			meta:        model.NoteChangeMetaInput{Path: "../note.md", Unset: []string{"status"}},
			wantPayload: &model.ErrorPayload{Message: "invalid path: ../note.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.patterns != nil {
				ctx = appreq.NewContext(ctx, &appreq.Request{WebhookWritePatterns: tt.patterns})
			}
			nvs := makeNVSWithNote("note.md", stored, 5)

			var inserted *appmodel.RawNote
			env := &mockEnv{
				latestNoteViews: func() *appmodel.NoteViews { return nvs },
				insertNote: func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					inserted = &note
					return appmodel.NoteSaveResult{PathID: 5, VersionID: 50}, nil
				},
				prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					return nvs, nil
				},
				handleLatestNotesAfterSave: noopHandle,
			}

			meta := tt.meta
			out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
				Changes: []model.NoteChangeInput{{Meta: &meta}},
			})
			require.NoError(t, err)

			if tt.wantPayload != nil {
				require.Equal(t, tt.wantPayload, out)
				require.Nil(t, inserted, "a refused meta change must not write")
				return
			}

			require.Equal(t, model.UpdateNotesSuccessPayload{
				Paths:   []string{"note.md"},
				Updated: []model.NoteWriteResult{{Path: "note.md", VersionID: 50, Warnings: []appmodel.NoteWarning{}}},
			}, out)
			require.NotNil(t, inserted)
			require.Equal(t, appmodel.RawNote{Path: "note.md", Content: tt.wantContent}, *inserted)
		})
	}
}

func TestResolve_MetaOnUneditableFrontmatter(t *testing.T) {
	nvs := makeNVSWithNote("note.md", "---\n{a: 1}\n---\nBody\n", 5)
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
	}

	out, err := updatenotes.Resolve(context.Background(), env, model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Meta: &model.NoteChangeMetaInput{Path: "note.md", Set: map[string]any{"a": 2}}},
		},
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected *ErrorPayload, got %T", out)
	require.Contains(t, ep.Message, "cannot be edited key by key")
}
