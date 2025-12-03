package settelegramaccountchatpublishtags_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/settelegramaccountchatpublishtags"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg settelegramaccountchatpublishtags_test . Env

type Env = settelegramaccountchatpublishtags.Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		env         Env
		input       model.AdminSetTelegramAccountChatPublishTagsInput
		wantErr     bool
		wantPayload bool
		errContains string
	}{
		{
			name: "successful set tags",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteTelegramPublishAccountChatsByAccountAndChatIDFunc: func(ctx context.Context, arg db.DeleteTelegramPublishAccountChatsByAccountAndChatIDParams) error {
					return nil
				},
				InsertTelegramPublishAccountChatFunc: func(ctx context.Context, arg db.InsertTelegramPublishAccountChatParams) error {
					return nil
				},
			},
			input: model.AdminSetTelegramAccountChatPublishTagsInput{
				AccountID:      1,
				TelegramChatID: "123456789",
				TagIds:         []int64{1, 2, 3},
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "empty tags - just deletes",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteTelegramPublishAccountChatsByAccountAndChatIDFunc: func(ctx context.Context, arg db.DeleteTelegramPublishAccountChatsByAccountAndChatIDParams) error {
					return nil
				},
			},
			input: model.AdminSetTelegramAccountChatPublishTagsInput{
				AccountID:      1,
				TelegramChatID: "123456789",
				TagIds:         []int64{},
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "invalid chat id",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			input: model.AdminSetTelegramAccountChatPublishTagsInput{
				AccountID:      1,
				TelegramChatID: "invalid",
				TagIds:         []int64{1},
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "Invalid telegram chat ID",
		},
		{
			name: "admin token error",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			input: model.AdminSetTelegramAccountChatPublishTagsInput{
				AccountID:      1,
				TelegramChatID: "123456789",
				TagIds:         []int64{1},
			},
			wantErr:     true,
			wantPayload: false,
		},
		{
			name: "delete error",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteTelegramPublishAccountChatsByAccountAndChatIDFunc: func(ctx context.Context, arg db.DeleteTelegramPublishAccountChatsByAccountAndChatIDParams) error {
					return errors.New("delete failed")
				},
			},
			input: model.AdminSetTelegramAccountChatPublishTagsInput{
				AccountID:      1,
				TelegramChatID: "123456789",
				TagIds:         []int64{1},
			},
			wantErr:     true,
			wantPayload: false,
		},
		{
			name: "insert error",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteTelegramPublishAccountChatsByAccountAndChatIDFunc: func(ctx context.Context, arg db.DeleteTelegramPublishAccountChatsByAccountAndChatIDParams) error {
					return nil
				},
				InsertTelegramPublishAccountChatFunc: func(ctx context.Context, arg db.InsertTelegramPublishAccountChatParams) error {
					return errors.New("insert failed")
				},
			},
			input: model.AdminSetTelegramAccountChatPublishTagsInput{
				AccountID:      1,
				TelegramChatID: "123456789",
				TagIds:         []int64{1},
			},
			wantErr:     true,
			wantPayload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := settelegramaccountchatpublishtags.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPayload {
				p, ok := payload.(*model.AdminSetTelegramAccountChatPublishTagsPayload)
				require.True(t, ok, "expected AdminSetTelegramAccountChatPublishTagsPayload")
				require.True(t, p.Success)
			} else {
				errPayload, ok := payload.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", payload)
				require.Contains(t, errPayload.Message, tt.errContains)
			}
		})
	}
}
