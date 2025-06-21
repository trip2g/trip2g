package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestPerparePermalink(t *testing.T) {
	n := NoteView{Path: "/Моя заметка + другая заметка.md"}
	n.PreparePermalink()

	require.Equal(t, "/moya_zametka_drugaya_zametka", n.Permalink)

	n.Path = "Моя особая + страница"
	n.PreparePermalink()

	require.Equal(t, "/moya_osobaya_stranica", n.Permalink)
}

func TestExtractReadingTime(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		rawMeta     map[string]interface{}
		expectedMin int
	}{
		{
			name:        "empty content",
			content:     "",
			rawMeta:     make(map[string]interface{}),
			expectedMin: 0,
		},
		{
			name:        "short content",
			content:     "Hello world",
			rawMeta:     make(map[string]interface{}),
			expectedMin: 1, // minimum is 1 minute
		},
		{
			name: "content with markdown",
			content: `# Header
This is a **bold** text with [link](https://example.com) and [[wikilink|display text]].

` + "```go\ncode block\n```" + `

- List item 1
- List item 2

> Blockquote text

Some more text to reach word count.`,
			rawMeta:     make(map[string]interface{}),
			expectedMin: 1,
		},
		{
			name:        "meta override as int",
			content:     "Some content here",
			rawMeta:     map[string]interface{}{"reading_time": 5},
			expectedMin: 5,
		},
		{
			name:        "meta override as float",
			content:     "Some content here",
			rawMeta:     map[string]interface{}{"reading_time": 3.0},
			expectedMin: 3,
		},
		{
			name:        "meta override as string",
			content:     "Some content here",
			rawMeta:     map[string]interface{}{"reading_time": "7"},
			expectedMin: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{
				Content: []byte(tt.content),
				RawMeta: tt.rawMeta,
			}

			n.extractReadingTime()

			require.Equal(t, tt.expectedMin, n.ReadingTime)
		})
	}
}

func TestExtractReadingComplexity(t *testing.T) {
	tests := []struct {
		name         string
		rawMeta      map[string]interface{}
		expectedComp int
		expectError  bool
	}{
		{
			name:         "no complexity meta",
			rawMeta:      make(map[string]interface{}),
			expectedComp: 0, // default is easy
			expectError:  false,
		},
		{
			name:         "complexity as int 0",
			rawMeta:      map[string]interface{}{"complexity": 0},
			expectedComp: 0,
			expectError:  false,
		},
		{
			name:         "complexity as int 1",
			rawMeta:      map[string]interface{}{"complexity": 1},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as int 2",
			rawMeta:      map[string]interface{}{"complexity": 2},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "complexity as float 1.0",
			rawMeta:      map[string]interface{}{"complexity": 1.0},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as string easy",
			rawMeta:      map[string]interface{}{"complexity": "easy"},
			expectedComp: 0,
			expectError:  false,
		},
		{
			name:         "complexity as string medium",
			rawMeta:      map[string]interface{}{"complexity": "medium"},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as string hard",
			rawMeta:      map[string]interface{}{"complexity": "hard"},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "complexity as string 0",
			rawMeta:      map[string]interface{}{"complexity": "0"},
			expectedComp: 0,
			expectError:  false,
		},
		{
			name:         "complexity as string 1",
			rawMeta:      map[string]interface{}{"complexity": "1"},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as string 2",
			rawMeta:      map[string]interface{}{"complexity": "2"},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "reading_complexity key",
			rawMeta:      map[string]interface{}{"reading_complexity": "hard"},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "invalid int complexity",
			rawMeta:      map[string]interface{}{"complexity": 5},
			expectedComp: 0,
			expectError:  true,
		},
		{
			name:         "invalid string complexity",
			rawMeta:      map[string]interface{}{"complexity": "invalid"},
			expectedComp: 0,
			expectError:  true,
		},
		{
			name:         "invalid type complexity",
			rawMeta:      map[string]interface{}{"complexity": []string{"easy"}},
			expectedComp: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{
				RawMeta: tt.rawMeta,
			}

			err := n.extractReadingComplexity()

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expectedComp, n.ReadingComplexity)
		})
	}
}

