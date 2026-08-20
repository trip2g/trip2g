package pushnotes_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg pushnotes_test . Env

type Env interface {
	Logger() logger.Logger
	InsertNote(ctx context.Context, update appmodel.RawNote) (int64, error)
	InsertUncommittedPath(ctx context.Context, notePathID int64) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error
	Layouts() *appmodel.Layouts
	LatestNoteViews() *appmodel.NoteViews
	CheckStorageLimits(ctx context.Context, additionalAssetBytes int64) (string, error)
	PublicURL() string
}

// newEnvMock returns an EnvMock with safe defaults for all methods.
// Individual tests override only the methods they care about.
func newEnvMock(log logger.Logger) *EnvMock {
	return &EnvMock{
		LoggerFunc:             func() logger.Logger { return log },
		CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
		PublicURLFunc:          func() string { return "" },
		// A push where nothing changed skips the reload and reports against the
		// snapshot already in memory.
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
	}
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	mockLogger := &logger.TestLogger{}

	tests := []struct {
		name     string
		input    model.PushNotesInput
		setupEnv func() *EnvMock
		wantErr  bool
		validate func(t *testing.T, result model.PushNotesOrErrorPayload)
	}{
		{
			name: "unsupported file extension",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.txt", Content: "content"},
				},
			},
			setupEnv: func() *EnvMock {
				return newEnvMock(mockLogger)
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				errPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Contains(t, errPayload.Message, ".canvas")
				require.Contains(t, errPayload.Message, ".base")
			},
		},
		{
			name: "canvas file accepted",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "demo.canvas", Content: `{"nodes":[],"edges":[]}`},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					require.Equal(t, "demo.canvas", note.Path)
					return appmodel.NoteSaveResult{PathID: 1, VersionID: 1}, nil
				}
				env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				_, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
			},
		},
		{
			name: "excalidraw file accepted",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{
						Path:    "example.excalidraw",
						Content: `{"type":"excalidraw","version":2,"source":"https://excalidraw.com","elements":[],"appState":{},"files":{}}`,
					},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					require.Equal(t, "example.excalidraw", note.Path)
					return appmodel.NoteSaveResult{PathID: 3, VersionID: 3}, nil
				}
				env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				_, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
			},
		},
		{
			name: "base file accepted",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "mybase.base", Content: "type: base\nname: My Base\n"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					require.Equal(t, "mybase.base", note.Path)
					return appmodel.NoteSaveResult{PathID: 2, VersionID: 2}, nil
				}
				env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				_, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
			},
		},
		{
			name: "successful push with md file",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello World"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(ctx context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					return appmodel.NoteSaveResult{PathID: 1, VersionID: 1}, nil
				}
				env.PrepareLatestNotesFunc = func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
					return &appmodel.NoteViews{
						List: []*appmodel.NoteView{
							{
								Path:      "test.md",
								PathID:    1,
								VersionID: 100,
								Assets:    map[string]struct{}{},
							},
						},
						Subgraphs: map[string]*appmodel.NoteSubgraph{},
					}, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(ctx context.Context, changedPathIDs []int64) error {
					return nil
				}
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				payload, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
				require.Len(t, payload.Notes, 1)
				require.Equal(t, "test.md", payload.Notes[0].Path)
			},
		},
		{
			name: "insert note fails",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(ctx context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					return appmodel.NoteSaveResult{}, errors.New("database error")
				}
				return env
			},
			wantErr: true,
		},
		{
			name: "storage limit exceeded",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.CheckStorageLimitsFunc = func(_ context.Context, _ int64) (string, error) {
					return "database storage limit exceeded", nil
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				errPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Equal(t, "database storage limit exceeded", errPayload.Message)
			},
		},
		{
			name: "prepare latest notes fails",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(ctx context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					return appmodel.NoteSaveResult{PathID: 1, VersionID: 1}, nil
				}
				env.PrepareLatestNotesFunc = func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
					return nil, errors.New("prepare error")
				}
				return env
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()

			result, err := pushnotes.Resolve(ctx, env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// P1: a mid-batch validation failure must not leave earlier updates persisted
// (gitapi rolls back only the git ref on failure, so a partial DB write diverges DB from Git).
func TestResolve_MidBatchValidationFailure_NoPartialWrite(t *testing.T) {
	ctx := context.Background()
	env := newEnvMock(&logger.TestLogger{})

	insertCount := 0
	env.InsertNoteFunc = func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
		insertCount++
		return appmodel.NoteSaveResult{PathID: int64(insertCount), VersionID: int64(insertCount)}, nil
	}

	result, err := pushnotes.Resolve(ctx, env, model.PushNotesInput{
		Updates: []model.PushNoteInput{
			{Path: "a.md", Content: "# A\n"},
			{Path: "b.txt", Content: "not allowed"},
		},
	})
	require.NoError(t, err)
	_, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for invalid second update")
	require.Equal(t, 0, insertCount, "no update may be persisted when any update in the batch is invalid")
}

