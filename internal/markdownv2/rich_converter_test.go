package markdownv2_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/markdownv2"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

// loadRichNote parses markdown through the real loader, so the converter sees
// exactly the AST the publish path would hand it.
func loadRichNote(t *testing.T, markdown string) *model.NoteView {
	t.Helper()

	nvs, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte("---\ntitle: \"Sample Note\"\n---\n" + markdown),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	})
	require.NoError(t, err)

	return nvs.List[0]
}

func convertRich(t *testing.T, markdown string) markdownv2.RichConverterResult {
	t.Helper()

	c := markdownv2.RichConverter{}
	return c.Process(loadRichNote(t, markdown))
}

func plain(s string) *tgrich.RichText {
	return &tgrich.RichText{Text: s}
}

func TestRichHeadings(t *testing.T) {
	res := convertRich(t, "# One\n\n## Two\n\n### Three\n\n#### Four\n\n##### Five\n\n###### Six")

	require.Empty(t, res.Losses)
	require.Len(t, res.Blocks, 6)

	for i, want := range []string{"One", "Two", "Three", "Four", "Five", "Six"} {
		require.Equal(t, tgrich.BlockHeading, res.Blocks[i].Type)
		require.Equal(t, i+1, res.Blocks[i].Size)
		require.Equal(t, want, res.Blocks[i].Text.PlainText())
	}

	// The generated ids are recorded but never sent: measured, the server
	// echoes back no anchor field on a heading under any spelling.
	require.Len(t, res.Anchors, 6)
	require.Contains(t, res.Anchors, "one")
}

func TestRichParagraphInlineRuns(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     tgrich.RichText
	}{
		{
			name:     "plain text",
			markdown: "hello",
			want:     tgrich.RichText{Text: "hello"},
		},
		{
			name:     "bold",
			markdown: "**bold**",
			want:     tgrich.RichText{Children: []tgrich.RichText{{Text: "bold", Bold: true}}},
		},
		{
			name:     "italic",
			markdown: "*it*",
			want:     tgrich.RichText{Children: []tgrich.RichText{{Text: "it", Italic: true}}},
		},
		{
			name:     "strikethrough",
			markdown: "~~gone~~",
			want:     tgrich.RichText{Children: []tgrich.RichText{{Text: "gone", Strike: true}}},
		},
		{
			name:     "code span",
			markdown: "`x := 1`",
			want:     tgrich.RichText{Children: []tgrich.RichText{{Text: "x := 1", Code: true}}},
		},
		{
			name:     "highlight becomes marked, not a spoiler",
			markdown: "==hot==",
			want:     tgrich.RichText{Children: []tgrich.RichText{{Text: "hot", Marked: true}}},
		},
		{
			name:     "raw u tags become underline",
			markdown: "<u>under</u>",
			want:     tgrich.RichText{Children: []tgrich.RichText{{Text: "under", Underline: true}}},
		},
		{
			name:     "link",
			markdown: "[site](https://example.com)",
			want: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "site", URL: "https://example.com"},
			}},
		},
		{
			name:     "autolink",
			markdown: "<https://example.com>",
			want: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "https://example.com", URL: "https://example.com"},
			}},
		},
		{
			name:     "nested marks",
			markdown: "**bold *and italic***",
			want: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "bold ", Bold: true},
				{Text: "and italic", Bold: true, Italic: true},
			}},
		},
		{
			name:     "mixed run keeps document order",
			markdown: "a **b** c",
			want: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "a "},
				{Text: "b", Bold: true},
				{Text: " c"},
			}},
		},
		{
			name:     "soft line break becomes a newline run",
			markdown: "one\ntwo",
			want: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "one"},
				{Text: "\n"},
				{Text: "two"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := convertRich(t, tt.markdown)

			require.Empty(t, res.Losses)
			require.Len(t, res.Blocks, 1)
			require.Equal(t, tgrich.BlockParagraph, res.Blocks[0].Type)
			require.Equal(t, tt.want, *res.Blocks[0].Text)
		})
	}
}

