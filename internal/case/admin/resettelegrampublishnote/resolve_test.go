package resettelegrampublishnote_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/resettelegrampublishnote"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

func assertResetPayloadEquals(
	t *testing.T,
	got, want *model.ResetTelegramPublishNotePayload,
) {
	t.Helper()
	require.Equal(t, want.PublishNote.NotePathID, got.PublishNote.NotePathID)
	require.Equal(t, want.PublishNote.PublishedAt.Valid, got.PublishNote.PublishedAt.Valid)
	require.Equal(
		t,
		want.PublishNote.PublishedVersionID.Valid,
		got.PublishNote.PublishedVersionID.Valid,
	)
}

func assertErrorPayloadEquals(t *testing.T, got, want *model.ErrorPayload) {
	t.Helper()
	require.Equal(t, want.Message, got.Message)
}

func assertPayloadMatches(t *testing.T, got, want resettelegrampublishnote.Payload) {
	t.Helper()

	if want == nil {
		require.Nil(t, got)
		return
	}

	require.NotNil(t, got)

	switch wantPayload := want.(type) {
	case *model.ResetTelegramPublishNotePayload:
		gotPayload, ok := got.(*model.ResetTelegramPublishNotePayload)
		require.True(t, ok, "expected *model.ResetTelegramPublishNotePayload, got %T", got)
		assertResetPayloadEquals(t, gotPayload, wantPayload)
	case *model.ErrorPayload:
		gotPayload, ok := got.(*model.ErrorPayload)
		require.True(t, ok, "expected *model.ErrorPayload, got %T", got)
		assertErrorPayloadEquals(t, gotPayload, wantPayload)
	default:
		t.Fatalf("unexpected payload type: %T", want)
	}
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	validToken := &usertoken.Data{ID: 1, Role: "admin"}
	mockLogger := &logger.TestLogger{}

	publishTime := time.Now().Add(-2 * time.Hour)
	publishedTime := time.Now().Add(-1 * time.Hour)

	validPublishNote := db.TelegramPublishNote{
		NotePathID:         123,
		CreatedAt:          time.Now().Add(-3 * time.Hour),
		PublishAt:          publishTime,
		PublishedAt:        sql.NullTime{Time: publishedTime, Valid: true},
		PublishedVersionID: sql.NullInt64{Int64: 456, Valid: true},
	}

	resetPublishNote := db.TelegramPublishNote{
		NotePathID:         123,
		CreatedAt:          time.Now().Add(-3 * time.Hour),
		PublishAt:          publishTime,
		PublishedAt:        sql.NullTime{},
		PublishedVersionID: sql.NullInt64{},
	}

	sentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{ChatID: 1, MessageID: 100, TelegramID: 1001},
		{ChatID: 2, MessageID: 200, TelegramID: 1002},
	}

	// Helper to add default account message mocks
	addAccountMocks := func(env *EnvMock) *EnvMock {
		env.ListTelegramPublishSentAccountMessagesByNotePathIDFunc = func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error) {
			return nil, nil
		}
		env.DeleteTelegramPublishSentAccountMessagesByNotePathIDFunc = func(ctx context.Context, notePathID int64) error {
			return nil
		}
		env.GetTelegramAccountByIDFunc = func(ctx context.Context, id int64) (db.TelegramAccount, error) {
			return db.TelegramAccount{}, nil
		}
		env.DeleteTelegramAccountMessageFunc = func(ctx context.Context, account db.TelegramAccount, chatID, messageID int64) error {
			return nil
		}
		return env
	}

	tests := []struct {
		name    string
		input   model.ResetTelegramPublishNoteInput
		env     func() *EnvMock
		want    resettelegrampublishnote.Payload
		wantErr bool
	}{
		{
			name:  "successful reset",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				callCount := 0
				return addAccountMocks(&EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						callCount++
						if callCount == 1 {
							return validPublishNote, nil // First call - before reset
						}
						return resetPublishNote, nil // Second call - after reset
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return sentMessages, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						return nil
					},
					ResetTelegramPublishNoteFunc: func(ctx context.Context, notePathID int64) error {
						return nil
					},
					DeleteTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) error {
						return nil
					},
				})
			},
			want: &model.ResetTelegramPublishNotePayload{
				PublishNote: &resetPublishNote,
			},
			wantErr: false,
		},
		{
			name:  "unauthorized user",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				return &EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return nil, errors.New("unauthorized")
					},
				}
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "publish note not found",
			input: model.ResetTelegramPublishNoteInput{ID: 999},
			env: func() *EnvMock {
				return &EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						return db.TelegramPublishNote{}, sql.ErrNoRows
					},
				}
			},
			want: &model.ErrorPayload{
				Message: "Telegram publish note not found",
			},
			wantErr: false,
		},
		{
			name:  "database error on getting publish note",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				return &EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						return db.TelegramPublishNote{}, errors.New("database error")
					},
				}
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "error listing sent messages",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				return &EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						return validPublishNote, nil
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return nil, errors.New("database error")
					},
				}
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "successful reset with telegram deletion errors (should not fail)",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				callCount := 0
				return addAccountMocks(&EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						callCount++
						if callCount == 1 {
							return validPublishNote, nil // First call - before reset
						}
						return resetPublishNote, nil // Second call - after reset
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return sentMessages, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						// Simulate telegram API errors (should not fail the whole operation)
						return errors.New("telegram API error")
					},
					ResetTelegramPublishNoteFunc: func(ctx context.Context, notePathID int64) error {
						return nil
					},
					DeleteTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) error {
						return nil
					},
				})
			},
			want: &model.ResetTelegramPublishNotePayload{
				PublishNote: &resetPublishNote,
			},
			wantErr: false,
		},
		{
			name:  "error resetting publish note",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				return addAccountMocks(&EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						return validPublishNote, nil
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return sentMessages, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						return nil
					},
					ResetTelegramPublishNoteFunc: func(ctx context.Context, notePathID int64) error {
						return errors.New("database error")
					},
				})
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "error deleting sent message records",
			input: model.ResetTelegramPublishNoteInput{ID: 123},
			env: func() *EnvMock {
				return addAccountMocks(&EnvMock{
					CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
						return validToken, nil
					},
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					GetTelegramPublishNoteByNotePathIDFunc: func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
						return validPublishNote, nil
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return sentMessages, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						return nil
					},
					ResetTelegramPublishNoteFunc: func(ctx context.Context, notePathID int64) error {
						return nil
					},
					DeleteTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) error {
						return errors.New("database error")
					},
				})
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env()
			got, err := resettelegrampublishnote.Resolve(ctx, env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assertPayloadMatches(t, got, tt.want)
		})
	}
}