func TestResolve_UpdatedNotes(t *testing.T) {
	ctx := context.Background()
	mockLogger := &logger.TestLogger{}

	makeNVS := func() *appmodel.NoteViews {
		nvs := appmodel.NewNoteViews()
		note := &appmodel.NoteView{
			PathID:    42,
			VersionID: 1,
			Path:      "my-note.md",
			Permalink: "/my-note",
		}
		note.AddWarning(appmodel.NoteWarningWarning, "broken link to [[missing]]")
		nvs.RegisterNote(note)
		nvs.ExtractNoteList()
		return nvs
	}

	t.Run("updated contains pushed notes with url and warnings", func(t *testing.T) {
		env := newEnvMock(mockLogger)
		env.InsertNoteFunc = func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: 42, VersionID: 42}, nil
		}
		env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return makeNVS(), nil
		}
		env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error {
			return nil
		}
		env.LayoutsFunc = func() *appmodel.Layouts {
			return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
		}
		env.PublicURLFunc = func() string { return "https://example.com" }

		input := model.PushNotesInput{
			Updates: []model.PushNoteInput{
				{Path: "my-note.md", Content: "# Hello"},
			},
		}

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)

		require.Len(t, payload.Updated, 1)
		require.Equal(t, "my-note.md", payload.Updated[0].Path)
		require.NotNil(t, payload.Updated[0].URL)
		require.Equal(t, "https://example.com/my-note", *payload.Updated[0].URL)
		require.Len(t, payload.Updated[0].Warnings, 1)
		require.Equal(t, "broken link to [[missing]]", payload.Updated[0].Warnings[0].Message)

		require.Len(t, payload.Notes, 1)
		require.NotNil(t, payload.Notes[0].URL)
		require.Equal(t, "https://example.com/my-note", *payload.Notes[0].URL)
	})

	t.Run("updated is empty when no updates provided", func(t *testing.T) {
		env := newEnvMock(mockLogger)
		env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
			return makeNVS()
		}
		env.LayoutsFunc = func() *appmodel.Layouts {
			return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
		}
		env.PublicURLFunc = func() string { return "https://example.com" }

		input := model.PushNotesInput{Updates: []model.PushNoteInput{}}

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)
		require.Empty(t, payload.Updated)
	})
}