func TestRichLists(t *testing.T) {
	t.Run("unordered", func(t *testing.T) {
		res := convertRich(t, "- one\n- two")

		require.Empty(t, res.Losses)
		require.Len(t, res.Blocks, 1)

		list := res.Blocks[0]
		require.Equal(t, tgrich.BlockList, list.Type)
		require.Len(t, list.Items, 2)
		require.Nil(t, list.Items[0].Checked)
		require.Equal(t, plain("one"), list.Items[0].Blocks[0].Text)
	})

	// Ordering is not carried: measured, the server ignores every spelling of
	// it and labels every item with a bullet, so an ordered list renders as
	// bullets and the block says so rather than pretending otherwise.
	t.Run("ordered list keeps its items but loses its numbering", func(t *testing.T) {
		res := convertRich(t, "3. three\n4. four")

		list := res.Blocks[0]
		require.Equal(t, tgrich.BlockList, list.Type)
		require.Len(t, list.Items, 2)
	})

	t.Run("nested list nests as a block inside its item", func(t *testing.T) {
		res := convertRich(t, "- one\n    - inner\n        - deepest")

		outer := res.Blocks[0]
		require.Len(t, outer.Items, 1)
		require.Len(t, outer.Items[0].Blocks, 2)

		inner := outer.Items[0].Blocks[1]
		require.Equal(t, tgrich.BlockList, inner.Type)
		require.Equal(t, plain("inner"), inner.Items[0].Blocks[0].Text)

		deepest := inner.Items[0].Blocks[1]
		require.Equal(t, tgrich.BlockList, deepest.Type)
		require.Equal(t, plain("deepest"), deepest.Items[0].Blocks[0].Text)
	})

	t.Run("task list carries checkbox state", func(t *testing.T) {
		res := convertRich(t, "- [ ] todo\n- [x] done")

		require.Empty(t, res.Losses)

		list := res.Blocks[0]
		require.Len(t, list.Items, 2)

		require.NotNil(t, list.Items[0].Checked)
		require.False(t, *list.Items[0].Checked)
		require.Equal(t, plain("todo"), list.Items[0].Blocks[0].Text)

		require.NotNil(t, list.Items[1].Checked)
		require.True(t, *list.Items[1].Checked)
		require.Equal(t, plain("done"), list.Items[1].Blocks[0].Text)
	})
}

func TestRichBlockquote(t *testing.T) {
	res := convertRich(t, "> quoted\n>\n> more")

	require.Empty(t, res.Losses)
	require.Len(t, res.Blocks, 1)

	quote := res.Blocks[0]
	require.Equal(t, tgrich.BlockQuote, quote.Type)
	require.Nil(t, quote.Summary)
	require.Len(t, quote.Blocks, 2)
	require.Equal(t, plain("quoted"), quote.Blocks[0].Text)
	require.Equal(t, plain("more"), quote.Blocks[1].Text)
}

func TestRichCallouts(t *testing.T) {
	// Every callout becomes a details block: a blockquote takes only blocks,
	// so it has nowhere to put the callout's title. A non-foldable callout is
	// therefore an open details block, which keeps the title on screen.
	tests := []struct {
		name        string
		markdown    string
		wantSummary string
		wantOpen    bool
	}{
		{
			name:        "plain callout is an open details block",
			markdown:    "> [!note]\n> body",
			wantSummary: "Note",
			wantOpen:    true,
		},
		{
			name:        "custom title wins over the type",
			markdown:    "> [!warning] Careful\n> body",
			wantSummary: "Careful",
			wantOpen:    true,
		},
		{
			name:        "foldable callout collapses",
			markdown:    "> [!tip]- Hidden\n> body",
			wantSummary: "Hidden",
			wantOpen:    false,
		},
		{
			name:        "foldable-expanded callout stays open",
			markdown:    "> [!tip]+ Shown\n> body",
			wantSummary: "Shown",
			wantOpen:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := convertRich(t, tt.markdown)

			require.Empty(t, res.Losses)
			require.Len(t, res.Blocks, 1)

			block := res.Blocks[0]
			require.Equal(t, tgrich.BlockDetails, block.Type)
			require.Equal(t, tt.wantOpen, block.IsOpen)
			require.NotNil(t, block.Summary)
			require.Equal(t, tt.wantSummary, block.Summary.PlainText())
			require.Len(t, block.Blocks, 1)
			require.Equal(t, plain("body"), block.Blocks[0].Text)
		})
	}
}

func TestRichTable(t *testing.T) {
	res := convertRich(t, "| a | b | c |\n|:--|--:|:-:|\n| 1 | 2 | 3 |")

	require.Empty(t, res.Losses)
	require.Len(t, res.Blocks, 1)

	// Alignment is per cell: the wire form has no column descriptor at all, so
	// the column's alignment is stamped onto each of its cells.
	table := res.Blocks[0]
	require.Equal(t, tgrich.BlockTable, table.Type)
	require.Len(t, table.Cells, 2)

	require.Equal(t, []tgrich.Align{tgrich.AlignLeft, tgrich.AlignRight, tgrich.AlignCenter},
		[]tgrich.Align{table.Cells[0][0].Align, table.Cells[0][1].Align, table.Cells[0][2].Align})

	require.True(t, table.Cells[0][0].IsHeader)
	require.False(t, table.Cells[1][0].IsHeader)
	require.Equal(t, "a", table.Cells[0][0].Text.PlainText())
	require.Equal(t, "3", table.Cells[1][2].Text.PlainText())
}

