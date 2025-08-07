package mdloader

import (
	"bytes"
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
			name: "free_cut true - no --- markers renders all content",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": true,
			},
			expectContains:    []string{"First paragraph", "Second paragraph", "Third paragraph", "Fourth paragraph"},
			expectNotContains: []string{},
		},
		{
			name: "free_cut 2 - no --- markers renders all content",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph", "Third paragraph", "Fourth paragraph"},
			expectNotContains: []string{},
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
			name: "combined mode - no --- markers so paragraphs limit applies",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.`,
			metadata: map[string]interface{}{
				"free_cut":        1, // no --- markers, so this doesn't apply
				"free_paragraphs": 2, // limits to 2 paragraphs
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph"},
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
			name: "headings count as paragraphs with free_paragraphs",
			markdown: `# Main Title

First paragraph.

## Sub heading

Second paragraph.`,
			metadata: map[string]interface{}{
				"free_paragraphs": 2,
			},
			expectContains:    []string{"Main Title", "First paragraph"},
			expectNotContains: []string{"Sub heading", "Second paragraph"},
		},
		{
			name: "lists count as paragraphs with free_paragraphs",
			markdown: `First paragraph.

- Item 1
- Item 2

Second paragraph.`,
			metadata: map[string]interface{}{
				"free_paragraphs": 2,
			},
			expectContains:    []string{"First paragraph"},
			expectNotContains: []string{"Second paragraph"},
		},
		{
			name: "formatting preserved in free content with free_paragraphs",
			markdown: `First paragraph with **bold** and *italic* text.

Second paragraph with link.

Third paragraph excluded.`,
			metadata: map[string]interface{}{
				"free_paragraphs": 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph"},
			expectNotContains: []string{"Third paragraph excluded"},
		},
		{
			name: "float64 values converted to int for free_paragraphs",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.`,
			metadata: map[string]interface{}{
				"free_paragraphs": 2.0, // JSON numbers are float64
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
			name: "free_paragraphs_1",
			metadata: map[string]interface{}{
				"free_paragraphs": 1,
			},
			expectCount: 1,
			description: "Should only include first paragraph",
		},
		{
			name: "free_paragraphs_2",
			metadata: map[string]interface{}{
				"free_paragraphs": 2,
			},
			expectCount: 2,
			description: "Should include title and first paragraph",
		},
		{
			name: "free_paragraphs_3",
			metadata: map[string]interface{}{
				"free_paragraphs": 3,
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

func TestRenderFreeContent_ErrorCases(t *testing.T) {
	md := goldmark.New()
	ldr := &loader{md: md}

	doc := md.Parser().Parse(text.NewReader([]byte("test")))
	var buf bytes.Buffer

	// Test invalid limits - both must be 0 or negative
	err := ldr.renderFreeContent(&buf, doc, []byte("test"), 0, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one limit must be positive")

	err = ldr.renderFreeContent(&buf, doc, []byte("test"), -1, -1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one limit must be positive")
}

func TestGenerateFreeHTML_CutMode(t *testing.T) {
	tests := []struct {
		name              string
		markdown          string
		metadata          map[string]interface{}
		expectContains    []string
		expectNotContains []string
	}{
		{
			name: "free_cut true - cut at first ---",
			markdown: `First paragraph before cut.

Second paragraph before cut.

---

This content should be excluded.

More excluded content.`,
			metadata: map[string]interface{}{
				"free_cut": true,
			},
			expectContains:    []string{"First paragraph before cut", "Second paragraph before cut"},
			expectNotContains: []string{"This content should be excluded", "More excluded content"},
		},
		{
			name: "free_cut 2 - cut at second ---",
			markdown: `First section.

---

Second section.

---

Third section should be excluded.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"First section", "Second section"},
			expectNotContains: []string{"Third section should be excluded"},
		},
		{
			name: "free_cut with no --- renders all content",
			markdown: `First paragraph.

Second paragraph.

Third paragraph.`,
			metadata: map[string]interface{}{
				"free_cut": 2,
			},
			expectContains:    []string{"First paragraph", "Second paragraph", "Third paragraph"},
			expectNotContains: []string{},
		},
		{
			name: "combined mode - stops at first condition met (cut wins)",
			markdown: `First section.

---

Second section.

Third section.`,
			metadata: map[string]interface{}{
				"free_cut":        1, // stops at first ---
				"free_paragraphs": 3, // would allow 3 paragraphs
			},
			expectContains:    []string{"First section"},
			expectNotContains: []string{"Second section", "Third section"},
		},
		{
			name: "combined mode - stops at first condition met (paragraphs wins)",
			markdown: `First section.

Second section.

Third section should be excluded.`,
			metadata: map[string]interface{}{
				"free_cut":        5, // would allow up to 5 --- marks
				"free_paragraphs": 2, // stops after 2 paragraphs
			},
			expectContains:    []string{"First section", "Second section"},
			expectNotContains: []string{"Third section should be excluded"},
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

			doc := md.Parser().Parse(text.NewReader([]byte(tt.markdown)))

			note := &model.NoteView{
				Content: []byte(tt.markdown),
				RawMeta: tt.metadata,
			}
			note.SetAst(doc)

			err := ldr.generateFreeHTML(note)
			require.NoError(t, err)

			freeHTML := string(note.FreeHTML)
			require.NotEmpty(t, freeHTML)

			for _, expected := range tt.expectContains {
				require.Contains(t, freeHTML, expected, "Free HTML should contain: %s", expected)
			}

			for _, notExpected := range tt.expectNotContains {
				require.NotContains(t, freeHTML, notExpected, "Free HTML should NOT contain: %s", notExpected)
			}
		})
	}
}
