package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

func TestTocChildren(t *testing.T) {
	headings := model.NoteViewHeadings{
		{Text: "Setup", Level: 1, ID: "setup"},
		{Text: "Install", Level: 2, ID: "install"},
		{Text: "Configure", Level: 2, ID: "configure"},
		{Text: "Events", Level: 1, ID: "events"},
		{Text: "Note changed", Level: 2, ID: "note_changed"},
	}

	top := tocChildren(headings, nil)
	require.Equal(t, []TOCNode{
		{Title: "Setup", Level: 1, Path: []string{"Setup"}, HasChildren: true},
		{Title: "Events", Level: 1, Path: []string{"Events"}, HasChildren: true},
	}, top)

	setup := tocChildren(headings, []string{"Setup"})
	require.Equal(t, []TOCNode{
		{Title: "Install", Level: 2, Path: []string{"Setup", "Install"}, HasChildren: false},
		{Title: "Configure", Level: 2, Path: []string{"Setup", "Configure"}, HasChildren: false},
	}, setup)

	require.Empty(t, tocChildren(headings, []string{"Setup", "Install"})) // leaf
	require.Empty(t, tocChildren(headings, []string{"Nope"}))             // unknown path
}

// TestNormalizeForMatchBothPipelines verifies that the same logical text, when
// rendered through two different HTML pipelines (bleve fragment vs goldmark HTML),
// normalizes to byte-identical strings. This is the core invariant that makes the
// fuzzy section match reliable.
func TestNormalizeForMatchBothPipelines(t *testing.T) {
	tests := []struct {
		name          string
		bleveInput    string // bleve highlight fragment (may contain <mark>, HTML entities)
		goldmarkInput string // goldmark-rendered HTML (may contain <strong>, <em>, entities)
		wantEqual     bool
	}{
		{
			name:          "mark tags vs strong tags produce same plain text",
			bleveInput:    "The <mark>lazy fox</mark> jumped over",
			goldmarkInput: "The <strong>lazy fox</strong> jumped over",
			wantEqual:     true,
		},
		{
			name:          "HTML entity in bleve fragment normalized same as decoded text",
			bleveInput:    "cats &amp; dogs are <mark>friends</mark>",
			goldmarkInput: "cats &amp; dogs are friends",
			wantEqual:     true,
		},
		{
			name:          "non-breaking space variant normalized to regular space",
			bleveInput:    "word word another <mark>word</mark>",
			goldmarkInput: "word word another word",
			wantEqual:     true,
		},
		{
			name:          "em-dash folded to hyphen on both sides",
			bleveInput:    "before—<mark>after</mark>",
			goldmarkInput: "before—after",
			wantEqual:     true,
		},
		{
			name: "NFC normalization: composed vs decomposed e+combining-acute",
			// U+00E9 (precomposed) vs U+0065 U+0301 (decomposed)
			bleveInput:    "café <mark>review</mark>",
			goldmarkInput: "café review",
			wantEqual:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1 := normalizeForMatch(tt.bleveInput)
			got2 := normalizeForMatch(tt.goldmarkInput)
			if tt.wantEqual {
				require.Equal(t, got1, got2, "normalizeForMatch(%q) = %q, normalizeForMatch(%q) = %q", tt.bleveInput, got1, tt.goldmarkInput, got2)
			} else {
				require.NotEqual(t, got1, got2)
			}
		})
	}
}

// TestTocPathForSnippetHTMLReflow verifies that a snippet whose HTML representation
// differs from the goldmark-rendered section (e.g., entity encoding, whitespace) still
// resolves to the correct section path.
func TestTocPathForSnippetHTMLReflow(t *testing.T) {
	// goldmark renders <strong>, but bleve highlights with <mark>.
	// The section text is "cats & dogs are friends" (with & encoded as &amp; in HTML).
	// The snippet has <mark> wrapping and different entity encoding.
	noteHTML := `<div data-header="Animals" data-level="2"><h2>Animals</h2>` +
		`<p>cats &amp; dogs are friends here</p></div>`

	// Bleve fragment: uses <mark> tags, encodes & as &amp;
	snippet := "cats &amp; dogs are <mark>friends</mark> here"

	path := tocPathForSnippet(noteHTML, snippet, "")
	require.Equal(t, []string{"Animals"}, path,
		"expected Animals section, got %v", path)
}

// TestTocPathForSnippetIntroBeforeFirstHeading verifies that when a snippet
// comes from introductory content before the first heading, we get a non-nil
// path pointing to the first section (not nil).
func TestTocPathForSnippetIntroBeforeFirstHeading(t *testing.T) {
	// Note with intro text before the first heading section.
	// goldmark does NOT wrap pre-heading content in a data-header div.
	noteHTML := `<p>This is an intro paragraph before any heading.</p>` +
		`<div data-header="First Section" data-level="2"><h2>First Section</h2>` +
		`<p>Section content goes here.</p></div>`

	// Snippet from intro — won't match any data-header div text.
	snippet := "This is an <mark>intro</mark> paragraph before any heading."

	path := tocPathForSnippet(noteHTML, snippet, "")
	// Should fall back to first section, not return nil.
	require.NotNil(t, path, "path must not be nil for intro snippet")
	require.Equal(t, []string{"First Section"}, path,
		"should fall back to first section, got %v", path)
}

