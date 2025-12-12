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
			name:     "public channel link with different text",
			content:  "Check out [this post](https://t.me/channel_name/123)",
			expected: "Check out [[Note Title|this post]]",
		},
		{
			name:     "private channel link with different text",
			content:  "Check out [click here](https://t.me/c/1234567890/123)",
			expected: "Check out [[Note Title|click here]]",
		},
		{
			name:     "link text matches title - no alias needed",
			content:  "Check out [Note Title](https://t.me/channel_name/123)",
			expected: "Check out [[Note Title]]",
		},
		{
			name:     "multiple links mixed",
			content:  "See [post1](https://t.me/mychannel/123) and [post2](https://t.me/c/9876543210/456)",
			expected: "See [[Note Title|post1]] and [[Another Note|post2]]",
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
		{
			name:     "empty link text",
			content:  "Check out [](https://t.me/channel_name/123)",
			expected: "Check out [[Note Title]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceTelegramLinks(tt.content, postMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCustomEmojiIDs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single emoji",
			content:  "Hello ![🔥|20x20](https://ce.trip2g.com/5460736117236048513.webp) world",
			expected: []string{"5460736117236048513"},
		},
		{
			name:     "multiple different emojis",
			content:  "![](https://ce.trip2g.com/111.webp) and ![](https://ce.trip2g.com/222.webp)",
			expected: []string{"111", "222"},
		},
		{
			name:     "duplicate emojis - deduplicated",
			content:  "![](https://ce.trip2g.com/123.webp) and ![](https://ce.trip2g.com/123.webp)",
			expected: []string{"123"},
		},
		{
			name:     "no emojis",
			content:  "Just plain text with no emojis",
			expected: nil,
		},
		{
			name:     "emoji with alt text",
			content:  "![fire emoji](https://ce.trip2g.com/999.webp)",
			expected: []string{"999"},
		},
		{
			name:     "mixed content",
			content:  "Text ![](https://ce.trip2g.com/111.webp) more text ![](https://ce.trip2g.com/222.webp) end",
			expected: []string{"111", "222"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCustomEmojiIDs(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceCustomEmojiURLs(t *testing.T) {
	downloadedEmojis := map[string]string{
		"111": "tg_ce_111.webp",
		"222": "tg_ce_222.webp",
	}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "single emoji replacement",
			content:  "Hello ![🔥](https://ce.trip2g.com/111.webp) world",
			expected: "Hello ![🔥](./assets/tg_ce_111.webp) world",
		},
		{
			name:     "multiple emoji replacements",
			content:  "![](https://ce.trip2g.com/111.webp) and ![](https://ce.trip2g.com/222.webp)",
			expected: "![](./assets/tg_ce_111.webp) and ![](./assets/tg_ce_222.webp)",
		},
		{
			name:     "emoji not downloaded - keep original",
			content:  "![](https://ce.trip2g.com/999.webp)",
			expected: "![](https://ce.trip2g.com/999.webp)",
		},
		{
			name:     "mixed - some downloaded some not",
			content:  "![](https://ce.trip2g.com/111.webp) and ![](https://ce.trip2g.com/999.webp)",
			expected: "![](./assets/tg_ce_111.webp) and ![](https://ce.trip2g.com/999.webp)",
		},
		{
			name:     "no emojis in content",
			content:  "Just plain text",
			expected: "Just plain text",
		},
		{
			name:     "preserves alt text with size suffix",
			content:  "![emoji|20x20](https://ce.trip2g.com/111.webp)",
			expected: "![emoji|20x20](./assets/tg_ce_111.webp)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceCustomEmojiURLs(tt.content, downloadedEmojis)
			assert.Equal(t, tt.expected, result)
		})
	}
}
