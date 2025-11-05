package updatetelegrampublishpost_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/updatetelegrampublishpost"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestResolve_Success(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	// Create a proper AST for the test content
	content := []byte("Updated test note content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	// Create a test note view with proper AST
	testNote := &model.NoteView{
		Path:      "/test-note",
		Title:     "Test Note",
		PathID:    123,
		VersionID: 456,
		Content:   content,
		HTML:      "<p>Updated test note content</p>",
		Permalink: "/test-note",
		Free:      true,
		RawMeta: map[string]interface{}{
			"title": "Test Note",
			"free":  true,
		},
		InLinks:       map[string]struct{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}
	testNote.SetAst(doc)

	// Create NoteViews containing the test note
	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-note": testNote,
		},
		List:      []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "latest",
	}

	mockSentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{
			ChatID:      1,
			MessageID:   101,
			TelegramID:  -1001234567890,
			ContentHash: "oldhash1",
		},
		{
			ChatID:      2,
			MessageID:   102,
			TelegramID:  -1001234567891,
			ContentHash: "oldhash2",
		},
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			require.Equal(t, notePathID, id)
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		EnqueueUpdateTelegramMessageFunc: func(ctx context.Context, params model.TelegramUpdatePostParams) error {
			// Verify params
			require.Equal(t, notePathID, params.NotePathID)
			require.Equal(t, "Updated test note content", params.Post.Content)
			require.False(t, params.Instant)
			require.False(t, params.UpdateLinkedPosts)
			return nil
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	// Verify EnqueueUpdateTelegramMessage was called twice (for each sent message)
	require.Len(t, env.EnqueueUpdateTelegramMessageCalls(), 2)
}

func TestResolve_Success_SkipUnchanged(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Updated test note content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	testNote := &model.NoteView{
		Path:    "/test-note",
		PathID:  123,
		Content: content,
		InLinks: map[string]struct{}{},
		Assets:  map[string]struct{}{},
	}
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{"/test-note": testNote},
	}

	// Hash for "Updated test note content"
	expectedHash := "79c1b725091cf8266eff024a296e4fd7ff8ce3001aa1958fe591773246f072ad"

	// Message with matching hash
	mockSentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{
			ChatID:      1,
			MessageID:   101,
			TelegramID:  -1001234567890,
			ContentHash: expectedHash, // Same hash, should skip
		},
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		EnqueueUpdateTelegramMessageFunc: func(ctx context.Context, params model.TelegramUpdatePostParams) error {
			require.Fail(t, "EnqueueUpdateTelegramMessage should not be called when hash matches")
			return nil
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	// Verify EnqueueUpdateTelegramMessage was NOT called
	require.Empty(t, env.EnqueueUpdateTelegramMessageCalls())
}

func TestResolve_Error_NoteNotFound(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(999)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{},
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "note view not found")
}

func TestResolve_Success_NoSentMessages(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Test content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	testNote := &model.NoteView{
		Path:    "/test-note",
		PathID:  123,
		Content: content,
	}
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{"/test-note": testNote},
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			return []db.ListTelegramPublishSentMessagesByNotePathIDRow{}, nil
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	// Verify EnqueueUpdateTelegramMessage was NOT called
	require.Empty(t, env.EnqueueUpdateTelegramMessageCalls())
}

func TestResolve_Error_ListSentMessages(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Test content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	testNote := &model.NoteView{
		Path:    "/test-note",
		PathID:  123,
		Content: content,
	}
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{"/test-note": testNote},
	}

	expectedErr := errors.New("database error")

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			return nil, expectedErr
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get sent messages")
}

func TestResolve_Error_ConvertPost(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Test content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	testNote := &model.NoteView{
		Path:    "/test-note",
		PathID:  123,
		Content: content,
	}
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{"/test-note": testNote},
	}

	mockSentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{
			ChatID:      1,
			MessageID:   101,
			TelegramID:  -1001234567890,
			ContentHash: "oldhash",
		},
	}

	expectedErr := errors.New("conversion error")

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return nil, expectedErr
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to convert note to telegram post")
}

func TestResolve_Error_ConversionWarnings(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Test content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	testNote := &model.NoteView{
		Path:    "/test-note",
		PathID:  123,
		Content: content,
	}
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{"/test-note": testNote},
	}

	mockSentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{
			ChatID:      1,
			MessageID:   101,
			TelegramID:  -1001234567890,
			ContentHash: "oldhash",
		},
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Test content",
				Warnings: []string{"unsupported markdown feature"},
			}, nil
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "conversion produced warnings")
}

func TestResolve_Error_EnqueueUpdate(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Updated test note content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	testNote := &model.NoteView{
		Path:    "/test-note",
		PathID:  123,
		Content: content,
	}
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{"/test-note": testNote},
	}

	mockSentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{
			ChatID:      1,
			MessageID:   101,
			TelegramID:  -1001234567890,
			ContentHash: "oldhash",
		},
	}

	expectedErr := errors.New("enqueue error")

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		EnqueueUpdateTelegramMessageFunc: func(ctx context.Context, params model.TelegramUpdatePostParams) error {
			return expectedErr
		},
	}

	err := updatetelegrampublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to enqueue update job")
}
