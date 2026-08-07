package updatetelegrammessage_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"trip2g/internal/case/backjob/updatetelegrammessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

// recordingLogger keeps every line so a test can assert that a warning is
// absent. The spurious "cannot change post type" line reaches nothing but the
// log, so the log is the only place it can be observed.
type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) record(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
}

func (l *recordingLogger) Info(msg string, _ ...interface{})  { l.record(msg) }
func (l *recordingLogger) Error(msg string, _ ...interface{}) { l.record(msg) }
func (l *recordingLogger) Debug(msg string, _ ...interface{}) { l.record(msg) }
func (l *recordingLogger) Warn(msg string, _ ...interface{})  { l.record(msg) }

func (l *recordingLogger) contains(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range l.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}

	return false
}

func richBlocks() []tgrich.Block {
	return []tgrich.Block{
		tgrich.Heading(2, tgrich.RichText{Text: "Title"}),
		tgrich.Paragraph(tgrich.RichText{Text: "Edited body"}),
	}
}

func richUpdateParams() model.TelegramUpdatePostParams {
	return model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: -1004487679938,
			Post: model.TelegramPost{
				Content:    "classic fallback text",
				RichBlocks: richBlocks(),
			},
		},
		MessageID: 104,
	}
}

// richUpdateEnv fails the test on any classic edit. Both classic branches
// destroy a rich message: editMessageText with ParseMode HTML flattens it, and
// editMessageCaption targets a caption a rich message does not have.
func richUpdateEnv(t *testing.T, storedType string, log logger.Logger) *EnvMock {
	t.Helper()

	return &EnvMock{
		LoggerFunc: func() logger.Logger { return log },
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return storedType, nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			return "old_hash", nil
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			t.Errorf("a rich post must never reach a classic edit, got %T", msg)
			return nil
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			return nil
		},
		TelegramCaptionLengthLimitFunc: func(ctx context.Context, accountID *int64) int { return 1024 },
	}
}

func TestResolve_Rich_EditsThroughRichTransport(t *testing.T) {
	log := &recordingLogger{}
	env := richUpdateEnv(t, db.TelegramPublishSentMessagePostTypeRich, log)
	env.EditTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.EditRequest) (tgrich.SendResult, error) {
		require.Equal(t, int64(456), chatID)
		require.Equal(t, int64(-1004487679938), req.ChatID)
		require.Equal(t, int64(104), req.MessageID)
		require.Len(t, req.RichMessage.Blocks, 2)
		require.True(t, req.RichMessage.SkipEntityDetection, "entity auto-detection must stay off")
		require.Empty(t, req.RichMessage.Markdown)
		require.Empty(t, req.RichMessage.HTML)

		return tgrich.SendResult{MessageID: 104}, nil
	}

	require.NoError(t, updatetelegrammessage.Resolve(context.Background(), env, richUpdateParams()))

	require.Len(t, env.EditTelegramRichMessageCalls(), 1)
	require.Empty(t, env.SendTelegramRequestCalls(), "no plain-text and no caption edit may be issued")
	require.Len(t, env.UpdateTelegramPublishSentMessageContentCalls(), 1)
}

// The stored post type is what the next edit dispatches on, so it must still
// read "rich" afterwards. The update statement carries no post_type at all,
// which is what keeps it rich; a regression that starts writing one here would
// be caught by the following edit still routing to the rich transport.
func TestResolve_Rich_SurvivesAcrossConsecutiveEdits(t *testing.T) {
	log := &recordingLogger{}
	stored := db.TelegramPublishSentMessagePostTypeRich

	env := richUpdateEnv(t, stored, log)
	env.GetTelegramPublishSentMessagePostTypeFunc = func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
		return stored, nil
	}
	env.UpdateTelegramPublishSentMessageContentFunc = func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
		// Whatever this write does, it must not be able to demote the row.
		require.NotContains(t, arg.Content, "post_type")
		return nil
	}
	env.EditTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.EditRequest) (tgrich.SendResult, error) {
		return tgrich.SendResult{MessageID: 104}, nil
	}

	first := richUpdateParams()
	require.NoError(t, updatetelegrammessage.Resolve(context.Background(), env, first))

	second := richUpdateParams()
	second.Post.Content = "classic fallback text, edited again"
	second.Post.RichBlocks = append(richBlocks(), tgrich.Paragraph(tgrich.RichText{Text: "More"}))
	require.NoError(t, updatetelegrammessage.Resolve(context.Background(), env, second))

	require.Equal(t, db.TelegramPublishSentMessagePostTypeRich, stored, "stored post type must remain rich")
	require.Len(t, env.EditTelegramRichMessageCalls(), 2)
	require.Empty(t, env.SendTelegramRequestCalls())
}

