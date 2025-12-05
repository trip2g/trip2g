package importtelegramchannel

import "testing"

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello world this is a test message",
			expected: "Hello world this is a test message",
		},
		{
			name:     "more than 7 words truncates",
			input:    "one two three four five six seven eight nine ten",
			expected: "one two three four five six seven",
		},
		{
			name:     "with custom emoji markdown",
			input:    "![emoji](tg://emoji?id=123) Title here today",
			expected: "Title here today",
		},
		{
			name:     "with markdown links extracts text",
			input:    "[Click here](https://example.com) for more info",
			expected: "Click here for more info",
		},
		{
			name:     "with timecodes removes them",
			input:    "00:15 Introduction to the topic today",
			expected: "Introduction to the topic today",
		},
		{
			name:     "invalid filename chars removed",
			input:    "What is this? A test: yes/no",
			expected: "What is this A test yesno",
		},
		{
			name:     "trailing punctuation stripped",
			input:    "This is a title...",
			expected: "This is a title",
		},
		{
			name:     "bold markdown removed",
			input:    "**Bold title** with text here",
			expected: "Bold title with text here",
		},
		{
			name:     "multiline takes first paragraph",
			input:    "First line title\n\nSecond paragraph content",
			expected: "First line title",
		},
		{
			name:     "html tags removed",
			input:    "<b>Bold</b> and <i>italic</i> text",
			expected: "Bold and italic text",
		},
		{
			name:     "numbered emoji prefix stripped",
			input:    "![1](tg://emoji?id=123). First item here",
			expected: "First item here",
		},
		{
			name:     "ce.trip2g.com emoji stripped",
			input:    "![emoji](https://ce.trip2g.com/123.webp) Content here",
			expected: "Content here",
		},
		{
			name:     "percent sign removed",
			input:    "100% сил для успеха",
			expected: "100 сил для успеха",
		},
		{
			name:     "multiple special chars removed",
			input:    "File* name& with^ special~ chars",
			expected: "File name with special chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.input)
			if got != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		messageID     int
		usedFilenames map[string]bool
		expected      string
	}{
		{
			name:          "unique title",
			title:         "My Title",
			messageID:     123,
			usedFilenames: map[string]bool{},
			expected:      "My Title.md",
		},
		{
			name:          "duplicate title adds message ID",
			title:         "My Title",
			messageID:     456,
			usedFilenames: map[string]bool{"My Title.md": true},
			expected:      "My Title (456).md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilename(tt.title, tt.messageID, tt.usedFilenames)
			if got != tt.expected {
				t.Errorf("generateFilename() = %q, want %q", got, tt.expected)
			}
		})
	}
}
