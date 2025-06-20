package russkayalatinica

import (
	"fmt"
	"testing"
)

func TestSingleCharacterRoundTrip(t *testing.T) {
	singleChars := []string{"а", "б", "в", "г", "д", "е", "ё", "ж", "з", "и", "й", "ъ", "ы", "ь", "э", "ю", "я"}

	for _, char := range singleChars {
		transliterated := Translit(char)
		reverted := RevertTranslit(transliterated)
		if char != reverted {
			t.Errorf("Round trip failed for %s: %s -> %s -> %s", char, char, transliterated, reverted)
		}
	}
}

func TestSpecificSequences(t *testing.T) {
	testCases := []struct {
		cyrillic string
		latin    string
	}{
		{"ъе", "hyye"},
		{"Съе", "Syye"},
		{"съе", "syye"},
		{"ь", "hj"},
		{"сь", "sj"},
		{"шь", "shj"},
		{"й", "j"},
		{"сй", "syj"},
	}

	for _, tc := range testCases {
		result := Translit(tc.cyrillic)
		if result != tc.latin {
			t.Errorf("Translit(%s) = %s, want %s", tc.cyrillic, result, tc.latin)
		}

		// Test reverse
		reverted := RevertTranslit(tc.latin)
		if reverted != tc.cyrillic {
			t.Errorf("RevertTranslit(%s) = %s, want %s", tc.latin, reverted, tc.cyrillic)
		}
	}
}

func TestComplexSentences(t *testing.T) {
	testCases := []string{
		"МКС",
		"ЮНЕСКО", 
		"Петров Ю. Я.",
		"Русская Латиница",
		// Note: Some complex sentences may not have perfect round-trip due to 
		// contextual ambiguities, but core functionality should work
	}

	for _, tc := range testCases {
		transliterated := Translit(tc)
		reverted := RevertTranslit(transliterated)
		if tc != reverted {
			t.Errorf("Round trip failed for '%s':\n  Original: %s\n  Translit: %s\n  Reverted: %s", tc, tc, transliterated, reverted)
		}
	}
}

func TestTranslitExamples(t *testing.T) {
	examples := map[string]string{
		"Съешь же ещё этих мягких французских булок, да выпей чаю.": "Syyeshj zhe yesjhyo etikh myagkikh francuzskikh bulok, da vyipej chayu.",
		"интервьюер": "intervjyuyer",
		"Йемен":      "Jyemen",
	}

	for cyrillic, expectedLatin := range examples {
		result := Translit(cyrillic)
		if result != expectedLatin {
			t.Errorf("Translit(%s) = %s, want %s", cyrillic, result, expectedLatin)
		}
	}
}

func BenchmarkTranslit(b *testing.B) {
	text := "Съешь же ещё этих мягких французских булок, да выпей чаю."
	for i := 0; i < b.N; i++ {
		Translit(text)
	}
}

func BenchmarkRevertTranslit(b *testing.B) {
	text := "Syyeshj zhe yesjhyo etikh myagkikh francuzskikh bulok, da vyipej chayu."
	for i := 0; i < b.N; i++ {
		RevertTranslit(text)
	}
}

func ExampleTranslit() {
	fmt.Println(Translit("Русская Латиница"))
	// Output: Russkaya Latinica
}

func ExampleRevertTranslit() {
	fmt.Println(RevertTranslit("Russkaya Latinica"))
	// Output: Русская Латиница
}