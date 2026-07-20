package convertnoteviewtotgpost_test

import (
	"context"
	"strings"
	"testing"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

const richNoteBody = `## Heading two

A paragraph with [[other]] and **bold**.

| a | b |
|---|---|
| 1 | 2 |

> [!note]- Folded
> hidden body

- one
- two
`

func loadRichNote(t *testing.T, frontmatter string) *model.NoteViews {
	t.Helper()

	nvs, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{
			{Path: "main.md", Content: []byte("---\nfree: true\ntitle: \"Rich Note\"\n" + frontmatter + "---\n" + richNoteBody)},
			{Path: "other.md", Content: []byte("---\nfree: true\ntitle: \"Other\"\n---\nother body")},
		},
		Log:     &logger.TestLogger{},
		Version: "latest",
	})
	require.NoError(t, err)

	return nvs
}

func richEnv(nvs *model.NoteViews) *testEnv {
	return &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}
}

// requireWarning finds a warning anywhere in the list: the classic pass emits
// its own warnings first, so position means nothing.
func requireWarning(t *testing.T, warnings []string, substr string) {
	t.Helper()

	for _, warning := range warnings {
		if strings.Contains(warning, substr) {
			return
		}
	}

	require.Failf(t, "warning not found", "no warning contains %q, got %v", substr, warnings)
}

func blockTypes(blocks []tgrich.Block) []tgrich.BlockType {
	types := make([]tgrich.BlockType, 0, len(blocks))
	for _, block := range blocks {
		types = append(types, block.Type)
	}

	return types
}

func TestRichModeOnProducesBlocks(t *testing.T) {
	nvs := loadRichNote(t, "telegram_rich: on\n")

	source := model.TelegramPostSource{NoteView: nvs.Map["/main"], ChatID: 123}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), richEnv(nvs), source)
	require.NoError(t, err)

	require.True(t, post.IsRich())
	require.Equal(t, []tgrich.BlockType{
		tgrich.BlockHeading,
		tgrich.BlockParagraph,
		tgrich.BlockTable,
		tgrich.BlockDetails,
		tgrich.BlockList,
	}, blockTypes(post.RichBlocks))

	// The heading keeps its level rather than being flattened.
	require.Equal(t, 2, post.RichBlocks[0].Size)

	// The blocks must survive validation, or the send would be rejected later.
	require.NoError(t, tgrich.InputRichMessage{
		Blocks:              post.RichBlocks,
		SkipEntityDetection: true,
	}.Validate(tgrich.DefaultLimits()))

	// The classic content is still produced: the sent-message row stores it and
	// the update path still reads it.
	require.NotEmpty(t, post.Content)
}

func TestRichModeOffAndAbsentStayClassic(t *testing.T) {
	for name, frontmatter := range map[string]string{
		"absent": "",
		"off":    "telegram_rich: off\n",
		"auto":   "telegram_rich: auto\n",
	} {
		t.Run(name, func(t *testing.T) {
			nvs := loadRichNote(t, frontmatter)

			source := model.TelegramPostSource{NoteView: nvs.Map["/main"], ChatID: 123}

			post, err := convertnoteviewtotgpost.Resolve(context.Background(), richEnv(nvs), source)
			require.NoError(t, err)

			require.False(t, post.IsRich())
			require.Empty(t, post.RichBlocks)
			require.NotEmpty(t, post.Content)
		})
	}
}

// An invalid telegram_rich value must be visible, not silently classic.
func TestRichModeInvalidWarns(t *testing.T) {
	nvs := loadRichNote(t, "telegram_rich: maybe\n")

	source := model.TelegramPostSource{NoteView: nvs.Map["/main"], ChatID: 123}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), richEnv(nvs), source)
	require.NoError(t, err)

	require.False(t, post.IsRich())
	requireWarning(t, post.Warnings, "telegram_rich")
}

// The rich pass must not double-count links the classic pass already counted.
func TestRichModeDoesNotDoubleCountLinks(t *testing.T) {
	nvs := loadRichNote(t, "telegram_rich: on\n")
	classic := loadRichNote(t, "")

	source := model.TelegramPostSource{NoteView: nvs.Map["/main"], ChatID: 123}
	richPost, err := convertnoteviewtotgpost.Resolve(context.Background(), richEnv(nvs), source)
	require.NoError(t, err)

	classicSource := model.TelegramPostSource{NoteView: classic.Map["/main"], ChatID: 123}
	classicPost, err := convertnoteviewtotgpost.Resolve(context.Background(), richEnv(classic), classicSource)
	require.NoError(t, err)

	require.Equal(t, classicPost.LinkCount, richPost.LinkCount)
	require.Equal(t, classicPost.ExternalLinkCount, richPost.ExternalLinkCount)
	require.Equal(t, classicPost.UnresolvedLinkCount, richPost.UnresolvedLinkCount)
}

// Rich is a bot-path capability. An account destination must not silently send
// a classic post as if the author had not asked for rich.
func TestRichModeOnAccountDestinationWarns(t *testing.T) {
	nvs := loadRichNote(t, "telegram_rich: on\n")

	source := model.TelegramPostSource{
		NoteView:       nvs.Map["/main"],
		AccountID:      7,
		TelegramChatID: -1001234567890,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), richEnv(nvs), source)
	require.NoError(t, err)

	require.False(t, post.IsRich())
	requireWarning(t, post.Warnings, "account")
}
