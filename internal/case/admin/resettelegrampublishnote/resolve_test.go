package resettelegrampublishnote_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"trip2g/internal/case/admin/resettelegrampublishnote"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

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
				return &EnvMock{
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
				}
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
				return &EnvMock{
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
				}
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
						return sentMessages, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						return nil
					},
					ResetTelegramPublishNoteFunc: func(ctx context.Context, notePathID int64) error {
						return errors.New("database error")
					},
				}
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "error deleting sent message records",
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
				}
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env()
			got, err := resettelegrampublishnote.Resolve(ctx, env, tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got == nil && tt.want != nil {
				t.Errorf("Resolve() got = nil, want = %v", tt.want)
				return
			}

			if tt.want == nil && got != nil {
				t.Errorf("Resolve() got = %v, want = nil", got)
				return
			}

			switch gotPayload := got.(type) {
			case *model.ResetTelegramPublishNotePayload:
				wantPayload := tt.want.(*model.ResetTelegramPublishNotePayload)
				if gotPayload.PublishNote.NotePathID != wantPayload.PublishNote.NotePathID {
					t.Errorf("Resolve() got NotePathID = %v, want NotePathID = %v", gotPayload.PublishNote.NotePathID, wantPayload.PublishNote.NotePathID)
				}
				if gotPayload.PublishNote.PublishedAt.Valid != wantPayload.PublishNote.PublishedAt.Valid {
					t.Errorf("Resolve() got PublishedAt.Valid = %v, want PublishedAt.Valid = %v", gotPayload.PublishNote.PublishedAt.Valid, wantPayload.PublishNote.PublishedAt.Valid)
				}
				if gotPayload.PublishNote.PublishedVersionID.Valid != wantPayload.PublishNote.PublishedVersionID.Valid {
					t.Errorf("Resolve() got PublishedVersionID.Valid = %v, want PublishedVersionID.Valid = %v", gotPayload.PublishNote.PublishedVersionID.Valid, wantPayload.PublishNote.PublishedVersionID.Valid)
				}
			case *model.ErrorPayload:
				wantPayload := tt.want.(*model.ErrorPayload)
				if gotPayload.Message != wantPayload.Message {
					t.Errorf("Resolve() got Message = %v, want Message = %v", gotPayload.Message, wantPayload.Message)
				}
			default:
				t.Errorf("Resolve() got unexpected type = %T", got)
			}
		})
	}
}