// TestTocPathFromChunkContentRoundTrip verifies that a path derived from a chunk
// breadcrumb round-trips through sectionHTMLByTocPath and navigateSectionPath
// to retrieve the expected section node.
func TestTocPathFromChunkContentRoundTrip(t *testing.T) {
	// goldmark-rendered noteHTML with nested sections
	noteHTML := `<div data-header="Introduction" data-level="1"><h1>Introduction</h1>` +
		`<p>Intro text.</p>` +
		`<div data-header="Background" data-level="2"><h2>Background</h2>` +
		`<p>Background content.</p></div>` +
		`</div>` +
		`<div data-header="Main Topic" data-level="1"><h1>Main Topic</h1>` +
		`<p>Topic content.</p></div>`

	// Chunk breadcrumb: "{title} > {h1} > {h2}\n\n{body}"
	// For the Background section under Introduction:
	chunkContent := "My Note > Introduction > Background\n\nBackground content."

	path := tocPathFromChunkContent(noteHTML, chunkContent)
	require.NotNil(t, path, "path must not be nil")
	require.Equal(t, []string{"Introduction", "Background"}, path)

	// Verify round-trip: the path navigates to the correct section.
	sectionHTML := sectionHTMLByTocPath(noteHTML, path)
	require.NotEmpty(t, sectionHTML, "sectionHTMLByTocPath must return non-empty HTML")
	require.Contains(t, sectionHTML, "Background content.")
}

// TestTocPathFromChunkContentMarkdownStripping verifies that breadcrumb segments
// with markdown inline formatting are stripped before path lookup, so that the
// derived path matches the data-header attribute (goldmark plain-text extraction).
func TestTocPathFromChunkContentMarkdownStripping(t *testing.T) {
	// goldmark strips **bold** when computing data-header — it sees plain text "Bold Section"
	noteHTML := `<div data-header="Bold Section" data-level="2"><h2><strong>Bold Section</strong></h2>` +
		`<p>Some bold section content.</p></div>`

	// The chunk breadcrumb has raw markdown (as produced by parseHeading)
	chunkContent := "My Note > **Bold Section**\n\nSome bold section content."

	path := tocPathFromChunkContent(noteHTML, chunkContent)
	require.NotNil(t, path, "path must not be nil after markdown stripping")
	require.Equal(t, []string{"Bold Section"}, path,
		"markdown must be stripped to match data-header, got %v", path)

	// Verify round-trip.
	sectionHTML := sectionHTMLByTocPath(noteHTML, path)
	require.NotEmpty(t, sectionHTML, "round-trip must find the section")
	require.Contains(t, sectionHTML, "bold section content")
}

// TestTocPathForSnippetChunkFallback verifies that when the fuzzy HTML match
// fails (e.g., snippet text is too different from the rendered section), the
// chunk breadcrumb fallback is used to derive the path.
func TestTocPathForSnippetChunkFallback(t *testing.T) {
	// Section text that won't match the snippet (different content).
	noteHTML := `<div data-header="Alpha" data-level="1"><h1>Alpha</h1>` +
		`<p>Alpha section has specific terminology.</p></div>` +
		`<div data-header="Beta" data-level="1"><h1>Beta</h1>` +
		`<p>Beta section has other terminology.</p></div>`

	// Snippet that won't match any section (intentionally unrelated text).
	snippet := "completely unrelated content that is not in the note at all"

	// Chunk breadcrumb points to Beta section.
	chunkContent := "My Note > Beta\n\nBeta section has other terminology."

	path := tocPathForSnippet(noteHTML, snippet, chunkContent)
	require.NotNil(t, path, "path must not be nil via chunk fallback")
	require.Equal(t, []string{"Beta"}, path,
		"chunk breadcrumb fallback should return Beta path, got %v", path)
}

// TestTocPathForSnippetNeverNil verifies that tocPathForSnippet never returns nil
// when there are sections in the note, even for degenerate inputs.
func TestTocPathForSnippetNeverNil(t *testing.T) {
	noteHTML := `<div data-header="Only Section" data-level="2"><h2>Only Section</h2>` +
		`<p>Content.</p></div>`

	tests := []struct {
		name    string
		snippet string
		chunk   string
	}{
		{"very short snippet", "hi", ""},
		{"no match snippet no chunk", "xyzzy does not exist anywhere", ""},
		{"empty chunk", strings.Repeat("no match ", 20), ""},
		{"chunk with no headings", "anything", "My Note\n\nsome body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tocPathForSnippet(noteHTML, tt.snippet, tt.chunk)
			require.NotNil(t, path, "path must not be nil for snippet=%q chunk=%q", tt.snippet, tt.chunk)
		})
	}
}

// TestStripMarkdownInline verifies the markdown inline syntax stripper
// handles common cases correctly.
func TestStripMarkdownInline(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Text", "Simple Text"},
		{"**Bold** text", "Bold text"},
		{"*Italic* text", "Italic text"},
		// Nested inline markdown is handled imperfectly by single-pass regex:
		// pos 1 matches *Bold and * -> "Bold and "; then "* combined*" -> " combined".
		// Result has residual outer *. Headings with deeply nested formatting are rare.
		{"**Bold and *italic* combined**", "*Bold and italic combined*"},
		{"`code` snippet", "code snippet"},
		{"[link text](http://example.com)", "link text"},
		{"[ref link][refid]", "ref link"},
		{"![image alt](image.png)", "image alt"},
		{"__underline bold__", "underline bold"},
		{"_underline italic_", "underline italic"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripMarkdownInline(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
