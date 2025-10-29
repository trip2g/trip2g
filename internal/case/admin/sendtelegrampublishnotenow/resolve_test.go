package sendtelegrampublishnotenow_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/admin/sendtelegrampublishnotenow"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	adminToken := &usertoken.Data{
		ID:   1,
		Role: "admin",
	}

	tests := []struct {
		name     string
		input    model.SendTelegramPublishNoteNowInput
		mockFunc func() *EnvMock
		want     model.SendTelegramPublishNoteNowOrErrorPayload
		wantErr  bool
	}{
		{
			name: "success",
			input: model.SendTelegramPublishNoteNowInput{
				ID: 1,
			},
			mockFunc: func() *EnvMock {
				mock := &EnvMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return adminToken, nil
				}
				mock.GetTelegramPublishNoteByNotePathIDFunc = func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
					require.Equal(t, int64(1), notePathID)
					return db.TelegramPublishNote{
						NotePathID: 1,
					}, nil
				}
				mock.SendTelegramPublishPostFunc = func(ctx context.Context, notePathID int64, instant bool) error {
					require.Equal(t, int64(1), notePathID)
					require.Equal(t, false, instant)
					return nil
				}
				return mock
			},
			want: &model.SendTelegramPublishNoteNowPayload{
				Success: true,
			},
			wantErr: false,
		},
		{
			name: "unauthorized - no admin token",
			input: model.SendTelegramPublishNoteNowInput{
				ID: 1,
			},
			mockFunc: func() *EnvMock {
				mock := &EnvMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "note not found",
			input: model.SendTelegramPublishNoteNowInput{
				ID: 999,
			},
			mockFunc: func() *EnvMock {
				mock := &EnvMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return adminToken, nil
				}
				mock.GetTelegramPublishNoteByNotePathIDFunc = func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
					return db.TelegramPublishNote{}, sql.ErrNoRows
				}
				return mock
			},
			want: &model.ErrorPayload{
				Message: "Telegram publish note not found",
			},
			wantErr: false,
		},
		{
			name: "database error on get note",
			input: model.SendTelegramPublishNoteNowInput{
				ID: 1,
			},
			mockFunc: func() *EnvMock {
				mock := &EnvMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return adminToken, nil
				}
				mock.GetTelegramPublishNoteByNotePathIDFunc = func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
					return db.TelegramPublishNote{}, errors.New("database error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error sending telegram post",
			input: model.SendTelegramPublishNoteNowInput{
				ID: 1,
			},
			mockFunc: func() *EnvMock {
				mock := &EnvMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return adminToken, nil
				}
				mock.GetTelegramPublishNoteByNotePathIDFunc = func(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error) {
					return db.TelegramPublishNote{
						NotePathID: 1,
					}, nil
				}
				mock.SendTelegramPublishPostFunc = func(ctx context.Context, notePathID int64, instant bool) error {
					return errors.New("telegram send error")
				}
				return mock
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := tt.mockFunc()
			got, err := sendtelegrampublishnotenow.Resolve(context.Background(), env, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				switch wantPayload := tt.want.(type) {
				case *model.SendTelegramPublishNoteNowPayload:
					gotPayload := got.(*model.SendTelegramPublishNoteNowPayload)
					require.Equal(t, wantPayload.Success, gotPayload.Success)
				case *model.ErrorPayload:
					gotPayload := got.(*model.ErrorPayload)
					require.Equal(t, wantPayload.Message, gotPayload.Message)
				}
			}
		})
	}
}
