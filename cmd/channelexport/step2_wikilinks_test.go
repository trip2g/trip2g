package main

import "testing"

func TestReplaceTelegramLinks(t *testing.T) {
	postMap := map[string]PostInfo{
		"123": {ID: "123", Title: "Первый пост", Filename: "Первый пост.md"},
		"456": {ID: "456", Title: "Второй пост о важном", Filename: "Второй пост о важном.md"},
		"789": {ID: "789", Title: "Третий", Filename: "Третий.md"},
	}

	tests := []struct {
		name          string
		content       string
		expected      string
		expectedCount int
	}{
		{
			name:          "simple link",
			content:       `Смотри [тут](https://t.me/ryspaisensei/123) важное`,
			expected:      `Смотри [[Первый пост]] важное`,
			expectedCount: 1,
		},
		{
			name:          "multiple links",
			content:       `Читай [пост](https://t.me/ryspaisensei/123) и [другой](https://t.me/ryspaisensei/456)`,
			expected:      `Читай [[Первый пост]] и [[Второй пост о важном]]`,
			expectedCount: 2,
		},
		{
			name:          "link not in map",
			content:       `Смотри [тут](https://t.me/ryspaisensei/999) важное`,
			expected:      `Смотри [тут](https://t.me/ryspaisensei/999) важное`,
			expectedCount: 0,
		},
		{
			name:          "mixed - some in map some not",
			content:       `[Есть](https://t.me/ryspaisensei/123) и [нет](https://t.me/ryspaisensei/999)`,
			expected:      `[[Первый пост]] и [нет](https://t.me/ryspaisensei/999)`,
			expectedCount: 1,
		},
		{
			name:          "no telegram links",
			content:       `Просто текст без ссылок`,
			expected:      `Просто текст без ссылок`,
			expectedCount: 0,
		},
		{
			name:          "other domain link unchanged",
			content:       `Смотри [тут](https://example.com/123) важное`,
			expected:      `Смотри [тут](https://example.com/123) важное`,
			expectedCount: 0,
		},
		{
			name:          "http link (not https)",
			content:       `Смотри [тут](http://t.me/ryspaisensei/123) важное`,
			expected:      `Смотри [[Первый пост]] важное`,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, count := replaceTelegramLinks(tt.content, postMap)
			if got != tt.expected {
				t.Errorf("replaceTelegramLinks() content = %q, want %q", got, tt.expected)
			}
			if count != tt.expectedCount {
				t.Errorf("replaceTelegramLinks() count = %d, want %d", count, tt.expectedCount)
			}
		})
	}
}

func TestExtractMessageID(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "standard frontmatter",
			content: `---
telegram_publish_message_id: 123
---

Content`,
			expected: "123",
		},
		{
			name: "quoted value",
			content: `---
telegram_publish_message_id: "456"
---

Content`,
			expected: "456",
		},
		{
			name:     "no frontmatter",
			content:  `Just content without frontmatter`,
			expected: "",
		},
		{
			name: "frontmatter without message id",
			content: `---
title: Something
---

Content`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageID(tt.content)
			if got != tt.expected {
				t.Errorf("extractMessageID() = %q, want %q", got, tt.expected)
			}
		})
	}
}
