package model_test

import (
	"testing"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestExtractTelegramRichMode(t *testing.T) {
	tests := []struct {
		name    string
		raw     map[string]interface{}
		want    model.TelegramRichMode
		wantErr bool
	}{
		{
			name: "absent defaults to auto",
			raw:  map[string]interface{}{},
			want: model.TelegramRichAuto,
		},
		{"auto", map[string]interface{}{"telegram_rich": "auto"}, model.TelegramRichAuto, false},
		{"on", map[string]interface{}{"telegram_rich": "on"}, model.TelegramRichOn, false},
		{"off", map[string]interface{}{"telegram_rich": "off"}, model.TelegramRichOff, false},
		{
			name: "uppercase and padding are tolerated",
			raw:  map[string]interface{}{"telegram_rich": "  ON "},
			want: model.TelegramRichOn,
		},
		{
			name: "yaml bool true means on",
			raw:  map[string]interface{}{"telegram_rich": true},
			want: model.TelegramRichOn,
		},
		{
			name: "yaml bool false means off",
			raw:  map[string]interface{}{"telegram_rich": false},
			want: model.TelegramRichOff,
		},
		{
			name: "quoted true means on",
			raw:  map[string]interface{}{"telegram_rich": "true"},
			want: model.TelegramRichOn,
		},
		{
			name: "quoted false means off",
			raw:  map[string]interface{}{"telegram_rich": "false"},
			want: model.TelegramRichOff,
		},
		{
			name:    "a typo is an error, not a silent off",
			raw:     map[string]interface{}{"telegram_rich": "eanbled"},
			want:    model.TelegramRichAuto,
			wantErr: true,
		},
		{
			name:    "empty string is an error",
			raw:     map[string]interface{}{"telegram_rich": ""},
			want:    model.TelegramRichAuto,
			wantErr: true,
		},
		{
			name:    "wrong type is an error",
			raw:     map[string]interface{}{"telegram_rich": 1},
			want:    model.TelegramRichAuto,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &model.NoteView{RawMeta: tt.raw}

			got, err := note.ExtractTelegramRichMode()
			require.Equal(t, tt.want, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// V1 sends rich only on an explicit `on`. `auto` stays classic: the classic
// converter reports string warnings that mix conversion losses with length and
// policy warnings, so it cannot serve as a selection predicate.
func TestTelegramRichModeUseRich(t *testing.T) {
	require.True(t, model.TelegramRichOn.UseRich())
	require.False(t, model.TelegramRichAuto.UseRich())
	require.False(t, model.TelegramRichOff.UseRich())
}
