package markdownv2_test

import (
	"os"
	"strings"
	"testing"
	"time"
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

func TestHTMLContentEmoji(t *testing.T) {
	obsidianMarkdown, err := os.ReadFile("obsidian_emoji.md")
	require.NoError(t, err)

	telegramHTML, err := os.ReadFile("telegram_emoji.html")
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
		{
			name:     "autolink (bare URL)",
			markdown: "Watch https://www.youtube.com/watch?v=0S8DY7k_7Yw here",
			expected: `Watch <a href="https://www.youtube.com/watch?v=0S8DY7k_7Yw">https://www.youtube.com/watch?v=0S8DY7k_7Yw</a> here`,
		},
		{
			name:     "autolink youtube",
			markdown: "https://www.youtube.com/watch?v=abc123",
			expected: `<a href="https://www.youtube.com/watch?v=abc123">https://www.youtube.com/watch?v=abc123</a>`,
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
		{
			name: "paragraph followed by unordered list",
			markdown: `Unordered list:
- First item
- Second item
- Third item`,
			expected: `Unordered list:
- First item
- Second item
- Third item`,
		},
		{
			name: "paragraph followed by ordered list",
			markdown: `Numbered list:
1. First step
2. Second step
3. Third step`,
			expected: `Numbered list:
1. First step
2. Second step
3. Third step`,
		},
		{
			name: "multiple paragraphs with lists",
			markdown: `Unordered list:
- First item
- Second item
- Third item

Numbered list:
1. First step
2. Second step
3. Third step`,
			expected: `Unordered list:
- First item
- Second item
- Third item

Numbered list:
1. First step
2. Second step
3. Third step`,
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
		{
			name:     "unpublished link with PublishAt",
			markdown: "See [[future-note]] for details",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				publishAt := time.Date(2025, 11, 5, 14, 30, 0, 0, time.UTC)
				return &markdownv2.LinkResolverResult{
					URL:       "",
					Label:     "future-note",
					PublishAt: &publishAt,
				}, nil
			},
			expected: "See <u>future-note</u> for details\n\n—————————\n🔜 Скоро выйдут:\n• <u>future-note</u> — 5 ноября, 14:30\n\n📬 Подпишитесь, чтобы не пропустить",
			warnings: 0,
		},
		{
			name:     "multiple unpublished links",
			markdown: "Read [[first-post]] and [[second-post]]",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				var publishAt time.Time
				if target == "first-post" {
					publishAt = time.Date(2025, 11, 5, 14, 30, 0, 0, time.UTC)
				} else {
					publishAt = time.Date(2025, 11, 7, 10, 0, 0, 0, time.UTC)
				}
				return &markdownv2.LinkResolverResult{
					URL:       "",
					Label:     target,
					PublishAt: &publishAt,
				}, nil
			},
			expected: "Read <u>first-post</u> and <u>second-post</u>\n\n—————————\n🔜 Скоро выйдут:\n• <u>first-post</u> — 5 ноября, 14:30\n• <u>second-post</u> — 7 ноября, 10:00\n\n📬 Подпишитесь, чтобы не пропустить",
			warnings: 0,
		},
		{
			name:     "mixed published and unpublished links",
			markdown: "Read [[published]] and [[unpublished]]",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				if target == "published" {
					return &markdownv2.LinkResolverResult{
						URL:   "https://example.com/published",
						Label: "published",
					}, nil
				}
				publishAt := time.Date(2025, 11, 5, 14, 30, 0, 0, time.UTC)
				return &markdownv2.LinkResolverResult{
					URL:       "",
					Label:     "unpublished",
					PublishAt: &publishAt,
				}, nil
			},
			expected: "Read <a href=\"https://example.com/published\">published</a> and <u>unpublished</u>\n\n—————————\n🔜 Скоро выйдут:\n• <u>unpublished</u> — 5 ноября, 14:30\n\n📬 Подпишитесь, чтобы не пропустить",
			warnings: 0,
		},
		{
			name:     "unpublished link with custom label",
			markdown: "Read [[future-note|Custom Label]]",
			linkResolver: func(target string) (*markdownv2.LinkResolverResult, error) {
				publishAt := time.Date(2025, 12, 25, 18, 0, 0, 0, time.UTC)
				return &markdownv2.LinkResolverResult{
					URL:       "",
					Label:     "Custom Label",
					PublishAt: &publishAt,
				}, nil
			},
			expected: "Read <u>Custom Label</u>\n\n—————————\n🔜 Скоро выйдут:\n• <u>Custom Label</u> — 25 декабря, 18:00\n\n📬 Подпишитесь, чтобы не пропустить",
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

func TestHTMLCustomEmoji(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected string
		warnings int
	}{
		{
			name:     "tg://emoji format",
			markdown: "Hello ![👋](tg://emoji?id=5368324170671202286) world",
			expected: `Hello <tg-emoji emoji-id="5368324170671202286">👋</tg-emoji> world`,
			warnings: 0,
		},
		{
			name:     "tg://emoji format with text alt",
			markdown: "![emoji](tg://emoji?id=5395478998320946882)",
			expected: `<tg-emoji emoji-id="5395478998320946882">emoji</tg-emoji>`,
			warnings: 0,
		},
		{
			name:     "ce.trip2g.com format",
			markdown: "Hello ![](https://ce.trip2g.com/5460736117236048513.webp) world",
			expected: `Hello <tg-emoji emoji-id="5460736117236048513"></tg-emoji> world`,
			warnings: 0,
		},
		{
			name:     "ce.trip2g.com with alt text",
			markdown: "Hello ![emoji](https://ce.trip2g.com/5460736117236048513.webp) world",
			expected: `Hello <tg-emoji emoji-id="5460736117236048513">emoji</tg-emoji> world`,
			warnings: 0,
		},
		{
			name:     "ce.trip2g.com with size suffix in alt",
			markdown: "Hello ![🔥|20x20](https://ce.trip2g.com/5460736117236048513.webp) world",
			expected: `Hello <tg-emoji emoji-id="5460736117236048513">🔥</tg-emoji> world`,
			warnings: 0,
		},
		{
			name:     "multiple ce.trip2g.com emojis",
			markdown: "![](https://ce.trip2g.com/111.webp)![](https://ce.trip2g.com/222.webp)",
			expected: `<tg-emoji emoji-id="111"></tg-emoji><tg-emoji emoji-id="222"></tg-emoji>`,
			warnings: 0,
		},
		{
			name:     "mixed emoji formats",
			markdown: "![👋](tg://emoji?id=123) and ![](https://ce.trip2g.com/456.webp)",
			expected: `<tg-emoji emoji-id="123">👋</tg-emoji> and <tg-emoji emoji-id="456"></tg-emoji>`,
			warnings: 0,
		},
		{
			name:     "unsupported image source",
			markdown: "![](https://example.com/image.png)",
			expected: ``,
			warnings: 2, // Both Enclave and Image nodes are created
		},
		{
			name:     "ce.trip2g.com wrong extension",
			markdown: "![](https://ce.trip2g.com/123.png)",
			expected: ``,
			warnings: 2, // Both Enclave and Image nodes are created
		},
		{
			name:     "ce.trip2g.com non-numeric id",
			markdown: "![](https://ce.trip2g.com/abc.webp)",
			expected: ``,
			warnings: 2, // Both Enclave and Image nodes are created
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

			require.Len(t, res.Warnings, tt.warnings)
			require.Equal(t, tt.expected, res.Content)
		})
	}
}

// TestHTMLMediaFilesBlankLines tests that excessive blank lines are removed
// when media files (embedded wikilinks) are in the middle of content.
func TestHTMLMediaFilesBlankLines(t *testing.T) {
	markdown := `This post contains multiple media files.

The post type is: **media_group**

![[telegram_photo.png]]
![[telegram_photo2.jpg]]
![[telegram_video.mp4]]

Features:
- Multiple photos and videos (up to 10)
- Caption can be edited`

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Content: []byte(`---
free: true
title: "Media Test"
---
` + markdown),
		}},
		Log: &logger.TestLogger{},
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	convertor := markdownv2.HTMLConverter{}
	res := convertor.Process(nvs.List[0])

	// Should not have more than 2 newlines in a row (no triple \n\n\n)
	require.NotContains(t, res.Content, "\n\n\n", "Should not have excessive blank lines")

	// Should have proper spacing between paragraphs (exactly 2 newlines)
	expected := `This post contains multiple media files.

The post type is: <b>media_group</b>

Features:
- Multiple photos and videos (up to 10)
- Caption can be edited`

	require.Equal(t, expected, res.Content)
}