// TestResolve_LayoutWarnings covers the bug where a layout's parse/render
// warnings were dropped: buildPushedNotes hardcoded an empty Warnings slice
// for every layout instead of forwarding layout.Warnings.
func TestResolve_LayoutWarnings(t *testing.T) {
	ctx := context.Background()
	mockLogger := &logger.TestLogger{}

	env := newEnvMock(mockLogger)
	env.InsertNoteFunc = func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
		return appmodel.NoteSaveResult{PathID: 1, VersionID: 1}, nil
	}
	env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
		nvs := appmodel.NewNoteViews()
		nvs.ExtractNoteList()
		return nvs, nil
	}
	env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
	env.LayoutsFunc = func() *appmodel.Layouts {
		return &appmodel.Layouts{
			Map: map[string]appmodel.Layout{
				"broken": {
					VersionID: 7,
					Path:      "_layouts/broken.html",
					Warnings: []appmodel.NoteWarning{
						{Level: appmodel.NoteWarningCritical, Message: "parse error: unexpected token"},
					},
				},
			},
		}
	}
	env.PublicURLFunc = func() string { return "https://example.com" }

	input := model.PushNotesInput{
		Updates: []model.PushNoteInput{
			{Path: "_layouts/broken.html", Content: "{{ broken"},
		},
	}

	result, err := pushnotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(*model.PushNotesPayload)
	require.True(t, ok)

	require.Len(t, payload.Notes, 1)
	require.Len(t, payload.Notes[0].Warnings, 1)
	require.Equal(t, appmodel.NoteWarningCritical, payload.Notes[0].Warnings[0].Level)
	require.Equal(t, "parse error: unexpected token", payload.Notes[0].Warnings[0].Message)
}

// TestResolve_AssetURLAbsolutization covers the pushNotes GraphQL boundary:
// note/layout asset URLs are content-addressed relative paths
// (/_system/assets/{sha256}/{fileName}) and must be absolutized with
// PublicURL before crossing to a consumer on a different origin (see
// internal/model/asseturl.go doc comment on NoteAssetURLPath).
func TestResolve_AssetURLAbsolutization(t *testing.T) {
	ctx := context.Background()
	mockLogger := &logger.TestLogger{}

	const relativeAssetURL = "/_system/assets/deadbeef/image.png"

	makeNVSWithAsset := func() *appmodel.NoteViews {
		nvs := appmodel.NewNoteViews()
		note := &appmodel.NoteView{
			PathID:    42,
			VersionID: 1,
			Path:      "my-note.md",
			Permalink: "/my-note",
			Assets:    map[string]struct{}{"image.png": {}},
			AssetReplaces: map[string]*appmodel.NoteAssetReplace{
				"image.png": {ID: 7, URL: relativeAssetURL, Hash: "deadbeef"},
			},
		}
		nvs.RegisterNote(note)
		nvs.ExtractNoteList()
		return nvs
	}

	makeLayoutsWithAsset := func() *appmodel.Layouts {
		return &appmodel.Layouts{
			Map: map[string]appmodel.Layout{
				"_layouts/main.html": {
					VersionID: 2,
					Path:      "_layouts/main.html",
					Assets:    []appmodel.LayoutAsset{{Path: "logo.png", Hash: "beadfeed"}},
					AssetReplaces: map[string]*appmodel.NoteAssetReplace{
						"logo.png": {ID: 8, URL: "/_system/assets/beadfeed/logo.png", Hash: "beadfeed"},
					},
				},
			},
		}
	}

	setupEnv := func(publicURL string) *EnvMock {
		env := newEnvMock(mockLogger)
		env.InsertNoteFunc = func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: 42, VersionID: 42}, nil
		}
		env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return makeNVSWithAsset(), nil
		}
		env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error {
			return nil
		}
		env.LayoutsFunc = makeLayoutsWithAsset
		env.PublicURLFunc = func() string { return publicURL }
		return env
	}

	input := model.PushNotesInput{
		Updates: []model.PushNoteInput{
			{Path: "my-note.md", Content: "# Hello"},
		},
	}

	t.Run("public URL set absolutizes both note and layout asset urls", func(t *testing.T) {
		env := setupEnv("https://example.com")

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)

		require.Len(t, payload.Updated, 1)
		require.Len(t, payload.Updated[0].Assets, 1)
		require.Equal(t, "https://example.com"+relativeAssetURL, payload.Updated[0].Assets[0].URL)

		var noteEntry, layoutEntry *model.PushedNote
		for i := range payload.Notes {
			switch payload.Notes[i].Path {
			case "my-note.md":
				noteEntry = &payload.Notes[i]
			case "_layouts/main.html":
				layoutEntry = &payload.Notes[i]
			}
		}
		require.NotNil(t, noteEntry)
		require.Len(t, noteEntry.Assets, 1)
		require.Equal(t, "https://example.com"+relativeAssetURL, noteEntry.Assets[0].URL)

		require.NotNil(t, layoutEntry)
		require.Len(t, layoutEntry.Assets, 1)
		require.Equal(t, "https://example.com/_system/assets/beadfeed/logo.png", layoutEntry.Assets[0].URL)
	})

	t.Run("empty public URL leaves relative asset urls untouched", func(t *testing.T) {
		env := setupEnv("")

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)

		require.Len(t, payload.Updated, 1)
		require.Equal(t, relativeAssetURL, payload.Updated[0].Assets[0].URL)
	})

	t.Run("already-absolute asset url passes through unchanged", func(t *testing.T) {
		env := setupEnv("https://example.com")
		env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			nvs := appmodel.NewNoteViews()
			note := &appmodel.NoteView{
				PathID:    42,
				VersionID: 1,
				Path:      "my-note.md",
				Permalink: "/my-note",
				Assets:    map[string]struct{}{"image.png": {}},
				AssetReplaces: map[string]*appmodel.NoteAssetReplace{
					"image.png": {ID: 7, URL: "https://cdn.other.com/image.png", Hash: "deadbeef"},
				},
			}
			nvs.RegisterNote(note)
			nvs.ExtractNoteList()
			return nvs, nil
		}

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)

		require.Len(t, payload.Updated, 1)
		require.Equal(t, "https://cdn.other.com/image.png", payload.Updated[0].Assets[0].URL)
	})
}

