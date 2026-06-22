package mdchunk

import (
	"strings"
	"testing"
)

func TestTruncateToTokens(t *testing.T) {
	t.Run("returns text unchanged when under budget", func(t *testing.T) {
		in := "short note body"
		got := TruncateToTokens(in, 1000)
		if got != in {
			t.Fatalf("expected unchanged, got %q", got)
		}
	})

	t.Run("truncates oversized text to within budget", func(t *testing.T) {
		// ~4000 Latin chars ≈ 1000 estimated tokens, well over a 100-token budget.
		in := strings.Repeat("word ", 800)
		got := TruncateToTokens(in, 100)
		if estimateTokens(got) > 100 {
			t.Fatalf("result over budget: %d estimated tokens", estimateTokens(got))
		}
		if !strings.HasPrefix(in, got) {
			t.Fatalf("truncated result must be a prefix of the input")
		}
	})

	t.Run("truncates oversized Cyrillic text within budget", func(t *testing.T) {
		in := strings.Repeat("привет мир ", 2000)
		got := TruncateToTokens(in, 200)
		if estimateTokens(got) > 200 {
			t.Fatalf("result over budget: %d estimated tokens", estimateTokens(got))
		}
	})

	t.Run("does not split a multibyte rune", func(t *testing.T) {
		in := strings.Repeat("ё", 5000) // 2-byte runes
		got := TruncateToTokens(in, 50)
		if !strings.ContainsRune(got, 'ё') && got != "" {
			t.Fatalf("unexpected mangled output")
		}
		// Valid UTF-8: no replacement chars introduced by byte-cutting.
		if strings.ContainsRune(got, '�') {
			t.Fatalf("truncation split a multibyte rune")
		}
	})
}
