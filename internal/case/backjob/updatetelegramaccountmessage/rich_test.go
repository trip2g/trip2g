package updatetelegramaccountmessage_test

import (
	"context"
	"testing"

	"trip2g/internal/case/backjob/updatetelegramaccountmessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// Accounts cannot create a rich message today, so this guard exists for the day
// they can. It must refuse rather than guess: every account edit branch is a
// classic one, and each of them flattens a rich message irreversibly.
func TestResolve_Rich_AccountPathRefuses(t *testing.T) {
	tests := []struct {
		name  string
		media []string
	}{
		{name: "no media"},
		{name: "with media", media: []string{"https://example.com/a.jpg"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
				GetTelegramPublishSentAccountMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessagePostTypeParams) (string, error) {
					return db.TelegramPublishSentMessagePostTypeRich, nil
				},
				GetTelegramPublishSentAccountMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error) {
					t.Error("the refusal must come before any further work")
					return "", nil
				},
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					t.Error("no account session may be opened for a refused edit")
					return db.TelegramAccount{}, nil
				},
				UpdateTelegramPublishSentAccountMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentAccountMessageContentParams) error {
					t.Error("a refused edit must not demote the stored post type")
					return nil
				},
				TelegramCaptionLengthLimitFunc: func(ctx context.Context, accountID *int64) int { return 1024 },
			}

			params := model.TelegramAccountUpdatePostParams{
				TelegramAccountSendPostParams: model.TelegramAccountSendPostParams{
					NotePathID:     123,
					AccountID:      7,
					TelegramChatID: -1004487679938,
					Post: model.TelegramPost{
						Content: "classic fallback text",
						Media:   tt.media,
					},
				},
				MessageID: 104,
			}

			err := updatetelegramaccountmessage.Resolve(context.Background(), env, params)
			require.Error(t, err)
			require.Contains(t, err.Error(), "rich")

			require.Empty(t, env.UpdateTelegramPublishSentAccountMessageContentCalls())
			require.Empty(t, env.GetTelegramAccountByIDCalls())
		})
	}
}
