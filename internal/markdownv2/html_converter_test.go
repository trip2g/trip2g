package markdownv2_test

import (
	"os"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/markdownv2"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func TestHTMLContent(t *testing.T) {
	obsidianMarkdown, err := os.ReadFile("obsidian.md")
	require.NoError(t, err)

	telegramHTML, err := os.ReadFile("telegram.html")
	require.NoError(t, err)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Content: []byte(`---
free: true
title: "Sample Note"
---
` + string(obsidianMarkdown)),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	nvs.List[0].Ast().Dump(nvs.List[0].Content, 2)

	convertor := markdownv2.HTMLConverter{}

	res := convertor.Process(nvs.List[0])

	require.Empty(t, res.Warnings)
	require.Equal(t, strings.Trim(string(telegramHTML), "\n"), res.Content)
}

func TestHTMLNewLines(t *testing.T) {
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Content: []byte(`---
free: true
title: "Sample Note"
---
**Hello World**

A first paragraph.

A second paragraph
with 2 new lines above.

A third paragraph.`),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	convertor := markdownv2.HTMLConverter{}

	res := convertor.Process(nvs.List[0])

	expectedHTML := `<b>Hello World</b>

A first paragraph.

A second paragraph
with 2 new lines above.

A third paragraph.`

	require.Empty(t, res.Warnings)
	require.Equal(t, expectedHTML, res.Content)
}

func TestHTMLRegularLinks(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "simple link",
			markdown: "Check [this link](https://example.com) here",
			expected: `Check <a href="https://example.com">this link</a> here`,
		},
		{
			name:     "link with special chars",
			markdown: "See [docs](https://example.com/path?foo=bar&baz=qux)",
			expected: `See <a href="https://example.com/path?foo=bar&amp;baz=qux">docs</a>`,
		},
		{
			name:     "link in bold text",
			markdown: "**Bold [link](https://example.com) text**",
			expected: `<b>Bold <a href="https://example.com">link</a> text</b>`,
		},
		{
			name:     "multiple links",
			markdown: "[First](https://one.com) and [Second](https://two.com)",
			expected: `<a href="https://one.com">First</a> and <a href="https://two.com">Second</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdOptions := mdloader.Options{
				Sources: []mdloader.SourceFile{{
					Content: []byte(`---
free: true
title: "Test"
---
` + tt.markdown),
				}},
				Log:     &logger.TestLogger{},
				Version: "latest",
			}

			nvs, err := mdloader.Load(mdOptions)
			require.NoError(t, err)

			convertor := markdownv2.HTMLConverter{}
			res := convertor.Process(nvs.List[0])

			require.Empty(t, res.Warnings)
			require.Equal(t, tt.expected, res.Content)
		})
	}
}

func TestHTMLList(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name: "simple unordered list",
			markdown: `- first item
- second item
- third item`,
			expected: `- first item
- second item
- third item`,
		},
		{
			name: "simple ordered list",
			markdown: `1. first item
2. second item
3. third item`,
			expected: `1. first item
2. second item
3. third item`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdOptions := mdloader.Options{
				Sources: []mdloader.SourceFile{{
					Content: []byte(`---
free: true
title: "Test"
---
` + tt.markdown),
				}},
				Log:     &logger.TestLogger{},
				Version: "latest",
			}

			nvs, err := mdloader.Load(mdOptions)
			require.NoError(t, err)

			convertor := markdownv2.HTMLConverter{}
			res := convertor.Process(nvs.List[0])

			require.Empty(t, res.Warnings)
			require.Equal(t, tt.expected, res.Content)
		})
	}
}

func TestHTMLWikilinks(t *testing.T) {
	tests := []struct {
		name         string
		markdown     string
		linkResolver markdownv2.LinkResolver
		expected     string
		warnings     int
	}{
		{
			name:     "wikilink with resolver",
			markdown: "See [[internal-note]] for details",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				if target == "internal-note" {
					return &markdownv2.LinkResolverResult{
						URL:   "https://example.com/notes/internal-note",
						Label: "internal-note",
					}, nil
				}
				return nil, nil
			},
			expected: `See <a href="https://example.com/notes/internal-note">internal-note</a> for details`,
			warnings: 0,
		},
		{
			name:     "wikilink with fragment (custom text)",
			markdown: "See [[internal-note|Custom Text]] here",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				return &markdownv2.LinkResolverResult{
					URL:   "https://example.com/notes/" + target,
					Label: "Custom Text",
				}, nil
			},
			expected: `See <a href="https://example.com/notes/internal-note">Custom Text</a> here`,
			warnings: 0,
		},
		{
			name:     "wikilink with resolver error",
			markdown: "See [[missing-note]] here",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				return nil, &markdownv2.LinkResolverError{Target: target, Reason: "not found"}
			},
			expected: `See  here`,
			warnings: 1,
		},
		{
			name:     "multiple wikilinks",
			markdown: "Read [[first]] and [[second]]",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				return &markdownv2.LinkResolverResult{
					URL:   "https://example.com/" + target,
					Label: target,
				}, nil
			},
			expected: `Read <a href="https://example.com/first">first</a> and <a href="https://example.com/second">second</a>`,
			warnings: 0,
		},
		{
			name:     "wikilink in blockquote",
			markdown: "> Check [[note]] here",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				return &markdownv2.LinkResolverResult{
					URL:   "https://example.com/" + target,
					Label: target,
				}, nil
			},
			expected: `<blockquote>Check <a href="https://example.com/note">note</a> here</blockquote>`,
			warnings: 0,
		},
		{
			name:     "wikilink with fragment in blockquote",
			markdown: "> Read [[first|Fragment Text]] here",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				return &markdownv2.LinkResolverResult{
					URL:   "https://example.com/" + target,
					Label: "Fragment Text",
				}, nil
			},
			expected: `<blockquote>Read <a href="https://example.com/first">Fragment Text</a> here</blockquote>`,
			warnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdOptions := mdloader.Options{
				Sources: []mdloader.SourceFile{{
					Content: []byte(`---
free: true
title: "Test"
---
` + tt.markdown),
				}},
				Log:     &logger.TestLogger{},
				Version: "latest",
			}

			nvs, err := mdloader.Load(mdOptions)
			require.NoError(t, err)

			convertor := markdownv2.HTMLConverter{}
			convertor.SetLinkResolver(tt.linkResolver)
			res := convertor.Process(nvs.List[0])

			require.Len(t, res.Warnings, tt.warnings)
			require.Equal(t, tt.expected, res.Content)
		})
	}
}
