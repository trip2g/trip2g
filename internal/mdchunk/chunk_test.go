package mdchunk

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// StripFrontmatter tests
// ---------------------------------------------------------------------------

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with frontmatter",
			input: "---\nfree: true\n---\nbody",
			want:  "body",
		},
		{
			name:  "no frontmatter",
			input: "just text",
			want:  "just text",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "only frontmatter",
			input: "---\nkey: val\n---",
			want:  "",
		},
		{
			name:  "unclosed frontmatter",
			input: "---\nkey: val\nbody",
			want:  "---\nkey: val\nbody",
		},
		{
			name:  "frontmatter with trailing newline",
			input: "---\nfree: true\n---\n\nbody paragraph",
			want:  "\nbody paragraph",
		},
		{
			name:  "dashes not at start",
			input: "text\n---\nkey: val\n---",
			want:  "text\n---\nkey: val\n---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripFrontmatter(tt.input)
			if got != tt.want {
				t.Errorf("StripFrontmatter(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Split tests
// ---------------------------------------------------------------------------

// searchAstronomyContent mirrors docs/demo/search_astronomy.md (body after frontmatter).
const searchAstronomyContent = `---
free: true
title: Астрономия и космос
---

Астрономия изучает планеты, звёзды и галактики.

Экзопланеты — это планеты за пределами Солнечной системы.
Их обнаруживают методом транзита и доплеровской спектроскопии.

Нейтронные звёзды — сверхплотные объекты, возникающие после взрыва сверхновой.
Гравитационные волны — рябь пространства-времени, предсказанная Эйнштейном.`

//nolint:gocognit // table-driven tests naturally have high complexity
func TestSplit(t *testing.T) {
	t.Run("short note (search_astronomy.md)", func(t *testing.T) {
		chunks := Split("Астрономия и космос", []byte(searchAstronomyContent))
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if !strings.HasPrefix(chunks[0].Content, "Астрономия и космос") {
			t.Error("chunk should start with title prefix")
		}
		if strings.Contains(chunks[0].Content, "free: true") {
			t.Error("chunk must not contain frontmatter YAML")
		}
		if chunks[0].Index != 0 {
			t.Errorf("expected index 0, got %d", chunks[0].Index)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		chunks := Split("Title", []byte(""))
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Content != "Title" {
			t.Errorf("expected %q, got %q", "Title", chunks[0].Content)
		}
	})

	t.Run("only frontmatter", func(t *testing.T) {
		chunks := Split("Title", []byte("---\nfree: true\n---"))
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Content != "Title" {
			t.Errorf("expected %q, got %q", "Title", chunks[0].Content)
		}
	})

	t.Run("multi-paragraph >2000 chars produces >=2 chunks", func(t *testing.T) {
		para := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 20) // ~1120 chars
		body := para + "\n\n" + para + "\n\n" + para
		chunks := Split("Big Note", []byte(body))
		if len(chunks) < 2 {
			t.Fatalf("expected >=2 chunks for >2000 chars body, got %d", len(chunks))
		}
		for i, c := range chunks {
			if !strings.HasPrefix(c.Content, "Big Note") {
				t.Errorf("chunk %d missing title prefix", i)
			}
			if c.Index != i {
				t.Errorf("chunk %d has wrong index %d", i, c.Index)
			}
		}
	})

	t.Run("heading boundary splits into 2 chunks", func(t *testing.T) {
		body := "# Section1\n\nparagraph one\n\n# Section2\n\nparagraph two"
		chunks := Split("Article", []byte(body))
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunksContent(chunks))
		}
		if !strings.Contains(chunks[0].Content, "Section1") {
			t.Error("chunk 0 should contain Section1")
		}
		if !strings.Contains(chunks[1].Content, "Section2") {
			t.Error("chunk 1 should contain Section2")
		}
	})

	t.Run("code fence not split mid-block", func(t *testing.T) {
		fence := "```go\nfunc main() {\n\n\tfmt.Println(\"hello\")\n\n}\n```"
		body := "Introduction paragraph.\n\n" + fence
		chunks := Split("Code Note", []byte(body))
		// The code fence should remain intact in one chunk
		for _, c := range chunks {
			open := strings.Count(c.Content, "```")
			if open%2 != 0 {
				t.Errorf("code fence split across chunks: chunk content = %q", c.Content)
			}
		}
	})

	t.Run("overlap: last block of chunk N appears in chunk N+1", func(t *testing.T) {
		// Build 3+ paragraphs that exceed chunkTargetSize
		para := strings.Repeat("Word ", 400) // ~2000 chars each
		body := para + "\n\n" + para + "\n\n" + para
		chunks := Split("Overlap Note", []byte(body))
		if len(chunks) < 2 {
			t.Skip("need >=2 chunks to test overlap")
		}
		// The last block of chunk 0 (trimmed to chunkOverlap) should appear in chunk 1
		lastOfFirst := chunks[0].Content
		// Find the last "\n\n" section
		lastPart := lastOfFirst
		if idx := strings.LastIndex(lastOfFirst, "\n\n"); idx >= 0 {
			lastPart = lastOfFirst[idx+2:]
		}
		// Take up to chunkOverlap chars
		if len(lastPart) > chunkOverlap {
			lastPart = lastPart[len(lastPart)-chunkOverlap:]
		}
		if !strings.Contains(chunks[1].Content, lastPart) {
			t.Error("overlap block from chunk 0 not found in chunk 1")
		}
	})

	t.Run("tiny paragraphs accumulated into 1 chunk", func(t *testing.T) {
		// 10 × 50-char paragraphs = ~500 chars total, all below chunkMinSize individually
		var parts []string
		for i := range 10 {
			_ = i
			parts = append(parts, strings.Repeat("x", 50))
		}
		body := strings.Join(parts, "\n\n")
		chunks := Split("Tiny Note", []byte(body))
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk for tiny paragraphs, got %d", len(chunks))
		}
	})

	t.Run("unicode and Cyrillic multi-paragraph", func(t *testing.T) {
		para := strings.Repeat("Привет мир. Тестируем кириллицу. ", 30) // ~1050 chars
		body := para + "\n\n" + para + "\n\n" + para
		chunks := Split("Кириллица", []byte(body))
		if len(chunks) == 0 {
			t.Fatal("expected at least 1 chunk")
		}
		for _, c := range chunks {
			if !strings.HasPrefix(c.Content, "Кириллица") {
				t.Error("chunk missing title prefix")
			}
		}
	})

	t.Run("whitespace-only body", func(t *testing.T) {
		chunks := Split("Title", []byte("\n\n\n"))
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Content != "Title" {
			t.Errorf("expected %q, got %q", "Title", chunks[0].Content)
		}
	})
}

