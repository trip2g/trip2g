package hidenotes

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	internalmodel "trip2g/internal/model"
	"trip2g/internal/notebus"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type envMock = EnvMock

func newTestEnv(hideFunc func(ctx context.Context, params db.HideNotePathParams) error) *envMock {
	return &envMock{
		HideNotePathFunc:    hideFunc,
		LatestNoteViewsFunc: internalmodel.NewNoteViews,
		PrepareLatestNotesFunc: func(_ context.Context, _ bool) (*internalmodel.NoteViews, error) {
			return internalmodel.NewNoteViews(), nil
		},
		LoggerFunc: func() logger.Logger {
			return slog.Default()
		},
		PublishNoteChangesFunc: func(_ notebus.Batch) {},
	}
}

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.HideNotesInput
	}

	tests := []struct {
		name          string
		env           Env
		args          args
		want          model.HideNotesOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful hide single note",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return nil
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{"/test/note.md"},
					ApiKey: db.ApiKey{
						CreatedBy: 123,
					},
				},
			},
			want: &model.HideNotesPayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.HideNotePathCalls(), 1)

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/test/note.md", hideParams.Value)
				require.NotNil(t, hideParams.HiddenBy)
				require.Equal(t, int64(123), *hideParams.HiddenBy)

				// Hiding must reload NoteViews so the page stops resolving.
				require.Len(t, mockEnv.PrepareLatestNotesCalls(), 1)
			},
		},
		{
			name: "successful hide multiple notes",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return nil
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{"/test/note1.md", "/test/note2.md", "/folder/note3.md"},
					ApiKey: db.ApiKey{
						CreatedBy: 456,
					},
				},
			},
			want: &model.HideNotesPayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.HideNotePathCalls(), 3)

				expectedPaths := []string{"/test/note1.md", "/test/note2.md", "/folder/note3.md"}
				for i, call := range mockEnv.HideNotePathCalls() {
					require.Equal(t, expectedPaths[i], call.Params.Value)
					require.NotNil(t, call.Params.HiddenBy)
					require.Equal(t, int64(456), *call.Params.HiddenBy)
				}

				// A single reload covers the whole batch.
				require.Len(t, mockEnv.PrepareLatestNotesCalls(), 1)
			},
		},
		{
			name: "database error when hiding first note",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return errors.New("database connection failed")
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{"/another/note.md", "/second/note.md"},
					ApiKey: db.ApiKey{
						CreatedBy: 789,
					},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.HideNotePathCalls(), 1)

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/another/note.md", hideParams.Value)
				require.NotNil(t, hideParams.HiddenBy)
				require.Equal(t, int64(789), *hideParams.HiddenBy)

				// Hide failed -> no reload.
				require.Empty(t, mockEnv.PrepareLatestNotesCalls())
			},
		},
		{
			name: "hide notes with empty paths array",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return nil
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{},
					ApiKey: db.ApiKey{
						CreatedBy: 999,
					},
				},
			},
			want: &model.HideNotesPayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.HideNotePathCalls())
				// No paths -> nothing to hide, no reload.
				require.Empty(t, mockEnv.PrepareLatestNotesCalls())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got, pretty.Diff(tt.want, got))
			}

			if tt.afterCallback != nil {
				mockEnv, ok := tt.env.(*envMock)
				require.True(t, ok, "env should be a mock for callback tests")
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}

// TestResolve_ScopedToken pins the scope guard: hideNotes is a bulk-hide infra
// mutation that never consults write_patterns, so a scoped shortapitoken must
// be denied outright; unscoped full API keys keep working.
func TestResolve_ScopedToken(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		wantDenied bool
	}{
		{
			name: "scoped token denied",
			ctx: appreq.NewContext(context.Background(), &appreq.Request{
				WebhookScoped:        true,
				WebhookDeliveryKind:  "change",
				WebhookWritePatterns: []string{"notes/**"},
			}),
			wantDenied: true,
		},
		{
			name:       "unscoped key allowed",
			ctx:        context.Background(),
			wantDenied: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(func(_ context.Context, _ db.HideNotePathParams) error { return nil })

			result, err := Resolve(tt.ctx, env, model.HideNotesInput{Paths: []string{"note.md"}, ApiKey: db.ApiKey{CreatedBy: 1}})
			require.NoError(t, err)

			if tt.wantDenied {
				ep, ok := result.(*model.ErrorPayload)
				require.True(t, ok, "expected *ErrorPayload for scoped token, got %T", result)
				require.Contains(t, ep.Message, "scoped token")
				require.Empty(t, env.HideNotePathCalls(), "HideNotePath must not be reached for scoped tokens")
				return
			}
			require.IsType(t, &model.HideNotesPayload{}, result)
			require.Len(t, env.HideNotePathCalls(), 1)
		})
	}
}

