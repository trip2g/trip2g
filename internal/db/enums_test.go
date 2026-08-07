package db_test

import (
	"testing"
	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

// Rich must be decided before media count is consulted: FromMediaCount can only
// ever answer with one of the three classic types, so letting it decide alone is
// what makes a rich row look classic to the update path.
func TestTelegramPublishSentMessagePostTypeFor(t *testing.T) {
	tests := []struct {
		name       string
		isRich     bool
		mediaCount int
		want       string
	}{
		{name: "classic text", isRich: false, mediaCount: 0, want: db.TelegramPublishSentMessagePostTypeText},
		{name: "classic photo", isRich: false, mediaCount: 1, want: db.TelegramPublishSentMessagePostTypePhoto},
		{name: "classic media group", isRich: false, mediaCount: 3, want: db.TelegramPublishSentMessagePostTypeMediaGroup},
		{name: "rich without media", isRich: true, mediaCount: 0, want: db.TelegramPublishSentMessagePostTypeRich},
		{name: "rich outranks media count", isRich: true, mediaCount: 4, want: db.TelegramPublishSentMessagePostTypeRich},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, db.TelegramPublishSentMessagePostTypeFor(tt.isRich, tt.mediaCount))
		})
	}
}

func TestTelegramPublishSentMessagePostTypeRichIsDistinct(t *testing.T) {
	require.Equal(t, "rich", db.TelegramPublishSentMessagePostTypeRich)
	require.NotEqual(t, db.TelegramPublishSentMessagePostTypeText, db.TelegramPublishSentMessagePostTypeRich)
}
