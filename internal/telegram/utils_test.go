package telegram_test

import (
	"strings"
	"testing"
	"trip2g/internal/telegram"
)

func TestTruncateContent_NoTruncationNeeded(t *testing.T) {
	content := "Short message"
	result := telegram.TruncateContent(content, false)
	if result != content {
		t.Errorf("expected no truncation for short content, got %q", result)
	}
}

func TestTruncateContent_SimpleText(t *testing.T) {
	// Create content that exceeds 4093 chars (4096 - 3 for '...')
	content := strings.Repeat("a", 5000)
	result := telegram.TruncateContent(content, false)

	// Should be truncated to 4093 + 3 ('...') = 4096
	if telegram.GetTelegramLength(result) > 4096 {
		t.Errorf("expected length <= 4096, got %d", telegram.GetTelegramLength(result))
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_WithImages(t *testing.T) {
	// Create content that exceeds 1021 chars (1024 - 3 for '...')
	content := strings.Repeat("a", 1500)
	result := telegram.TruncateContent(content, true)

	// Should be truncated to 1021 + 3 ('...') = 1024
	if telegram.GetTelegramLength(result) > 1024 {
		t.Errorf("expected length <= 1024, got %d", telegram.GetTelegramLength(result))
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_RemoveIncompleteTags(t *testing.T) {
	// Content that ends with incomplete tag after truncation
	baseContent := strings.Repeat("a", 4100)
	content := baseContent + "<b"

	result := telegram.TruncateContent(content, false)

	// Should remove the incomplete tag
	if strings.Contains(result, "<b") && !strings.Contains(result, "<b>") {
		t.Error("expected incomplete tag to be removed")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_RemoveUnclosedTag(t *testing.T) {
	// Content with unclosed tag
	content := "Some text <b>bold text without closing tag" + strings.Repeat("a", 4100)

	result := telegram.TruncateContent(content, false)

	// Should remove the unclosed <b> tag and its content
	if strings.Contains(result, "<b>") {
		t.Error("expected unclosed <b> tag to be removed")
	}

	if strings.Contains(result, "bold text") {
		t.Error("expected content of unclosed tag to be removed")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_PreserveClosedTags(t *testing.T) {
	// Content with properly closed tags
	content := "Text <b>bold</b> <i>italic</i> " + strings.Repeat("a", 4100)

	result := telegram.TruncateContent(content, false)

	// Closed tags before truncation point should be preserved
	if !strings.Contains(result, "<b>bold</b>") {
		t.Error("expected closed <b> tag to be preserved")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_NestedTags(t *testing.T) {
	// Content with nested unclosed tags
	content := "Text <b>bold <i>italic" + strings.Repeat("a", 4100)

	result := telegram.TruncateContent(content, false)

	// Should remove the outermost unclosed tag
	if strings.Contains(result, "<b>") {
		t.Error("expected unclosed nested tags to be removed")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_ComplexHTML(t *testing.T) {
	// Content with various HTML tags
	content := "<b>Bold</b> <i>Italic</i> <code>Code</code> <u>Underline</u> " + strings.Repeat("a", 4100)

	result := telegram.TruncateContent(content, false)

	// All properly closed tags should be preserved
	if !strings.Contains(result, "<b>Bold</b>") {
		t.Error("expected <b> tag to be preserved")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_TagWithAttributes(t *testing.T) {
	// Content with tag that has attributes
	content := `<a href="https://example.com">Link</a> ` + strings.Repeat("a", 4100)

	result := telegram.TruncateContent(content, false)

	// Properly closed tag with attributes should be preserved
	if !strings.Contains(result, `<a href="https://example.com">Link</a>`) {
		t.Error("expected tag with attributes to be preserved")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestTruncateContent_UnclosedTagWithAttributes(t *testing.T) {
	// Content with unclosed tag that has attributes
	content := `Text <a href="https://example.com">Link without closing` + strings.Repeat("a", 4100)

	result := telegram.TruncateContent(content, false)

	// Should remove the unclosed tag
	if strings.Contains(result, `<a href`) {
		t.Error("expected unclosed tag with attributes to be removed")
	}

	if !strings.HasSuffix(result, "...") {
		t.Error("expected result to end with '...'")
	}
}

func TestGetTelegramLength_ASCII(t *testing.T) {
	content := "Hello World"
	length := telegram.GetTelegramLength(content)
	if length != 11 {
		t.Errorf("expected length 11, got %d", length)
	}
}

func TestGetTelegramLength_Cyrillic(t *testing.T) {
	content := "Привет"
	length := telegram.GetTelegramLength(content)
	// Cyrillic characters are within BMP, so 1 UTF-16 code unit each
	if length != 6 {
		t.Errorf("expected length 6, got %d", length)
	}
}

func TestGetTelegramLength_Emoji(t *testing.T) {
	content := "😀"
	length := telegram.GetTelegramLength(content)
	// Emoji are outside BMP, so 2 UTF-16 code units
	if length != 2 {
		t.Errorf("expected length 2, got %d", length)
	}
}

func TestGetTelegramLength_Mixed(t *testing.T) {
	content := "Hello 😀 Привет"
	length := telegram.GetTelegramLength(content)
	// "Hello " = 6, "😀" = 2, " Привет" = 7
	expected := 6 + 2 + 7
	if length != expected {
		t.Errorf("expected length %d, got %d", expected, length)
	}
}