// A rich post that is still rich has not changed type; the warning that said
// otherwise fired on every single edit.
func TestResolve_Rich_NoSpuriousPostTypeChangeWarning(t *testing.T) {
	log := &recordingLogger{}
	env := richUpdateEnv(t, db.TelegramPublishSentMessagePostTypeRich, log)
	env.EditTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.EditRequest) (tgrich.SendResult, error) {
		return tgrich.SendResult{MessageID: 104}, nil
	}

	params := richUpdateParams()
	require.NoError(t, updatetelegrammessage.Resolve(context.Background(), env, params))

	require.False(t, log.contains("post type change"), "a rich post that is still rich has not changed type")
}

// Dropping `telegram_rich: on` from a published note must not rewrite the
// message as plain text. There is no way back, so the edit is refused.
func TestResolve_Rich_RefusesDowngradeToClassic(t *testing.T) {
	log := &recordingLogger{}
	env := richUpdateEnv(t, db.TelegramPublishSentMessagePostTypeRich, log)
	env.EditTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.EditRequest) (tgrich.SendResult, error) {
		t.Error("a post with no blocks cannot be sent as a rich edit")
		return tgrich.SendResult{}, nil
	}

	params := richUpdateParams()
	params.Post.RichBlocks = nil

	require.NoError(t, updatetelegrammessage.Resolve(context.Background(), env, params))

	require.Empty(t, env.SendTelegramRequestCalls(), "the published blocks must be left alone")
	require.Empty(t, env.EditTelegramRichMessageCalls())
	require.Empty(t, env.UpdateTelegramPublishSentMessageContentCalls(),
		"the stored content must keep describing what is actually posted")
	require.True(t, log.contains("rich"), "the refusal must be visible in the log")
}

// The classic branches must be untouched by the rich dispatch.
func TestResolve_Classic_NeverUsesRichTransport(t *testing.T) {
	tests := []struct {
		name       string
		storedType string
		media      []string
		wantEdit   interface{}
	}{
		{
			name:       "text",
			storedType: db.TelegramPublishSentMessagePostTypeText,
			wantEdit:   tgbotapi.EditMessageTextConfig{},
		},
		{
			name:       "photo",
			storedType: db.TelegramPublishSentMessagePostTypePhoto,
			media:      []string{"https://example.com/a.jpg"},
			wantEdit:   tgbotapi.EditMessageCaptionConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
				GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
					return tt.storedType, nil
				},
				GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
					return "old_hash", nil
				},
				TelegramCaptionLengthLimitFunc: func(ctx context.Context, accountID *int64) int { return 1024 },
				SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					require.IsType(t, tt.wantEdit, msg)
					return nil
				},
				UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
					return nil
				},
				EditTelegramRichMessageFunc: func(ctx context.Context, chatID int64, req tgrich.EditRequest) (tgrich.SendResult, error) {
					t.Error("a classic post must not use the rich transport")
					return tgrich.SendResult{}, nil
				},
			}

			params := richUpdateParams()
			params.Post.RichBlocks = nil
			params.Post.Media = tt.media

			require.NoError(t, updatetelegrammessage.Resolve(context.Background(), env, params))
			require.Len(t, env.SendTelegramRequestCalls(), 1)
			require.Empty(t, env.EditTelegramRichMessageCalls())
		})
	}
}
