package updatetelegramaccountpublishpost_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/updatetelegramaccountpublishpost"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestResolve_Success(t *testing.T) {
	ctx := context.Background()
	notePathID := int64(123)

	content := []byte("Updated test note content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

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

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-note": testNote,
		},
		List:      []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "latest",
	}

	mockSentMessages := []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			MessageID:      101,
			ContentHash:    "oldhash1",
		},
		{
			AccountID:      2,
			TelegramChatID: -1001234567891,
			MessageID:      102,
			ContentHash:    "oldhash2",
		},
	}

	mockAccount := db.TelegramAccount{
		ID:          1,
		ApiID:       12345,
		ApiHash:     "testhash",
		SessionData: []byte("session"),
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			require.Equal(t, notePathID, id)
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
			return mockAccount, nil
		},
		EnqueueUpdateTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountUpdatePostParams) error {
			require.Equal(t, notePathID, params.NotePathID)
			require.Equal(t, "Updated test note content", params.Post.Content)
			require.False(t, params.Instant)
			return nil
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	require.Len(t, env.EnqueueUpdateTelegramAccountMessageCalls(), 2)
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

	mockSentMessages := []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			MessageID:      101,
			ContentHash:    expectedHash, // Same hash, should skip
		},
	}

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		EnqueueUpdateTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountUpdatePostParams) error {
			require.Fail(t, "EnqueueUpdateTelegramAccountMessage should not be called when hash matches")
			return nil
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	require.Empty(t, env.EnqueueUpdateTelegramAccountMessageCalls())
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

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
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
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{}, nil
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	require.Empty(t, env.EnqueueUpdateTelegramAccountMessageCalls())
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
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return nil, expectedErr
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get sent account messages")
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

	mockSentMessages := []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			MessageID:      101,
			ContentHash:    "oldhash",
		},
	}

	expectedErr := errors.New("conversion error")

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return nil, expectedErr
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to convert note to telegram post")
}

func TestResolve_Error_GetAccount(t *testing.T) {
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

	mockSentMessages := []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			MessageID:      101,
			ContentHash:    "oldhash",
		},
	}

	expectedErr := errors.New("account not found")

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
			return db.TelegramAccount{}, expectedErr
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get account")
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

	mockSentMessages := []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			MessageID:      101,
			ContentHash:    "oldhash",
		},
	}

	mockAccount := db.TelegramAccount{
		ID:          1,
		ApiID:       12345,
		ApiHash:     "testhash",
		SessionData: []byte("session"),
	}

	expectedErr := errors.New("enqueue error")

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
			return mockAccount, nil
		},
		EnqueueUpdateTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountUpdatePostParams) error {
			return expectedErr
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to enqueue account update job")
}

func TestResolve_AccountCaching(t *testing.T) {
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

	// Two messages from the same account
	mockSentMessages := []db.ListTelegramPublishSentAccountMessagesByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			MessageID:      101,
			ContentHash:    "oldhash1",
		},
		{
			AccountID:      1,
			TelegramChatID: -1001234567891,
			MessageID:      102,
			ContentHash:    "oldhash2",
		},
	}

	mockAccount := db.TelegramAccount{
		ID:          1,
		ApiID:       12345,
		ApiHash:     "testhash",
		SessionData: []byte("session"),
	}

	getAccountCallCount := 0

	env := &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews {
			return noteViews
		},
		ListTelegramPublishSentAccountMessagesByNotePathIDFunc: func(ctx context.Context, id int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return mockSentMessages, nil
		},
		ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
			return &model.TelegramPost{
				Content:  "Updated test note content",
				Warnings: []string{},
			}, nil
		},
		GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
			getAccountCallCount++
			return mockAccount, nil
		},
		EnqueueUpdateTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountUpdatePostParams) error {
			return nil
		},
	}

	err := updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
	require.NoError(t, err)

	// GetTelegramAccountByID should only be called once due to caching
	require.Equal(t, 1, getAccountCallCount)
	require.Len(t, env.EnqueueUpdateTelegramAccountMessageCalls(), 2)
}
