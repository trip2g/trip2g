package handlenotewebhooks_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name           string
		changes        []handlenotewebhooks.NoteChange
		depth          int
		setupEnv       func(*mockEnv)
		wantEnqueued   int
		assertEnqueued func(*testing.T, []handlenotewebhooks.DeliverChangeWebhookParams)
	}{
		{
			name:    "empty changes returns nil",
			changes: []handlenotewebhooks.NoteChange{},
			depth:   0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["*.md"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						OnUpdate:        true,
						OnRemove:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
			},
			wantEnqueued: 0,
		},
		{
			name: "no enabled webhooks",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
			},
			wantEnqueued: 0,
		},
		{
			name: "depth filtering skips webhook at max depth",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
			},
			depth: 1,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        1, // depth >= MaxDepth → skip
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
			},
			wantEnqueued: 0,
		},
		{
			name: "depth filtering allows webhook when depth < max depth",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        1, // depth < MaxDepth → process
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Equal(t, int64(1), enqueued[0].WebhookID)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "notes/hello.md", enqueued[0].Changes[0].Path)
			},
		},
		{
			name: "event type filtering OnCreate=true matches create event",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						OnUpdate:        false,
						OnRemove:        false,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "create", enqueued[0].Changes[0].Event)
			},
		},
		{
			name: "event type filtering OnUpdate=false skips update event",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "update"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						OnUpdate:        false,
						OnRemove:        false,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
			},
			wantEnqueued: 0,
		},
		{
			name: "include pattern matching",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
				{PathID: 2, Path: "docs/api.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["notes/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
				env.addNote("docs/api.md", 2, 20, "API", "Content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "notes/hello.md", enqueued[0].Changes[0].Path)
			},
		},
		{
			name: "exclude pattern filtering",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
				{PathID: 2, Path: "notes/hello.draft", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `["**/*.draft"]`,
						OnCreate:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
				env.addNote("notes/hello.draft", 2, 20, "Draft", "Content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "notes/hello.md", enqueued[0].Changes[0].Path)
			},
		},
		{
			name: "remove event with path only",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 999, Path: "notes/deleted.md", Event: "remove"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnRemove:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				// Note is NOT in NoteViews (already deleted).
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "notes/deleted.md", enqueued[0].Changes[0].Path)
				require.Equal(t, "remove", enqueued[0].Changes[0].Event)
				require.Equal(t, int64(999), enqueued[0].Changes[0].PathID)
			},
		},
		{
			name: "multiple webhooks with different patterns",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
				{PathID: 2, Path: "docs/api.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/notes-webhook",
						IncludePatterns: `["notes/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
					{
						ID:              2,
						Url:             "https://example.com/docs-webhook",
						IncludePatterns: `["docs/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "Content")
				env.addNote("docs/api.md", 2, 20, "API", "Content")
			},
			wantEnqueued: 2,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 2)

				// First webhook should match notes/hello.md.
				require.Equal(t, int64(1), enqueued[0].WebhookID)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "notes/hello.md", enqueued[0].Changes[0].Path)

				// Second webhook should match docs/api.md.
				require.Equal(t, int64(2), enqueued[1].WebhookID)
				require.Len(t, enqueued[1].Changes, 1)
				require.Equal(t, "docs/api.md", enqueued[1].Changes[0].Path)
			},
		},
		{
			name: "include content populates content field",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        5,
						IncludeContent:  true,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "This is the content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, "This is the content", enqueued[0].Changes[0].Content)
			},
		},
		{
			name: "remove event does not include content",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 1, Path: "notes/hello.md", Event: "remove"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnRemove:        true,
						MaxDepth:        5,
						IncludeContent:  true,
					},
				})
				env.addNote("notes/hello.md", 1, 10, "Hello", "This is the content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 1)
				require.Empty(t, enqueued[0].Changes[0].Content)
			},
		},
		{
			name: "changes are sorted by path",
			changes: []handlenotewebhooks.NoteChange{
				{PathID: 3, Path: "notes/zebra.md", Event: "create"},
				{PathID: 1, Path: "notes/alpha.md", Event: "create"},
				{PathID: 2, Path: "notes/beta.md", Event: "create"},
			},
			depth: 0,
			setupEnv: func(env *mockEnv) {
				env.setWebhooks([]db.ChangeWebhook{
					{
						ID:              1,
						Url:             "https://example.com/webhook",
						IncludePatterns: `["**/*"]`,
						ExcludePatterns: `[]`,
						OnCreate:        true,
						MaxDepth:        5,
						IncludeContent:  false,
					},
				})
				env.addNote("notes/zebra.md", 3, 30, "Zebra", "Content")
				env.addNote("notes/alpha.md", 1, 10, "Alpha", "Content")
				env.addNote("notes/beta.md", 2, 20, "Beta", "Content")
			},
			wantEnqueued: 1,
			assertEnqueued: func(t *testing.T, enqueued []handlenotewebhooks.DeliverChangeWebhookParams) {
				require.Len(t, enqueued, 1)
				require.Len(t, enqueued[0].Changes, 3)
				require.Equal(t, "notes/alpha.md", enqueued[0].Changes[0].Path)
				require.Equal(t, "notes/beta.md", enqueued[0].Changes[1].Path)
				require.Equal(t, "notes/zebra.md", enqueued[0].Changes[2].Path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newMockEnv()
			tt.setupEnv(env)

			ctx := context.Background()

			err := handlenotewebhooks.Resolve(ctx, env, tt.changes, tt.depth)
			require.NoError(t, err)

			enqueued := env.getEnqueued()
			require.Len(t, enqueued, tt.wantEnqueued)

			if tt.assertEnqueued != nil {
				tt.assertEnqueued(t, enqueued)
			}
		})
	}
}