func TestRichTableWithoutExplicitAlignment(t *testing.T) {
	res := convertRich(t, "| a |\n|---|\n| 1 |")

	// No declared alignment leaves the field empty and the server defaults it.
	require.Equal(t, tgrich.Align(""), res.Blocks[0].Cells[0][0].Align)
}

func TestRichFencedCode(t *testing.T) {
	t.Run("keeps the language and expands tabs", func(t *testing.T) {
		res := convertRich(t, "```go\nfunc x() {\n\treturn\n}\n```")

		require.Empty(t, res.Losses)
		require.Len(t, res.Blocks, 1)
		require.Equal(t, tgrich.BlockPre, res.Blocks[0].Type)
		require.Equal(t, "go", res.Blocks[0].Language)
		// Telegram collapses a literal tab to one space, so expand before send.
		require.Equal(t, "func x() {\n    return\n}", res.Blocks[0].Text.PlainText())
	})

	t.Run("without a language", func(t *testing.T) {
		res := convertRich(t, "```\nplain\n```")

		require.Empty(t, res.Blocks[0].Language)
		require.Equal(t, "plain", res.Blocks[0].Text.PlainText())
	})
}

func TestRichThematicBreak(t *testing.T) {
	res := convertRich(t, "a\n\n---\n\nb")

	require.Empty(t, res.Losses)
	require.Len(t, res.Blocks, 3)
	require.Equal(t, tgrich.BlockDivider, res.Blocks[1].Type)
}

func TestRichImages(t *testing.T) {
	t.Run("standalone https image becomes a photo block", func(t *testing.T) {
		res := convertRich(t, "![alt](https://example.com/a.png)")

		require.Empty(t, res.Losses)
		require.Len(t, res.Blocks, 1)
		require.Equal(t, tgrich.BlockPhoto, res.Blocks[0].Type)
		require.Equal(t, "https://example.com/a.png", res.Blocks[0].Photo.URL)
		require.Equal(t, "alt", res.Blocks[0].Caption.PlainText())
	})

	t.Run("image with no alt has no caption", func(t *testing.T) {
		res := convertRich(t, "![](https://example.com/a.png)")

		require.Nil(t, res.Blocks[0].Caption)
	})

	t.Run("video extension becomes a video block", func(t *testing.T) {
		res := convertRich(t, "![](https://example.com/a.mp4)")

		require.Equal(t, tgrich.BlockVideo, res.Blocks[0].Type)
		require.Equal(t, "https://example.com/a.mp4", res.Blocks[0].Video.URL)
	})

	// Only https was documented and measured. An http URL must not be passed
	// through as if it were fetchable; it goes to the asset resolver like any
	// other unusable destination, and is a loss when nothing resolves it.
	t.Run("an http image is not a usable media source", func(t *testing.T) {
		res := convertRich(t, "![](http://example.com/a.png)")

		require.Empty(t, res.Blocks)
		require.Equal(t, []markdownv2.RichLoss{
			{Kind: markdownv2.LossUnresolvedMedia, Node: "Enclave", Detail: "http://example.com/a.png"},
		}, res.Losses)
	})

	t.Run("a local asset needs an app-layer URL and is a typed loss", func(t *testing.T) {
		res := convertRich(t, "![](assets/a.png)")

		require.Empty(t, res.Blocks)
		require.Equal(t, []markdownv2.RichLoss{
			{Kind: markdownv2.LossUnresolvedMedia, Node: "Enclave", Detail: "assets/a.png"},
		}, res.Losses)
	})

	t.Run("an asset resolver turns a local asset into a photo", func(t *testing.T) {
		c := markdownv2.RichConverter{}
		c.SetAssetResolver(func(path string) (string, bool) {
			require.Equal(t, "assets/a.png", path)
			return "https://cdn.example.com/a.png", true
		})

		res := c.Process(loadRichNote(t, "![](assets/a.png)"))

		require.Empty(t, res.Losses)
		require.Equal(t, "https://cdn.example.com/a.png", res.Blocks[0].Photo.URL)
	})

	t.Run("media mid-paragraph loses its caption on the wire, so it is a loss", func(t *testing.T) {
		res := convertRich(t, "before ![](https://example.com/a.png) after")

		require.Len(t, res.Blocks, 1)
		require.Equal(t, tgrich.BlockParagraph, res.Blocks[0].Type)
		require.Equal(t, []markdownv2.RichLoss{
			{Kind: markdownv2.LossInlineMedia, Node: "Enclave", Detail: "https://example.com/a.png"},
		}, res.Losses)
	})

	// Images on consecutive lines are one paragraph joined by a softbreak, which
	// is how Obsidian writes a gallery. Every image is an inline-media loss, and
	// what remains is the softbreak alone: a paragraph of pure whitespace. The
	// server echoes such a block back verbatim, so it costs a block against the
	// message limit and renders as a blank gap. Drop it.
	t.Run("a paragraph left holding only whitespace is not emitted", func(t *testing.T) {
		res := convertRich(t, "![](https://example.com/a.png)\n![](https://example.com/b.png)")

		require.Empty(t, res.Blocks)
		require.Len(t, res.Losses, 2)
	})
}

