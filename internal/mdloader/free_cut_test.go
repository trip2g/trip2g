package mdloader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"trip2g/internal/model"
)

func TestGenerateFreeHTML(t *testing.T) {
	tests := []struct {
		name              string
		markdown          string
		metadata          map[string]interface{}
		config            Config
		expectContains    []string
		expectNotContains []string
		expectEmpty       bool
	}{
		{
			name: "free_cut true - should include first paragraph only",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": true,
			},
			expectContains:    []string{"First paragraph"},
			expectNotContains: []string{"Second paragraph", "Third paragraph", "Fourth paragraph"},
		},
		{
			name: "free_cut 2 - should include first two paragraphs",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph", "Fourth paragraph"},
		},
		{
			name: "free_paragraphs 2 - should include first two paragraphs",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.`,
			metadata: map[string]interface{}{
				"free_paragraphs": 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph", "Fourth paragraph"},
		},
		{
			name: "free_cut takes precedence over free_paragraphs",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.`,
			metadata: map[string]interface{}{
				"free_cut":        1,
				"free_paragraphs": 3,
			},
			expectContains:    []string{"First paragraph"},
			expectNotContains: []string{"Second paragraph", "Third paragraph"},
		},
		{
			name: "config default when no metadata",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.`,
			config: Config{
				FreeParagraphs: 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph"},
		},
		{
			name: "no free HTML when nothing configured",
			markdown: `First paragraph.

Second paragraph.`,
			expectEmpty: true,
		},
		{
			name: "headings count as paragraphs",
			markdown: `# Main Title

First paragraph.

## Sub heading

Second paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"Main Title", "First paragraph"},
			expectNotContains: []string{"Sub heading", "Second paragraph"},
		},
		{
			name: "lists count as paragraphs",
			markdown: `First paragraph.

- Item 1
- Item 2

Second paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"First paragraph"},
			expectNotContains: []string{"Second paragraph"},
		},
		{
			name: "formatting preserved in free content",
			markdown: `First paragraph with **bold** and *italic* text.

Second paragraph with link.

Third paragraph excluded.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph excluded"},
		},
		{
			name: "float64 values converted to int",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": 2.0, // JSON numbers are float64
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create loader with goldmark
			md := goldmark.New(
				goldmark.WithExtensions(extension.GFM),
			)

			ldr := &loader{
				md:     md,
				config: tt.config,
			}

			// Parse markdown
			doc := md.Parser().Parse(text.NewReader([]byte(tt.markdown)))

			// Create note view
			note := &model.NoteView{
				Content: []byte(tt.markdown),
				RawMeta: tt.metadata,
			}
			note.SetAst(doc)

			// Generate free HTML
			err := ldr.generateFreeHTML(note)
			require.NoError(t, err)

			if tt.expectEmpty {
				require.Empty(t, note.FreeHTML)
				return
			}

			freeHTML := string(note.FreeHTML)
			require.NotEmpty(t, freeHTML)

			// Check that expected content is included
			for _, expected := range tt.expectContains {
				require.Contains(t, freeHTML, expected, "Free HTML should contain: %s", expected)
			}

			// Check that unwanted content is excluded
			for _, notExpected := range tt.expectNotContains {
				require.NotContains(t, freeHTML, notExpected, "Free HTML should NOT contain: %s", notExpected)
			}
		})
	}
}

func TestFreeHTMLIntegration(t *testing.T) {
	markdown := `# Article Title

This is paragraph 1 with some content.

This is paragraph 2 with more content.

This is paragraph 3 that should be excluded.

This is paragraph 4 that should also be excluded.`

	tests := []struct {
		name        string
		metadata    map[string]interface{}
		expectCount int
		description string
	}{
		{
			name: "free_cut_1",
			metadata: map[string]interface{}{
				"free_cut": 1,
			},
			expectCount: 1,
			description: "Should only include first paragraph",
		},
		{
			name: "free_cut_2",
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectCount: 2,
			description: "Should include title and first paragraph",
		},
		{
			name: "free_cut_3",
			metadata: map[string]interface{}{
				"free_cut": 3,
			},
			expectCount: 3,
			description: "Should include title, first, and second paragraphs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := goldmark.New(
				goldmark.WithExtensions(extension.GFM),
			)

			ldr := &loader{
				md: md,
			}

			doc := md.Parser().Parse(text.NewReader([]byte(markdown)))

			note := &model.NoteView{
				Content: []byte(markdown),
				RawMeta: tt.metadata,
			}
			note.SetAst(doc)

			err := ldr.generateFreeHTML(note)
			require.NoError(t, err)

			freeHTML := string(note.FreeHTML)
			require.NotEmpty(t, freeHTML)

			// Count paragraphs in the free HTML
			// This is a rough count - we look for opening paragraph and heading tags
			paragraphCount := strings.Count(freeHTML, "<p>") + strings.Count(freeHTML, "<h1>")
			require.Equal(t, tt.expectCount, paragraphCount, tt.description)

			// Verify content based on expected count
			switch tt.expectCount {
			case 1:
				require.Contains(t, freeHTML, "Article Title")
				require.NotContains(t, freeHTML, "paragraph 1")
			case 2:
				require.Contains(t, freeHTML, "Article Title")
				require.Contains(t, freeHTML, "paragraph 1")
				require.NotContains(t, freeHTML, "paragraph 2")
			case 3:
				require.Contains(t, freeHTML, "Article Title")
				require.Contains(t, freeHTML, "paragraph 1")
				require.Contains(t, freeHTML, "paragraph 2")
				require.NotContains(t, freeHTML, "paragraph 3")
			}
		})
	}
}

func TestExtractFirstNParagraphs_ErrorCases(t *testing.T) {
	md := goldmark.New()
	ldr := &loader{md: md}

	doc := md.Parser().Parse(text.NewReader([]byte("test")))

	// Test invalid paragraph counts
	_, err := ldr.extractFirstNParagraphs(doc, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid number of paragraphs")

	_, err = ldr.extractFirstNParagraphs(doc, -1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid number of paragraphs")
}
