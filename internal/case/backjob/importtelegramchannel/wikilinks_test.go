package importtelegramchannel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceTelegramLinks(t *testing.T) {
	postMap := map[string]string{
		"123": "Note Title",
		"456": "Another Note",
	}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "public channel link",
			content:  "Check out [this post](https://t.me/channel_name/123)",
			expected: "Check out [[Note Title]]",
		},
		{
			name:     "private channel link",
			content:  "Check out [this post](https://t.me/c/1234567890/123)",
			expected: "Check out [[Note Title]]",
		},
		{
			name:     "multiple links mixed",
			content:  "See [post1](https://t.me/mychannel/123) and [post2](https://t.me/c/9876543210/456)",
			expected: "See [[Note Title]] and [[Another Note]]",
		},
		{
			name:     "link not in postMap - keep original",
			content:  "Check [unknown](https://t.me/channel/999)",
			expected: "Check [unknown](https://t.me/channel/999)",
		},
		{
			name:     "no links",
			content:  "Just plain text",
			expected: "Just plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceTelegramLinks(tt.content, postMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}