func TestExtractHeadings(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedHeadings NoteViewHeadings
	}{
		{
			name:             "no headings",
			content:          "Just some regular text content.",
			expectedHeadings: nil,
		},
		{
			name: "single heading",
			content: `# Main Title
Some content here.`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Main Title", Level: 1, ID: "main_title_h"},
			},
		},
		{
			name: "multiple headings different levels",
			content: `# Chapter 1
Some intro text.

## Section 1.1
More content.

### Subsection 1.1.1
Even more content.

## Section 1.2
Final content.`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Chapter 1", Level: 1, ID: "chapter_1_h"},
				{Text: "Section 1.1", Level: 2, ID: "section_1_1_h"},
				{Text: "Subsection 1.1.1", Level: 3, ID: "subsection_1_1_1_h"},
				{Text: "Section 1.2", Level: 2, ID: "section_1_2_h"},
			},
		},
		{
			name: "headings with formatting",
			content: `# **Bold** Heading
## *Italic* Heading
### [Link](http://example.com) Heading`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Bold Heading", Level: 1, ID: "bold_heading_h"},
				{Text: "Italic Heading", Level: 2, ID: "italic_heading_h"},
				{Text: "Link Heading", Level: 3, ID: "link_heading_h"},
			},
		},
		{
			name: "headings with gaps get normalized",
			content: `## Second Level
#### Fourth Level
###### Sixth Level`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Second Level", Level: 1, ID: "second_level_h"},
				{Text: "Fourth Level", Level: 2, ID: "fourth_level_h"},
				{Text: "Sixth Level", Level: 3, ID: "sixth_level_h"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to parse the markdown to create an AST
			parser := goldmark.New()
			doc := parser.Parser().Parse(text.NewReader([]byte(tt.content)))

			n := &NoteView{
				Content: []byte(tt.content),
				ast:     doc,
			}

			n.extractHeadingsAndGenerateIDs()

			require.Equal(t, tt.expectedHeadings, n.Headings)
		})
	}
}

func TestNoteViewHeadings_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    NoteViewHeadings
		expected NoteViewHeadings
	}{
		{
			name:     "empty headings",
			input:    NoteViewHeadings{},
			expected: NoteViewHeadings{},
		},
		{
			name: "already normalized levels 1,2,3",
			input: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H2", Level: 2, ID: "h2"},
				{Text: "H3", Level: 3, ID: "h3"},
			},
			expected: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H2", Level: 2, ID: "h2"},
				{Text: "H3", Level: 3, ID: "h3"},
			},
		},
		{
			name: "only level 2 becomes level 1",
			input: NoteViewHeadings{
				{Text: "H2a", Level: 2, ID: "h2a"},
				{Text: "H2b", Level: 2, ID: "h2b"},
			},
			expected: NoteViewHeadings{
				{Text: "H2a", Level: 1, ID: "h2a"},
				{Text: "H2b", Level: 1, ID: "h2b"},
			},
		},
		{
			name: "levels 2 and 6 become 1 and 2",
			input: NoteViewHeadings{
				{Text: "H2", Level: 2, ID: "h2"},
				{Text: "H6a", Level: 6, ID: "h6a"},
				{Text: "H6b", Level: 6, ID: "h6b"},
			},
			expected: NoteViewHeadings{
				{Text: "H2", Level: 1, ID: "h2"},
				{Text: "H6a", Level: 2, ID: "h6a"},
				{Text: "H6b", Level: 2, ID: "h6b"},
			},
		},
		{
			name: "levels 1,3,5 become 1,2,3",
			input: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H3", Level: 3, ID: "h3"},
				{Text: "H5", Level: 5, ID: "h5"},
				{Text: "H3b", Level: 3, ID: "h3b"},
			},
			expected: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H3", Level: 2, ID: "h3"},
				{Text: "H5", Level: 3, ID: "h5"},
				{Text: "H3b", Level: 2, ID: "h3b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since Normalize modifies in place
			input := make(NoteViewHeadings, len(tt.input))
			copy(input, tt.input)

			input.Normalize()

			require.Equal(t, tt.expected, input)
		})
	}
}
