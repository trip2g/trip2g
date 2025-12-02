package main

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestApplyEntities(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []tg.MessageEntityClass
		expected string
	}{
		{
			name:     "no entities",
			text:     "Hello world",
			entities: nil,
			expected: "Hello world",
		},
		{
			name: "single bold",
			text: "Hello world",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityBold{Offset: 0, Length: 5},
			},
			expected: "**Hello** world",
		},
		{
			name: "single custom emoji",
			text: "🗣 Hello",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityCustomEmoji{Offset: 0, Length: 2, DocumentID: 123456},
			},
			expected: "![🗣](https://ce.trip2g.com/123456.webp) Hello",
		},
		{
			name: "bold then custom emoji - non overlapping",
			text: "Text 🗣 more",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityBold{Offset: 0, Length: 4},
				&tg.MessageEntityCustomEmoji{Offset: 5, Length: 2, DocumentID: 123456},
			},
			expected: "**Text** ![🗣](https://ce.trip2g.com/123456.webp) more",
		},
		{
			name: "custom emoji inside bold",
			text: "Hello 🗣 world",
			entities: []tg.MessageEntityClass{
				// "Hello " (6) + emoji (2 surrogate) + " world" (6) = 14 UTF-16 units
				&tg.MessageEntityBold{Offset: 0, Length: 14},
				&tg.MessageEntityCustomEmoji{Offset: 6, Length: 2, DocumentID: 123456},
			},
			expected: "**Hello ![🗣](https://ce.trip2g.com/123456.webp) world**",
		},
		{
			name: "nested bold and italic",
			text: "Hello world",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityBold{Offset: 0, Length: 11},
				&tg.MessageEntityItalic{Offset: 6, Length: 5},
			},
			expected: "**Hello *world***",
		},
		{
			name: "text link with custom emoji inside",
			text: "🗣 НАВИГАЦИЯ",
			entities: []tg.MessageEntityClass{
				// emoji (2) + " НАВИГАЦИЯ" (10) = 12 UTF-16 units
				&tg.MessageEntityCustomEmoji{Offset: 0, Length: 2, DocumentID: 5821302890932736039},
				&tg.MessageEntityTextURL{Offset: 0, Length: 12, URL: "https://t.me/ryspaisensei/659"},
			},
			// Link wraps everything, emoji is inside
			expected: "[![🗣](https://ce.trip2g.com/5821302890932736039.webp) НАВИГАЦИЯ](https://t.me/ryspaisensei/659)",
		},
		{
			name: "bold and italic same range",
			text: "Hello",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityBold{Offset: 0, Length: 5},
				&tg.MessageEntityItalic{Offset: 0, Length: 5},
			},
			expected: "***Hello***",
		},
		{
			name: "link with bold text",
			text: "Click here",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityTextURL{Offset: 0, Length: 10, URL: "https://example.com"},
				&tg.MessageEntityBold{Offset: 0, Length: 10},
			},
			expected: "[**Click here**](https://example.com)",
		},
		{
			name: "bold wrapping custom emoji",
			text: "⚡️ Фрагменты",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityBold{Offset: 0, Length: 2},
				&tg.MessageEntityCustomEmoji{Offset: 0, Length: 2, DocumentID: 5463038705038007921},
			},
			// Bold should wrap the emoji image, not be inside alt text
			expected: "**![⚡️](https://ce.trip2g.com/5463038705038007921.webp)** Фрагменты",
		},
		{
			name: "italic with bold inside ending same position",
			text: "Диалог. Человек один.\n\nСложный фильм.",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityItalic{Offset: 0, Length: 23}, // includes \n\n
				&tg.MessageEntityBold{Offset: 8, Length: 15},   // "Человек один.\n\n"
			},
			// Formatting closes before \n\n and reopens after
			expected: "*Диалог. **Человек один.***\n\n***Сложный фильм.",
		},
		{
			name: "formatting across paragraph - closes and reopens",
			text: "First para.\n\nSecond para.",
			entities: []tg.MessageEntityClass{
				&tg.MessageEntityItalic{Offset: 0, Length: 25}, // entire text
			},
			expected: "*First para.*\n\n*Second para.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyEntities(tt.text, tt.entities)
			if got != tt.expected {
				t.Errorf("applyEntities() =\n  got:  %q\n  want: %q", got, tt.expected)
			}
		})
	}
}