// expectedHash gates each hide on its own: a mismatch leaves that note
// visible and reports it, a matched "" has nothing to hide and touches
// nothing, and paths without expectedHash hide as before.
func TestResolve_ExpectedHash(t *testing.T) {
	current := map[string]string{
		"match.md":    "hash-match",
		"conflict.md": "hash-server",
	}
	ptr := func(s string) *string { return &s }

	env := newTestEnv(func(_ context.Context, _ db.HideNotePathParams) error { return nil })
	env.LatestNoteContentHashFunc = func(_ context.Context, path string) (string, error) {
		return current[path], nil
	}
	var published notebus.Batch
	env.PublishNoteChangesFunc = func(batch notebus.Batch) { published = batch }

	result, err := Resolve(context.Background(), env, model.HideNotesInput{
		Paths: []string{"free.md"},
		Notes: []model.HideNoteInput{
			{Path: "match.md", ExpectedHash: ptr("hash-match")},
			{Path: "conflict.md", ExpectedHash: ptr("hash-base")},
			{Path: "gone.md", ExpectedHash: ptr("")},
			{Path: "unconditional.md"},
		},
		ApiKey: db.ApiKey{CreatedBy: 1},
	})
	require.NoError(t, err)

	require.Equal(t, &model.HideNotesPayload{
		Success:        true,
		HashMismatches: []model.NoteHashMismatch{{Path: "conflict.md", ActualHash: "hash-server"}},
	}, result)

	hidden := []string{}
	for _, call := range env.HideNotePathCalls() {
		hidden = append(hidden, call.Params.Value)
	}
	want := []string{"free.md", "match.md", "unconditional.md"}
	require.Equal(t, want, hidden)

	removed := []string{}
	for _, change := range published.Changes {
		removed = append(removed, change.Path)
	}
	require.Equal(t, want, removed, "a refused or absent path is not announced as removed")

	checked := []string{}
	for _, call := range env.LatestNoteContentHashCalls() {
		checked = append(checked, call.Path)
	}
	require.Equal(t, []string{"match.md", "conflict.md", "gone.md"}, checked)
}

// When every path is refused nothing changed, so the views are not reloaded.
func TestResolve_ExpectedHashAllRefused(t *testing.T) {
	env := newTestEnv(func(_ context.Context, _ db.HideNotePathParams) error { return nil })
	env.LatestNoteContentHashFunc = func(_ context.Context, _ string) (string, error) {
		return "hash-server", nil
	}
	expected := "hash-base"

	result, err := Resolve(context.Background(), env, model.HideNotesInput{
		Notes:  []model.HideNoteInput{{Path: "note.md", ExpectedHash: &expected}},
		ApiKey: db.ApiKey{CreatedBy: 1},
	})
	require.NoError(t, err)

	payload, ok := result.(*model.HideNotesPayload)
	require.True(t, ok, "got %T", result)
	require.Len(t, payload.HashMismatches, 1)
	require.Empty(t, env.HideNotePathCalls())
	require.Empty(t, env.PrepareLatestNotesCalls())
	require.Empty(t, env.PublishNoteChangesCalls())
}

// A request whose only entry is a matched "" changes nothing, so nothing is
// reloaded, announced or sent to webhooks.
func TestResolve_ExpectedHashEmptyIsNoOp(t *testing.T) {
	env := newTestEnv(func(_ context.Context, _ db.HideNotePathParams) error { return nil })
	env.LatestNoteContentHashFunc = func(_ context.Context, _ string) (string, error) {
		return "", nil
	}
	empty := ""

	result, err := Resolve(context.Background(), env, model.HideNotesInput{
		Notes:  []model.HideNoteInput{{Path: "gone.md", ExpectedHash: &empty}},
		ApiKey: db.ApiKey{CreatedBy: 1},
	})
	require.NoError(t, err)
	require.Equal(t, &model.HideNotesPayload{Success: true}, result)
	require.Empty(t, env.HideNotePathCalls())
	require.Empty(t, env.PrepareLatestNotesCalls())
	require.Empty(t, env.PublishNoteChangesCalls())
	require.Empty(t, env.HandleNoteWebhooksCalls())
}
