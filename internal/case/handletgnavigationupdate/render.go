package handletgnavigationupdate

import (
	"errors"
	"fmt"
	"strings"
	"trip2g/internal/markdownv2"
	"trip2g/internal/model"
	"trip2g/internal/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yuin/goldmark/ast"
)

const maxNoteTextLen = 4000

// RenderNote builds the message text and inline keyboard for a note.
// stack contains PathIDs of notes to return to (back button is shown when non-empty).
// botLink is used to build deep links, e.g. "https://t.me/mybot".
func RenderNote(noteViews *model.NoteViews, pathID int64, stack []int64, botLink string) (string, *tgbotapi.InlineKeyboardMarkup, error) {
	if noteViews == nil {
		return "", nil, errors.New("notes not loaded yet")
	}

	note := noteByPathID(noteViews, pathID)
	if note == nil {
		return "", nil, fmt.Errorf("note not found: %d", pathID)
	}

	converter := &markdownv2.HTMLConverter{}
	converter.SetLinkResolver(func(target string) (*markdownv2.LinkResolverResult, error) {
		resolved := resolveBasename(noteViews, target)
		if resolved == nil {
			return nil, &markdownv2.LinkResolverError{Target: target, Reason: "not found"}
		}
		return &markdownv2.LinkResolverResult{
			URL:   fmt.Sprintf("%s?start=browse_%d", botLink, resolved.PathID),
			Label: resolved.Title,
		}, nil
	})
	converter.UnknownNodeHandler = func(c *markdownv2.HTMLConverter, n ast.Node, src []byte, entering bool) (ast.WalkStatus, bool) {
		if _, ok := n.(*ast.Heading); ok {
			if entering {
				c.Write("<b>")
			} else {
				c.Write("</b>\n")
			}
			return ast.WalkContinue, true
		}
		return ast.WalkContinue, true
	}

	result := converter.Process(note)
	text := telegram.TruncateContent(result.Content, maxNoteTextLen)

	// Build keyboard: telegram_buttons from frontmatter, then ← Back if stack non-empty
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, raw := range note.ExtractTelegramButtons() {
		links := ParseWikilinks(raw)
		if len(links) == 0 {
			continue
		}
		target := resolveBasename(noteViews, links[0].Target)
		if target == nil {
			continue
		}
		label := links[0].Display
		if label == "" {
			if target.Title != "" {
				label = target.Title
			} else {
				label = links[0].Target
			}
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("nav:open:%d", target.PathID)),
		))
	}
	if len(stack) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Back", "nav:back"),
		))
	}

	if len(rows) == 0 {
		return text, nil, nil
	}
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return text, &kb, nil
}

// FindStartNote returns the note named _bot_start.md, or nil.
func FindStartNote(noteViews *model.NoteViews) *model.NoteView {
	if noteViews == nil {
		return nil
	}
	for _, note := range noteViews.List {
		if strings.HasSuffix(note.Path, "_bot_start.md") {
			return note
		}
	}
	return nil
}

func noteByPathID(noteViews *model.NoteViews, pathID int64) *model.NoteView {
	for _, note := range noteViews.List {
		if note.PathID == pathID {
			return note
		}
	}
	return nil
}

func resolveBasename(noteViews *model.NoteViews, target string) *model.NoteView {
	key := strings.ToLower(strings.TrimSuffix(target, ".md"))
	candidates := noteViews.BasenameMap[key]
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}
