package sendtelegrammessage_test

import (
	"context"
	"testing"
	"trip2g/internal/case/backjob/sendtelegrammessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

func richParams() model.TelegramSendPostParams {
	return model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: -1004487679938,
		Post: model.TelegramPost{
			Content: "<b>classic fallback</b>",
			RichBlocks: []tgrich.Block{
				tgrich.Heading(2, tgrich.RichText{Text: "Title"}, "title"),
				tgrich.Paragraph(tgrich.RichText{Text: "Body"}),
			},
		},
	}
}

func richEnvMock() *EnvMock {
	return &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			return nil
		},
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
		LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			t := &testing.T{}
			t.Error("classic send must not be used for a rich post")
			return 0, nil
		},
	}
}

func TestResolve_Rich_SendsRichMessage(t *testing.T) {
	params := richParams()

	env := richEnvMock()
	env.SendTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.Request) (tgrich.SendResult, error) {
		require.Equal(t, int64(456), chatID)
		require.Equal(t, int64(-1004487679938), req.ChatID)
		require.Len(t, req.RichMessage.Blocks, 2)
		require.True(t, req.RichMessage.SkipEntityDetection, "entity auto-detection must be off")
		require.Empty(t, req.RichMessage.Markdown)
		require.Empty(t, req.RichMessage.HTML)

		return tgrich.SendResult{MessageID: 999}, nil
	}

	require.NoError(t, sendtelegrammessage.Resolve(context.Background(), env, params))

	require.Len(t, env.SendTelegramRichMessageCalls(), 1)
	require.Empty(t, env.SendTelegramMessageCalls())

	inserts := env.InsertTelegramPublishSentMessageCalls()
	require.Len(t, inserts, 1)
	require.Equal(t, int64(999), inserts[0].Arg.MessageID)
	require.NotEmpty(t, inserts[0].Arg.ContentHash)
}

// Local validation must run before the request leaves: several server-side
// violations answer ok:true and discard content silently.
func TestResolve_Rich_InvalidBlocksNeverSent(t *testing.T) {
	params := richParams()
	params.Post.RichBlocks = []tgrich.Block{{Type: tgrich.BlockHeading, Size: 9, Text: &tgrich.RichText{Text: "too deep"}}}

	env := richEnvMock()
	env.SendTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.Request) (tgrich.SendResult, error) {
		return tgrich.SendResult{MessageID: 1}, nil
	}

	err := sendtelegrammessage.Resolve(context.Background(), env, params)
	require.Error(t, err)

	require.Empty(t, env.SendTelegramRichMessageCalls())
	require.Empty(t, env.InsertTelegramPublishSentMessageCalls())
	require.Len(t, env.SetTelegramPublishNoteLastErrorCalls(), 1)
}

// A classic post must keep going through the classic transport untouched.
func TestResolve_Classic_DoesNotUseRichTransport(t *testing.T) {
	params := richParams()
	params.Post.RichBlocks = nil

	env := richEnvMock()
	env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
		return 111, nil
	}
	env.SendTelegramRichMessageFunc = func(ctx context.Context, chatID int64, req tgrich.Request) (tgrich.SendResult, error) {
		t.Error("rich transport must not be used for a classic post")
		return tgrich.SendResult{}, nil
	}

	require.NoError(t, sendtelegrammessage.Resolve(context.Background(), env, params))

	require.Len(t, env.SendTelegramMessageCalls(), 1)
	require.Empty(t, env.SendTelegramRichMessageCalls())
}
