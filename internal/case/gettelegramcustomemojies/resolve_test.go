package gettelegramcustomemojies_test

import (
	"context"
	"testing"

	"trip2g/internal/case/gettelegramcustomemojies"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func TestResolve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  model.TelegramCustomEmojiesFilter
		setup   func(*EnvMock)
		want    []model.TelegramCustomEmoji
		wantErr bool
	}{
		{
			name: "empty ids returns empty result",
			filter: model.TelegramCustomEmojiesFilter{
				Ids: []string{},
			},
			setup:   func(env *EnvMock) {},
			want:    []model.TelegramCustomEmoji{},
			wantErr: false,
		},
		{
			name: "returns cached emojies when all are in database",
			filter: model.TelegramCustomEmojiesFilter{
				Ids: []string{"emoji1", "emoji2"},
			},
			setup: func(env *EnvMock) {
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
			setup: func(env *EnvMock) {
				env.ListTelegramCustomEmojiesFunc = func(ctx context.Context, ids []string) ([]db.TelegramCustomEmojy, error) {
					return []db.TelegramCustomEmojy{
						{ID: "emoji1", Base64Data: "data:image/webp;base64,abc123"},
					}, nil
				}
				env.GetTelegramCustomEmojiStickersFunc = func(ctx context.Context, emojiIDs []string) ([]appmodel.CustomEmojiSticker, error) {
					return []appmodel.CustomEmojiSticker{
						{ID: "emoji2", Base64Data: "data:video/webm;base64,xyz789"},
					}, nil
				}
				env.InsertTelegramCustomEmojiFunc = func(ctx context.Context, arg db.InsertTelegramCustomEmojiParams) error {
					return nil
				}
			},
			want: []model.TelegramCustomEmoji{
				{ID: "emoji1", Base64Uri: "data:image/webp;base64,abc123"},
				{ID: "emoji2", Base64Uri: "data:video/webm;base64,xyz789"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}

			if tt.setup != nil {
				tt.setup(env)
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