// TestResolve_ScopedToken pins the scope guard: pushNotes is the infra/sync
// lane (full-vault push) with no per-path scoping, so a scoped shortapitoken
// must be denied outright; unscoped full API keys keep working.
func TestResolve_ScopedToken(t *testing.T) {
	mockLogger := &logger.TestLogger{}

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
			env := newEnvMock(mockLogger)
			env.LatestNoteViewsFunc = appmodel.NewNoteViews
			env.LayoutsFunc = func() *appmodel.Layouts {
				return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
			}

			result, err := pushnotes.Resolve(tt.ctx, env, model.PushNotesInput{})
			require.NoError(t, err)

			if tt.wantDenied {
				ep, ok := result.(*model.ErrorPayload)
				require.True(t, ok, "expected *ErrorPayload for scoped token, got %T", result)
				require.Contains(t, ep.Message, "scoped token")
				return
			}
			require.IsType(t, &model.PushNotesPayload{}, result)
		})
	}
}

// TestResolve_UnchangedPushIsInert pins the no-op rule on the sync lane: a resync
// that re-pushes identical content stores nothing, so it must not reload the vault,
// must not defer anything to commitNotes, and must raise no change events — while
// still reporting the pushed note back to the client.
func TestResolve_UnchangedPushIsInert(t *testing.T) {
	ctx := context.Background()

	env := newEnvMock(&logger.TestLogger{})
	env.InsertNoteFunc = func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
		return appmodel.NoteSaveResult{PathID: 5}, nil // content already stored
	}
	env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
		t.Error("an unchanged push must not reload the vault")
		return appmodel.NewNoteViews(), nil
	}
	env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error {
		t.Error("an unchanged push must raise no change events")
		return nil
	}
	env.InsertUncommittedPathFunc = func(_ context.Context, _ int64) error {
		t.Error("an unchanged push has no side effects to defer")
		return nil
	}
	env.LayoutsFunc = func() *appmodel.Layouts {
		return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
	}

	_, err := pushnotes.Resolve(ctx, env, model.PushNotesInput{
		Updates: []model.PushNoteInput{{Path: "note.md", Content: "same"}},
	})
	require.NoError(t, err)
}