func TestMatchChange(t *testing.T) {
	tests := []struct {
		name        string
		change      handlenotewebhooks.NoteChange
		webhook     db.ChangeWebhook
		noteViews   *model.NoteViews
		wantMatch   bool
		wantPath    string
		wantContent string
	}{
		{
			name: "create event - OnCreate=false",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "create",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        false,
				OnUpdate:        true,
				OnRemove:        true,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: createNoteView("test.md", "Test", "content"),
			wantMatch: false,
		},
		{
			name: "update event - OnUpdate=false",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "update",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        true,
				OnUpdate:        false,
				OnRemove:        true,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: createNoteView("test.md", "Test", "content"),
			wantMatch: false,
		},
		{
			name: "remove event - OnRemove=false",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "remove",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        true,
				OnUpdate:        true,
				OnRemove:        false,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: createNoteView("test.md", "Test", "content"),
			wantMatch: false,
		},
		{
			name: "remove event - missing note view with path",
			change: handlenotewebhooks.NoteChange{
				PathID: 999,
				Path:   "deleted.md",
				Event:  "remove",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        false,
				OnUpdate:        false,
				OnRemove:        true,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: model.NewNoteViews(),
			wantMatch: true,
			wantPath:  "deleted.md",
		},
		{
			name: "remove event - missing note view without path",
			change: handlenotewebhooks.NoteChange{
				PathID: 999,
				Path:   "",
				Event:  "remove",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        false,
				OnUpdate:        false,
				OnRemove:        true,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: model.NewNoteViews(),
			wantMatch: false,
		},
		{
			name: "include pattern - no match",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "create",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        true,
				OnUpdate:        false,
				OnRemove:        false,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["blog/**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: createNoteView("test.md", "Test", "content"),
			wantMatch: false,
		},
		{
			name: "include pattern - match",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "blog/post.md",
				Event:  "create",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        true,
				OnUpdate:        false,
				OnRemove:        false,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["blog/**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: createNoteView("blog/post.md", "Post", "content"),
			wantMatch: true,
			wantPath:  "blog/post.md",
		},
		{
			name: "exclude pattern - match",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "draft/test.md",
				Event:  "create",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        true,
				OnUpdate:        false,
				OnRemove:        false,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `["draft/**"]`,
			},
			noteViews: createNoteView("draft/test.md", "Draft", "content"),
			wantMatch: false,
		},
		{
			name: "include content - create event",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "create",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        true,
				OnUpdate:        false,
				OnRemove:        false,
				IncludeContent:  true,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews:   createNoteView("test.md", "Test", "test content"),
			wantMatch:   true,
			wantPath:    "test.md",
			wantContent: "test content",
		},
		{
			name: "include content - update event",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "update",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        false,
				OnUpdate:        true,
				OnRemove:        false,
				IncludeContent:  true,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews:   createNoteView("test.md", "Test", "updated content"),
			wantMatch:   true,
			wantPath:    "test.md",
			wantContent: "updated content",
		},
		{
			name: "include content - remove event (content not included)",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "test.md",
				Event:  "remove",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        false,
				OnUpdate:        false,
				OnRemove:        true,
				IncludeContent:  true,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews:   createNoteView("test.md", "Test", "removed content"),
			wantMatch:   true,
			wantPath:    "test.md",
			wantContent: "",
		},
		{
			name: "note view provides path when available",
			change: handlenotewebhooks.NoteChange{
				PathID: 1,
				Path:   "old-path.md",
				Event:  "update",
			},
			webhook: db.ChangeWebhook{
				ID:              1,
				OnCreate:        false,
				OnUpdate:        true,
				OnRemove:        false,
				IncludeContent:  false,
				MaxDepth:        10,
				IncludePatterns: `["**"]`,
				ExcludePatterns: `[]`,
			},
			noteViews: createNoteView("new-path.md", "Test", "content"),
			wantMatch: true,
			wantPath:  "new-path.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test matchChange indirectly through Resolve.
			// matchChange is unexported, so we verify its behavior through integration.
			env := newMockEnv()
			env.setWebhooks([]db.ChangeWebhook{tt.webhook})
			env.noteViews = tt.noteViews

			err := handlenotewebhooks.Resolve(context.Background(), env, []handlenotewebhooks.NoteChange{tt.change}, 0)
			require.NoError(t, err)

			enqueued := env.getEnqueued()

			if tt.wantMatch {
				require.Len(t, enqueued, 1, "expected webhook to match and enqueue")
				require.Len(t, enqueued[0].Changes, 1)
				require.Equal(t, tt.wantPath, enqueued[0].Changes[0].Path)
				require.Equal(t, tt.change.Event, enqueued[0].Changes[0].Event)
				require.Equal(t, tt.wantContent, enqueued[0].Changes[0].Content)
			} else {
				require.Empty(t, enqueued, "expected webhook not to match")
			}
		})
	}
}

// createNoteView creates a NoteViews with a single note.
func createNoteView(path string, title string, content string) *model.NoteViews {
	nvs := model.NewNoteViews()
	note := &model.NoteView{
		PathID:    1,
		VersionID: 10,
		Path:      path,
		Title:     title,
		Content:   []byte(content),
	}
	nvs.PathMap[path] = note
	nvs.Map[path] = note
	return nvs
}
