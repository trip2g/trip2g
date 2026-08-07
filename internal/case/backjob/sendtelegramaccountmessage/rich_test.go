package sendtelegramaccountmessage_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"trip2g/internal/case/backjob/sendtelegramaccountmessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

// errStopBeforeSend cuts a test off at the session boundary: everything past it
// needs a live MTProto connection.
var errStopBeforeSend = errors.New("stop before send")

func richPost() model.TelegramPost {
	return model.TelegramPost{
		Content: "classic fallback",
		RichBlocks: []tgrich.Block{
			tgrich.Heading(1, tgrich.RichText{Text: "Title"}),
			tgrich.Paragraph(tgrich.RichText{Text: "Body"}),
		},
	}
}

func appConfig(t *testing.T, mode string) string {
	t.Helper()

	raw, err := json.Marshal(map[string]interface{}{tgrich.KeyRichMessagePosting: mode})
	require.NoError(t, err)

	return string(raw)
}

// Premium is enforced server-side: the call itself is refused with
// RICH_MESSAGE_UNSUPPORTED. Checking the stored capability first turns that into
// a named reason, and the session must never be opened for a send that cannot
// succeed — opening one costs a connection and buries the reason.
func TestRichSendRefusedWithoutCapability(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		isPremium int64
		wantIn    string
	}{
		{
			name:      "premium mode without premium",
			mode:      tgrich.PostingPremium,
			isPremium: 0,
			wantIn:    "Premium",
		},
		{
			name:      "disabled refuses even a premium account",
			mode:      tgrich.PostingDisabled,
			isPremium: 1,
			wantIn:    "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastError string

			env := &EnvMock{
				CheckTelegramPublishSentAccountMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentAccountMessageExistsParams) (int64, error) {
					return 0, nil
				},
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{
						ID:        id,
						IsPremium: tt.isPremium,
						AppConfig: appConfig(t, tt.mode),
					}, nil
				},
				DecryptDataFunc: func(ciphertext []byte) ([]byte, error) {
					t.Error("no account session may be decrypted for a refused rich send")
					return nil, nil
				},
				SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
					if arg.LastError != nil {
						lastError = *arg.LastError
					}
					return nil
				},
				LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
			}

			err := sendtelegramaccountmessage.Resolve(context.Background(), env, model.TelegramAccountSendPostParams{
				NotePathID: 1,
				AccountID:  7,
				Post:       richPost(),
			})

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantIn)

			// The reason has to reach the admin, not just the job log.
			require.Contains(t, lastError, tt.wantIn)
		})
	}
}

// A capable account must get past the preflight and reach the transport, which
// is where a unit test stops: the send itself needs a live MTProto session.
func TestRichSendWithCapabilityReachesTransport(t *testing.T) {
	decrypted := false

	env := &EnvMock{
		CheckTelegramPublishSentAccountMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentAccountMessageExistsParams) (int64, error) {
			return 0, nil
		},
		GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
			return db.TelegramAccount{
				ID:        id,
				IsPremium: 1,
				AppConfig: appConfig(t, tgrich.PostingPremium),
			}, nil
		},
		DecryptDataFunc: func(ciphertext []byte) ([]byte, error) {
			decrypted = true
			return nil, errStopBeforeSend
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
		LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
	}

	err := sendtelegramaccountmessage.Resolve(context.Background(), env, model.TelegramAccountSendPostParams{
		NotePathID: 1,
		AccountID:  7,
		Post:       richPost(),
	})

	require.Error(t, err)
	require.True(t, decrypted, "a capable account must proceed to open its session")
}

// A rich post is stored as 'rich' so the account edit guard has something to act
// on: that guard refuses to cross formats, and a post typed 'text' would let a
// classic edit flatten a published rich message.
func TestRichPostTypeIsRich(t *testing.T) {
	require.Equal(t,
		db.TelegramPublishSentMessagePostTypeRich,
		db.TelegramPublishSentMessagePostTypeFor(richPost().IsRich(), 0),
	)
}
