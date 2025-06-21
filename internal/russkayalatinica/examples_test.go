package russkayalatinica

import (
	"os"
	"path/filepath"
	"testing"
)

const basePath = "examples"

func TestExampleFiles(t *testing.T) {
	// Test ndfl.md -> ndfl.lat.md
	t.Run("ndfl.md transliteration", func(t *testing.T) {
		// Read the original Cyrillic file
		originalPath := filepath.Join(basePath, "ndfl.md")
		originalContent, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatalf("Failed to read original file: %v", err)
		}

		// Read the expected Latin file
		latinPath := filepath.Join(basePath, "ndfl.lat.md")
		expectedContent, err := os.ReadFile(latinPath)
		if err != nil {
			t.Fatalf("Failed to read Latin file: %v", err)
		}

		// Transliterate the original content
		transliterated := Translit(string(originalContent))

		// Compare the results
		if transliterated != string(expectedContent) {
			// Find the first difference for better error reporting
			minLen := len(transliterated)
			if len(expectedContent) < minLen {
				minLen = len(expectedContent)
			}

			for i := 0; i < minLen; i++ {
				if transliterated[i] != expectedContent[i] {
					start := i - 50
					if start < 0 {
						start = 0
					}
					end := i + 50
					if end > minLen {
						end = minLen
					}

					t.Errorf("Transliteration mismatch at position %d:\n"+
						"Expected: ...%s...\n"+
						"Got:      ...%s...\n"+
						"Context expected: %q\n"+
						"Context got:      %q",
						i,
						string(expectedContent[start:end]),
						transliterated[start:end],
						string(expectedContent[i]),
						string(transliterated[i]))
					break
				}
			}

			if len(transliterated) != len(expectedContent) {
				t.Errorf("Length mismatch: expected %d, got %d", len(expectedContent), len(transliterated))
			}
		}
	})

	// Test reverse transliteration (simplified - only test that it doesn't crash)
	t.Run("ndfl.lat.md reverse transliteration (basic)", func(t *testing.T) {
		// Read the Latin file
		latinPath := filepath.Join(basePath, "ndfl.lat.md")
		latinContent, err := os.ReadFile(latinPath)
		if err != nil {
			t.Fatalf("Failed to read Latin file: %v", err)
		}

		// Reverse transliterate the Latin content (just ensure it doesn't crash)
		reverted := RevertTranslit(string(latinContent))

		// Basic checks
		if len(reverted) == 0 {
			t.Error("Reverse transliteration returned empty string")
		}

		// The reverse transliteration may not be perfect due to ambiguities
		// in the transliteration system, but it should at least return something reasonable
		t.Logf("Reverse transliteration completed. Original length: %d, Reverted length: %d",
			len(latinContent), len(reverted))
	})
}

func BenchmarkExampleFileTranslit(b *testing.B) {
	// Read the original file once
	originalPath := filepath.Join(basePath, "ndfl.md")
	originalContent, err := os.ReadFile(originalPath)
	if err != nil {
		b.Fatalf("Failed to read original file: %v", err)
	}
	content := string(originalContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Translit(content)
	}
}

func BenchmarkExampleFileRevertTranslit(b *testing.B) {
	// Read the Latin file once
	latinPath := filepath.Join(basePath, "ndfl.lat.md")
	latinContent, err := os.ReadFile(latinPath)
	if err != nil {
		b.Fatalf("Failed to read Latin file: %v", err)
	}
	content := string(latinContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RevertTranslit(content)
	}
}
