package gettelegramcustomemojies_test

import (
	"context"
	"testing"

	"trip2g/internal/case/gettelegramcustomemojies"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/tgbots"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg gettelegramcustomemojies_test . Env TgBotsInterface

func TestResolve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  model.TelegramCustomEmojiesFilter
		setup   func(*EnvMock, *TgBotsInterfaceMock)
		want    []model.TelegramCustomEmoji
		wantErr bool
	}{
		{
			name: "empty ids returns empty result",
			filter: model.TelegramCustomEmojiesFilter{
				Ids: []string{},
			},
			setup:   func(env *EnvMock, tgBots *TgBotsInterfaceMock) {},
			want:    []model.TelegramCustomEmoji{},
			wantErr: false,
		},
		{
			name: "returns cached emojies when all are in database",
			filter: model.TelegramCustomEmojiesFilter{
				Ids: []string{"emoji1", "emoji2"},
			},
			setup: func(env *EnvMock, tgBots *TgBotsInterfaceMock) {
				env.ListTelegramCustomEmojiesFunc = func(ctx context.Context, ids []string) ([]db.TelegramCustomEmojy, error) {
					return []db.TelegramCustomEmojy{
						{ID: "emoji1", Base64Data: "data:image/webp;base64,abc123"},
						{ID: "emoji2", Base64Data: "data:image/webp;base64,def456"},
					}, nil
				}
			},
			want: []model.TelegramCustomEmoji{
				{ID: "emoji1", Base64Uri: "data:image/webp;base64,abc123"},
				{ID: "emoji2", Base64Uri: "data:image/webp;base64,def456"},
			},
			wantErr: false,
		},
		{
			name: "fetches missing emojies from telegram when not in cache",
			filter: model.TelegramCustomEmojiesFilter{
				Ids: []string{"emoji1", "emoji2"},
			},
			setup: func(env *EnvMock, tgBots *TgBotsInterfaceMock) {
				env.ListTelegramCustomEmojiesFunc = func(ctx context.Context, ids []string) ([]db.TelegramCustomEmojy, error) {
					return []db.TelegramCustomEmojy{
						{ID: "emoji1", Base64Data: "data:image/webp;base64,abc123"},
					}, nil
				}
				env.GetTgBotsFunc = func() gettelegramcustomemojies.TgBotsInterface {
					return tgBots
				}
				env.InsertTelegramCustomEmojiFunc = func(ctx context.Context, arg db.InsertTelegramCustomEmojiParams) error {
					return nil
				}

				tgBots.GetBotIDsFunc = func() []int64 {
					return []int64{1}
				}
				tgBots.GetHandlerIOFunc = func(botID int64) *tgbots.HandlerIO {
					return &tgbots.HandlerIO{}
				}
			},
			want: []model.TelegramCustomEmoji{
				{ID: "emoji1", Base64Uri: "data:image/webp;base64,abc123"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tgBots := &TgBotsInterfaceMock{}

			if tt.setup != nil {
				tt.setup(env, tgBots)
			}

			got, err := gettelegramcustomemojies.Resolve(ctx, env, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("Resolve() got %d results, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].ID != tt.want[i].ID {
					t.Errorf("Resolve() got[%d].ID = %v, want %v", i, got[i].ID, tt.want[i].ID)
				}
				if got[i].Base64Uri != tt.want[i].Base64Uri {
					t.Errorf("Resolve() got[%d].Base64Uri = %v, want %v", i, got[i].Base64Uri, tt.want[i].Base64Uri)
				}
			}
		})
	}
}