// Custom emoji do not survive a rich message: the server resolves the id and
// substitutes the sticker set's own fallback, and a word in the emoji slot is
// rejected outright. Report it, never emit it.
func TestRichCustomEmojiIsATypedLoss(t *testing.T) {
	res := convertRich(t, "hi ![🔥](https://ce.trip2g.com/5460736117236048513.webp) there")

	// The alt text is what the reader lost. Without it the loss set can report
	// that an emoji went missing but not which one.
	require.Equal(t, []markdownv2.RichLoss{
		{Kind: markdownv2.LossCustomEmoji, Node: "Enclave", Detail: "5460736117236048513", Alt: "🔥"},
	}, res.Losses)

	require.Len(t, res.Blocks, 1)
	require.NotContains(t, res.Blocks[0].Text.PlainText(), "5460736117236048513")
}

func TestRichLossesAreTyped(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []markdownv2.RichLoss
	}{
		{
			name:     "raw html block",
			markdown: "<div>hello</div>",
			want:     []markdownv2.RichLoss{{Kind: markdownv2.LossRawHTML, Node: "HTMLBlock"}},
		},
		{
			name:     "inline raw html other than u",
			markdown: "a <span>b</span>",
			want: []markdownv2.RichLoss{
				{Kind: markdownv2.LossRawHTML, Node: "RawHTML", Detail: "<span>"},
				{Kind: markdownv2.LossRawHTML, Node: "RawHTML", Detail: "</span>"},
			},
		},
		{
			name:     "embedded wikilink",
			markdown: "![[picture.png]]",
			want: []markdownv2.RichLoss{
				{Kind: markdownv2.LossEmbeddedWikiLink, Node: "WikiLink", Detail: "picture.png"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := convertRich(t, tt.markdown)
			require.Equal(t, tt.want, res.Losses)
		})
	}
}

func TestRichWikilinkUsesTheLinkResolver(t *testing.T) {
	c := markdownv2.RichConverter{}
	c.SetLinkResolver(func(target string) (*markdownv2.LinkResolverResult, error) {
		require.Equal(t, "Other Note", target)
		return &markdownv2.LinkResolverResult{URL: "https://trip2g.com/other"}, nil
	})

	res := c.Process(loadRichNote(t, "see [[Other Note]] please"))

	require.Empty(t, res.Losses)
	require.Equal(t, tgrich.RichText{Children: []tgrich.RichText{
		{Text: "see "},
		{Text: "Other Note", URL: "https://trip2g.com/other"},
		{Text: " please"},
	}}, *res.Blocks[0].Text)
}

func TestRichWikilinkWithoutAResolverIsALoss(t *testing.T) {
	res := convertRich(t, "see [[Other Note]]")

	require.Equal(t, []markdownv2.RichLoss{
		{Kind: markdownv2.LossUnresolvedWikiLink, Node: "WikiLink", Detail: "Other Note"},
	}, res.Losses)
}

func TestRichVisibleLengthCountsUTF16Units(t *testing.T) {
	res := convertRich(t, "😀ab")

	require.Equal(t, 4, res.VisibleUTF16Length)
}

// The typed loss set exists so a future `auto` can be a real predicate. In V1
// it selects nothing: `auto` stays classic however much the rich conversion
// would have lost.
func TestAutoStaysClassicRegardlessOfLosses(t *testing.T) {
	note := loadRichNote(t, "<div>raw</div>\n\nsee [[Other Note]]")

	c := markdownv2.RichConverter{}
	res := c.Process(note)
	require.NotEmpty(t, res.Losses)

	mode, err := note.ExtractTelegramRichMode()
	require.NoError(t, err)
	require.Equal(t, model.TelegramRichAuto, mode)
	require.False(t, mode.UseRich())
}

func TestRichResultValidatesAgainstDefaultLimits(t *testing.T) {
	res := convertRich(t, "# Title\n\nbody")

	msg := tgrich.InputRichMessage{Blocks: res.Blocks, SkipEntityDetection: true}
	require.NoError(t, msg.Validate(tgrich.DefaultLimits()))
}
