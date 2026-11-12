package mdchunk

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

const pinnedChunkSeparator = "\n<<<chunk>>>\n"

func chunkBody(content string) string {
	_, body, _ := strings.Cut(content, "\n\n")
	return body
}

// TestSplitPinnedNotes pins the exact output for notes whose chunks already fit
// the cap. A changed chunk is re-embedded on deploy, so these must not move.
// The .chunks files were written by the chunker as it was before the cap.
func TestSplitPinnedNotes(t *testing.T) {
	for _, name := range []string{"guide", "zametka", "flat"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/pinned/" + name + ".md")
			if err != nil {
				t.Fatal(err)
			}
			golden, err := os.ReadFile("testdata/pinned/" + name + ".chunks")
			if err != nil {
				t.Fatal(err)
			}
			want := strings.Split(strings.TrimSuffix(string(golden), "\n"), pinnedChunkSeparator)

			got := chunksContent(Split("Pinned "+name, raw))
			if len(got) != len(want) {
				t.Fatalf("got %d chunks, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("chunk %d changed:\ngot  %q\nwant %q", i, got[i], want[i])
				}
				if tokens := EstimateTokens(chunkBody(want[i])); tokens > chunkTargetTokens {
					t.Errorf("pinned chunk %d is %d tokens, over the cap: the fixture pins nothing", i, tokens)
				}
			}
		})
	}
}

func numberedLines(n int, format string) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf(format, i)
	}
	return strings.Join(lines, "\n")
}

func TestSplitCapsChunkBody(t *testing.T) {
	sentences := make([]string, 400)
	for i := range sentences {
		sentences[i] = fmt.Sprintf("Sentence number %d talks about the import.", i)
	}
	cyrSentences := make([]string, 400)
	for i := range cyrSentences {
		cyrSentences[i] = fmt.Sprintf("Предложение номер %d рассказывает про импорт.", i)
	}
	table := "| id | owner | note |\n|---|---|---|\n" + numberedLines(400, "| %d | anna | moved to the new board |")
	fence := "```go\n" + numberedLines(200, "\tfmt.Println(\"line %d\")\n") + "\n```"

	tests := []struct {
		name string
		body string
		// units must each survive whole in some chunk
		units []string
	}{
		{"long paragraph", strings.Join(sentences, " "), sentences},
		{"long cyrillic paragraph", strings.Join(cyrSentences, " "), cyrSentences},
		{"chat log with single newlines", numberedLines(500, "anna: message %d, the import is done"), nil},
		{"list", numberedLines(500, "- item %d of the backlog"), nil},
		{"table", table, nil},
		{"code fence with blank lines", fence, nil},
		{"no whitespace at all", strings.Repeat("abcdefghij", 2000), nil},
		{"cyrillic without whitespace", strings.Repeat("абвгдежзик", 2000), nil},
		{"heading glued to a long block", "## Log\n" + numberedLines(500, "boris: reply %d"), nil},
		{
			"two blocks that only fit apart",
			strings.Repeat("word ", 352) + "\n\n" + strings.Repeat("next ", 352),
			[]string{"word word", "next next"},
		},
		{
			"small block before an oversized one",
			"## Notes\n\nShort intro.\n\n" + strings.Join(sentences, " "),
			sentences,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := Split("Note", []byte(tt.body))
			for _, c := range chunks {
				if tokens := EstimateTokens(chunkBody(c.Content)); tokens > chunkTargetTokens {
					t.Errorf("chunk %d body is %d tokens, cap is %d", c.Index, tokens, chunkTargetTokens)
				}
				if !utf8.ValidString(c.Content) {
					t.Errorf("chunk %d is not valid UTF-8", c.Index)
				}
				if strings.TrimSpace(chunkBody(c.Content)) == "" {
					t.Errorf("chunk %d has an empty body", c.Index)
				}
			}

			units := tt.units
			if units == nil && strings.Contains(tt.body, "\n") {
				units = strings.FieldsFunc(tt.body, func(r rune) bool { return r == '\n' })
			}
			all := strings.Join(chunksContent(chunks), "\n")
			for _, unit := range units {
				if !strings.Contains(all, strings.TrimSpace(unit)) {
					t.Fatalf("%q was lost or cut", unit)
				}
			}
		})
	}
}

// With no boundary to cut at, the pieces (each chunk minus its overlap block)
// must still add up to the input.
func TestSplitMidWordKeepsEveryRune(t *testing.T) {
	for _, body := range []string{
		strings.Repeat("abcdefghij", 2000),
		strings.Repeat("абвгдежзик", 2000),
		strings.Repeat("日本語のテキスト", 2000),
	} {
		var joined strings.Builder
		for _, c := range Split("Note", []byte(body)) {
			piece := chunkBody(c.Content)
			if _, rest, ok := strings.Cut(piece, "\n\n"); ok {
				piece = rest
			}
			joined.WriteString(piece)
		}
		if joined.String() != body {
			t.Errorf("pieces of %.20q do not add up to the input", body)
		}
	}
}

func TestSplitKeepsBreadcrumbAndOverlapOnPieces(t *testing.T) {
	body := "## Log\n\n" + numberedLines(500, "anna: message %d, the import is done")
	chunks := Split("Note", []byte(body))
	if len(chunks) < 3 {
		t.Fatalf("expected the block to be split, got %d chunks", len(chunks))
	}
	for i, c := range chunks {
		if firstLine(c.Content) != "Note > Log" {
			t.Errorf("chunk %d breadcrumb = %q", i, firstLine(c.Content))
		}
		if i == 0 {
			continue
		}
		prev := chunks[i-1].Content
		overlap, _, _ := strings.Cut(chunkBody(c.Content), "\n\n")
		if len(overlap) > chunkOverlap || !strings.HasSuffix(prev, overlap) {
			t.Errorf("chunk %d does not start with the tail of chunk %d: %q", i, i-1, overlap)
		}
	}
}

func TestCutBlock(t *testing.T) {
	tests := []struct {
		name     string
		block    string
		room     int
		wantHead string
		wantRest string
	}{
		{"newline first", "one. two\nthree. four five", 20, "one. two", "three. four five"},
		{"then sentence end", "one. two three. four five", 20, "one. two three.", "four five"},
		{"then whitespace", "one two three four five", 20, "one two three four", "five"},
		{"then mid-word", "abcdefghijklmnopqrstuvwxyz", 20, "abcdefghijklmnopqrst", "uvwxyz"},
		{"never inside a rune", "абвгдежзик", 5, "аб", "вгдежзик"},
		{"question mark ends a sentence", "готово? да, почти всё", 20, "готово?", "да, почти всё"},
		{"always makes progress", "абв", 0, "а", "бв"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head, rest := cutBlock(tt.block, tt.room)
			if head != tt.wantHead || rest != tt.wantRest {
				t.Errorf("cutBlock = %q, %q; want %q, %q", head, rest, tt.wantHead, tt.wantRest)
			}
		})
	}
}
