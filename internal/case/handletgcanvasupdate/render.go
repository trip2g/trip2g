package handletgcanvasupdate

import (
	"encoding/json"
	"fmt"
	"strings"
	"trip2g/internal/model"
	"trip2g/internal/obsidiancanvas"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// renderNode produces the Telegram message text, optional media path, and
// inline keyboard markup JSON for a given canvas node.
func renderNode(env Env, canvas *obsidiancanvas.Canvas, nodeID string) (text, media, markup string) {
	node, ok := canvas.Node(nodeID)
	if !ok {
		return "(node not found)", "", ""
	}

	switch node.Type {
	case "text":
		text = renderBodyHTML(node.Text)
	case "file":
		text, media = renderFileNode(env, node)
	case "link":
		text = fmt.Sprintf(`<a href="%s">%s</a>`, htmlEscape(node.URL), htmlEscape(node.URL))
	default:
		text = "(unsupported node type: " + node.Type + ")"
	}

	// Build inline keyboard from outgoing edges
	edges := canvas.EdgesFrom(nodeID)
	markup = buildKeyboard(canvas, edges)

	return text, media, markup
}

// renderFileNode renders a file-type node using the HTMLConverter via Env.
func renderFileNode(env Env, node obsidiancanvas.Node) (text, media string) {
	nvs := env.LatestNoteViews()
	if nvs == nil {
		return "(notes not loaded)", ""
	}

	nv := nvs.PathMap[node.File]
	if nv == nil {
		// Try without leading path segments (obsidian uses vault-relative paths)
		nv = findNoteByFile(nvs, node.File)
	}
	if nv == nil {
		return fmt.Sprintf("(file not found: %s)", node.File), ""
	}

	return env.RenderNoteHTML(nv)
}

// findNoteByFile searches PathMap for a note matching the file path.
// Handles cases where the canvas file path may include a vault-internal prefix.
func findNoteByFile(nvs *model.NoteViews, file string) *model.NoteView {
	// Direct match first
	if nv, ok := nvs.PathMap[file]; ok {
		return nv
	}
	// Try stripping common prefixes (e.g. "demo/" prefix in demo canvas)
	for path, nv := range nvs.PathMap {
		if strings.HasSuffix(file, "/"+path) || file == path {
			return nv
		}
	}
	return nil
}

// buildKeyboard constructs an inline keyboard JSON string from canvas edges.
// Adds a Back button row and an Exit button row.
func buildKeyboard(canvas *obsidiancanvas.Canvas, edges []obsidiancanvas.Edge) string {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, e := range edges {
		label := canvas.EdgeLabel(e)
		callbackData := "nav:open:" + e.ToNode
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, callbackData),
		))
	}

	// Back button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Back", "nav:back"),
	))

	// Exit button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✕ Exit", "nav:exit"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b, _ := json.Marshal(kb)
	return string(b)
}