func chunksContent(chunks []Chunk) []string {
	out := make([]string, len(chunks))
	for i, c := range chunks {
		out[i] = c.Content
	}
	return out
}

// TestSplitCarriesHeadingPath verifies F4: each chunk's content is prefixed with
// the breadcrumb "{title} > {heading path}", so a deep chunk carries its section
// context. A note with several headings must split into multiple chunks.
func TestSplitCarriesHeadingPath(t *testing.T) {
	title := "My Note"
	// Build a body long enough to span multiple chunks, with distinct headings.
	body := "Intro paragraph.\n\n" +
		"## Alpha\n\n" + strings.Repeat("alpha content word. ", 60) + "\n\n" +
		"## Beta\n\n" + strings.Repeat("beta content word. ", 60)
	chunks := Split(title, []byte(body))

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// Each chunk's first line is its breadcrumb "{title} > {heading path}".
	crumbs := map[string]bool{}
	for _, c := range chunks {
		crumbs[firstLine(c.Content)] = true
	}
	if !crumbs["My Note > Alpha"] {
		t.Errorf("no chunk carries breadcrumb 'My Note > Alpha'; got %v", crumbs)
	}
	if !crumbs["My Note > Beta"] {
		t.Errorf("no chunk carries breadcrumb 'My Note > Beta'; got %v", crumbs)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
