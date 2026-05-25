// Demo: Telegram Business Connection driving an Obsidian Canvas as a bot
// navigation scenario.
//
// Usage:
//
//	export TELEGRAM_BOT_TOKEN="..."
//	# optional overrides:
//	export TRIP2G_CANVAS="docs/demo/telegramnavigation/demo.canvas"
//	export TRIP2G_VAULT="docs"
//	go run ./cmd/businessdemo
//
// Setup before running:
//  1. @BotFather -> Bot Settings -> Business Mode -> enable for the bot.
//  2. In Telegram client (Premium account): Settings -> Telegram Business ->
//     Chatbots -> add @<bot_username>, grant "Reply to messages" right.
//     If the existing connection has wrong rights, you must DISCONNECT and
//     RECONNECT — rights are baked at connection time and cannot be edited.
//
// What this demo does:
//   - Loads an Obsidian .canvas file at startup.
//   - /browse opens the canvas at its START marker (text node containing "START").
//   - Edges from the current node become inline keyboard buttons; tapping moves
//     to the target node. File nodes render the .md file's body (frontmatter
//     stripped). Text nodes render their text. Link nodes render the URL with
//     an Open URL button (auto-attached to outgoing edge buttons).
//   - Stack-based Back; Exit drops the keyboard while keeping the last note text.
//   - In business chats, the bot replies as the owner via business_connection_id.
//   - Owner-sent business_messages are skipped to avoid echo loops.
//
// The tgbotapi v5 in use predates Telegram Business and lacks BusinessConnection
// types. We bypass tgbotapi's Update parsing (custom JSON structs) and send via
// the public BotAPI.MakeRequest + tgbotapi.Params, injecting business_connection_id
// directly into request params.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"trip2g/internal/telegram"
)

type bizUpdate struct {
	UpdateID           int64             `json:"update_id"`
	Message            *bizMessage       `json:"message,omitempty"`
	CallbackQuery      *bizCallbackQuery `json:"callback_query,omitempty"`
	BusinessConnection *bizConnection    `json:"business_connection,omitempty"`
	BusinessMessage    *bizMessage       `json:"business_message,omitempty"`
}

type bizConnection struct {
	ID         string          `json:"id"`
	User       bizUser         `json:"user"`
	UserChatID int64           `json:"user_chat_id"`
	IsEnabled  bool            `json:"is_enabled"`
	CanReply   bool            `json:"can_reply"`
	Rights     json.RawMessage `json:"rights"`
}

type bizMessage struct {
	MessageID            int     `json:"message_id"`
	BusinessConnectionID string  `json:"business_connection_id,omitempty"`
	From                 bizUser `json:"from"`
	Chat                 bizChat `json:"chat"`
	Date                 int64   `json:"date"`
	Text                 string  `json:"text"`
}

type bizCallbackQuery struct {
	ID      string      `json:"id"`
	From    bizUser     `json:"from"`
	Message *bizMessage `json:"message,omitempty"`
	Data    string      `json:"data"`
}

type bizUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type bizChat struct {
	ID int64 `json:"id"`
}

var allowedUpdates = []string{ //nolint:gochecknoglobals // demo command state
	"message",
	"callback_query",
	"business_connection",
	"business_message",
	"edited_business_message",
}

// business_message fires for both incoming customer messages and outgoing
// owner messages; owner_id from this map lets us skip the latter.
var connections = map[string]int64{} //nolint:gochecknoglobals // demo command state

// canvas is loaded once at startup. Demo-scope only — production handler will
// pull from LatestNoteViews after the ingestion phase lands.
var canvas *Canvas //nolint:gochecknoglobals // demo command state

// ------------------------- nav state -----------------------------------------

type navState struct {
	current   string
	stack     []string
	lastMedia string // non-empty if the last rendered message was sendPhoto
}

// states keyed by "<chat_id>|<business_connection_id>" so DM and business
// chats stay isolated even for the same user id.
var states = map[string]*navState{} //nolint:gochecknoglobals // demo command state

func stateKey(chatID int64, bcID string) string {
	return strconv.FormatInt(chatID, 10) + "|" + bcID
}

// ------------------------- rendering -----------------------------------------

const (
	telegramMessageMax = 3800
	telegramCaptionMax = 1000 // 1024 hard, leave headroom for HTML tags
)

func renderNode(nodeID string, stackLen int) (string, string, string, bool) {
	n, found := canvas.node(nodeID)
	if !found {
		return "", "", "", false
	}
	body, media := canvas.nodeContent(n)
	text := renderBodyHTML(body)
	limit := telegramMessageMax
	if media != "" {
		limit = telegramCaptionMax
	}
	// TruncateContent is HTML-aware: it counts only visible characters and
	// closes any unclosed tags after the cut so Telegram doesn't 400 on the
	// payload.
	text = telegram.TruncateContent(text, limit)

	var rows [][]map[string]string
	seen := map[string]bool{}
	for _, e := range canvas.edgesFrom(nodeID) {
		if seen[e.ToNode] {
			continue
		}
		seen[e.ToNode] = true
		target, exists := canvas.node(e.ToNode)
		if !exists {
			continue
		}
		label := canvas.edgeLabel(e)
		if target.Type == nodeTypeLink && target.URL != "" {
			rows = append(rows, []map[string]string{{
				"text": label,
				"url":  target.URL,
			}})
			continue
		}
		rows = append(rows, []map[string]string{{
			"text":          label,
			"callback_data": "nav:open:" + e.ToNode,
		}})
	}
	var nav []map[string]string
	if stackLen > 0 {
		nav = append(nav, map[string]string{"text": "← Back", "callback_data": "nav:back"})
	}
	nav = append(nav, map[string]string{"text": "✕ Exit", "callback_data": "nav:exit"})
	rows = append(rows, nav)

	j, _ := json.Marshal(map[string]any{"inline_keyboard": rows})
	return text, media, string(j), true
}

func emptyMarkup() string {
	j, _ := json.Marshal(map[string]any{"inline_keyboard": [][]any{}})
	return string(j)
}

// ------------------------- send helpers --------------------------------------

func sendText(bot *tgbotapi.BotAPI, chatID int64, bcID, text, markup string) error {
	p := tgbotapi.Params{
		"chat_id":    strconv.FormatInt(chatID, 10),
		"text":       text,
		"parse_mode": "HTML",
	}
	if markup != "" {
		p["reply_markup"] = markup
	}
	if bcID != "" {
		p["business_connection_id"] = bcID
	}
	_, err := bot.MakeRequest("sendMessage", p)
	return err
}

func editText(bot *tgbotapi.BotAPI, chatID int64, messageID int, bcID, text, markup string) error {
	p := tgbotapi.Params{
		"chat_id":    strconv.FormatInt(chatID, 10),
		"message_id": strconv.Itoa(messageID),
		"text":       text,
		"parse_mode": "HTML",
	}
	if markup != "" {
		p["reply_markup"] = markup
	}
	if bcID != "" {
		p["business_connection_id"] = bcID
	}
	_, err := bot.MakeRequest("editMessageText", p)
	return err
}

// sendNode dispatches to sendPhoto or sendMessage based on whether the node
// carries a resolved media path. Demo-scope: callers don't persist the new
// message id, so we only return error.
func sendNode(bot *tgbotapi.BotAPI, chatID int64, bcID, text, media, markup string) error {
	if media == "" {
		_, err := sendMessageRaw(bot, chatID, bcID, text, markup)
		return err
	}
	p := tgbotapi.Params{
		"chat_id":    strconv.FormatInt(chatID, 10),
		"caption":    text,
		"parse_mode": "HTML",
	}
	if markup != "" {
		p["reply_markup"] = markup
	}
	if bcID != "" {
		p["business_connection_id"] = bcID
	}
	_, err := bot.UploadFiles("sendPhoto", p, []tgbotapi.RequestFile{{
		Name: "photo",
		Data: tgbotapi.FilePath(media),
	}})
	return err
}

func sendMessageRaw(bot *tgbotapi.BotAPI, chatID int64, bcID, text, markup string) (int, error) {
	p := tgbotapi.Params{
		"chat_id":    strconv.FormatInt(chatID, 10),
		"text":       text,
		"parse_mode": "HTML",
	}
	if markup != "" {
		p["reply_markup"] = markup
	}
	if bcID != "" {
		p["business_connection_id"] = bcID
	}
	resp, err := bot.MakeRequest("sendMessage", p)
	if err != nil {
		return 0, err
	}
	return extractMessageID(resp.Result), nil
}

func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, bcID string) error {
	p := tgbotapi.Params{
		"chat_id":    strconv.FormatInt(chatID, 10),
		"message_id": strconv.Itoa(messageID),
	}
	if bcID != "" {
		p["business_connection_id"] = bcID
	}
	_, err := bot.MakeRequest("deleteMessage", p)
	return err
}

func extractMessageID(raw json.RawMessage) int {
	var m struct {
		MessageID int `json:"message_id"`
	}
	_ = json.Unmarshal(raw, &m)
	return m.MessageID
}

func editReplyMarkup(bot *tgbotapi.BotAPI, chatID int64, messageID int, bcID, markup string) error {
	p := tgbotapi.Params{
		"chat_id":      strconv.FormatInt(chatID, 10),
		"message_id":   strconv.Itoa(messageID),
		"reply_markup": markup,
	}
	if bcID != "" {
		p["business_connection_id"] = bcID
	}
	_, err := bot.MakeRequest("editMessageReplyMarkup", p)
	return err
}

func answerCallback(bot *tgbotapi.BotAPI, callbackID, text string, alert bool) error {
	p := tgbotapi.Params{"callback_query_id": callbackID}
	if text != "" {
		p["text"] = text
	}
	if alert {
		p["show_alert"] = "true"
	}
	_, err := bot.MakeRequest("answerCallbackQuery", p)
	return err
}

// ------------------------- handlers ------------------------------------------

func startBrowse(bot *tgbotapi.BotAPI, chatID int64, bcID string) error {
	entry := canvas.entry()
	if entry == "" {
		return sendText(bot, chatID, bcID, "Canvas has no START marker.", "")
	}
	st := &navState{current: entry}
	states[stateKey(chatID, bcID)] = st
	text, media, markup, ok := renderNode(entry, 0)
	if !ok {
		return sendText(bot, chatID, bcID, "Failed to render entry node.", "")
	}
	if err := sendNode(bot, chatID, bcID, text, media, markup); err != nil {
		return err
	}
	st.lastMedia = media
	return nil
}

func isStartCommand(text string) bool {
	t := strings.TrimSpace(text)
	return t == "/browse" || t == "/start" ||
		strings.HasPrefix(t, "/browse ") || strings.HasPrefix(t, "/start ")
}

func handleMessage(bot *tgbotapi.BotAPI, m *bizMessage, bcID string) error {
	if isStartCommand(m.Text) {
		return startBrowse(bot, m.Chat.ID, bcID)
	}
	return nil
}

// renderTransition renders the current node of st onto the chat. When neither
// the previous nor the new render involves a photo, it edits in place (smooth).
// Otherwise it deletes the previous message (best-effort — fails silently if
// the bot lacks the right) and sends a fresh photo/text message at the bottom.
func renderTransition(bot *tgbotapi.BotAPI, st *navState, chatID int64, prevMsgID int, bcID string) error {
	text, media, markup, ok := renderNode(st.current, len(st.stack))
	if !ok {
		return nil
	}
	if st.lastMedia == "" && media == "" {
		st.lastMedia = ""
		return editText(bot, chatID, prevMsgID, bcID, text, markup)
	}
	if err := deleteMessage(bot, chatID, prevMsgID, bcID); err != nil {
		log.Printf("delete prev message (best-effort): %v", err)
	}
	if err := sendNode(bot, chatID, bcID, text, media, markup); err != nil {
		return err
	}
	st.lastMedia = media
	return nil
}

func handleCallback(bot *tgbotapi.BotAPI, cq *bizCallbackQuery) error {
	if cq.Message == nil {
		return answerCallback(bot, cq.ID, "", false)
	}
	chatID := cq.Message.Chat.ID
	msgID := cq.Message.MessageID
	bcID := cq.Message.BusinessConnectionID

	key := stateKey(chatID, bcID)
	st, exists := states[key]
	if !exists {
		entry := canvas.entry()
		if entry == "" {
			return answerCallback(bot, cq.ID, "Session lost", true)
		}
		st = &navState{current: entry}
		states[key] = st
	}

	parts := strings.SplitN(cq.Data, ":", 3)
	if len(parts) < 2 || parts[0] != "nav" {
		return answerCallback(bot, cq.ID, "", false)
	}

	switch parts[1] {
	case "open":
		if len(parts) < 3 {
			return answerCallback(bot, cq.ID, "bad payload", true)
		}
		targetID := parts[2]
		if _, ok := canvas.node(targetID); !ok {
			return answerCallback(bot, cq.ID, "node not found", true)
		}
		st.stack = append(st.stack, st.current)
		st.current = targetID
		if err := answerCallback(bot, cq.ID, "", false); err != nil {
			log.Printf("ack: %v", err)
		}
		return renderTransition(bot, st, chatID, msgID, bcID)

	case "back":
		if len(st.stack) == 0 {
			return answerCallback(bot, cq.ID, "Already at the top", true)
		}
		st.current = st.stack[len(st.stack)-1]
		st.stack = st.stack[:len(st.stack)-1]
		if err := answerCallback(bot, cq.ID, "", false); err != nil {
			log.Printf("ack: %v", err)
		}
		return renderTransition(bot, st, chatID, msgID, bcID)

	case "exit":
		delete(states, key)
		if err := answerCallback(bot, cq.ID, "Saved", false); err != nil {
			log.Printf("ack: %v", err)
		}
		return editReplyMarkup(bot, chatID, msgID, bcID, emptyMarkup())
	}
	return answerCallback(bot, cq.ID, "", false)
}

// ------------------------- main loop -----------------------------------------

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN env var is required")
	}

	canvasPath := envOr("TRIP2G_CANVAS", "docs/demo/telegramnavigation/demo.canvas")
	vaultRoot := envOr("TRIP2G_VAULT", "docs")

	var err error
	canvas, err = loadCanvas(canvasPath, vaultRoot)
	if err != nil {
		log.Fatalf("loadCanvas %q: %v", canvasPath, err)
	}
	log.Printf("canvas loaded: %s (%d nodes, vault=%s, entry=%q)",
		canvasPath, len(canvas.nodesByID), vaultRoot, canvas.entry())

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("NewBotAPI: %v", err)
	}
	log.Printf("authorized as @%s (id=%d)", bot.Self.UserName, bot.Self.ID)
	log.Printf("send /browse to @%s in DM, or have anyone write /browse in a business chat", bot.Self.UserName)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	allowed, _ := json.Marshal(allowedUpdates)
	var offset int64

	for {
		if ctx.Err() != nil {
			log.Print("shutdown")
			return
		}
		var updates []bizUpdate
		updates, err = getUpdates(bot, offset, allowed)
		if err != nil {
			log.Printf("getUpdates: %v", err)
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
				return
			}
			continue
		}
		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			handleUpdate(bot, u)
		}
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func getUpdates(bot *tgbotapi.BotAPI, offset int64, allowed []byte) ([]bizUpdate, error) {
	params := tgbotapi.Params{
		"timeout":         "25",
		"allowed_updates": string(allowed),
	}
	if offset > 0 {
		params["offset"] = strconv.FormatInt(offset, 10)
	}
	resp, err := bot.MakeRequest("getUpdates", params)
	if err != nil {
		return nil, err
	}
	var raws []json.RawMessage
	if unmarshalErr := json.Unmarshal(resp.Result, &raws); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal raw updates: %w", unmarshalErr)
	}
	updates := make([]bizUpdate, 0, len(raws))
	for _, raw := range raws {
		var u bizUpdate
		if parseErr := json.Unmarshal(raw, &u); parseErr != nil {
			log.Printf("parse update failed: %v raw=%s", parseErr, string(raw))
			continue
		}
		updates = append(updates, u)
	}
	return updates, nil
}

func handleUpdate(bot *tgbotapi.BotAPI, u bizUpdate) {
	switch {
	case u.BusinessConnection != nil:
		c := u.BusinessConnection
		if c.IsEnabled {
			connections[c.ID] = c.User.ID
		} else {
			delete(connections, c.ID)
		}
		log.Printf("business_connection: id=%s owner=@%s (id=%d) enabled=%v can_reply=%v rights=%s",
			c.ID, c.User.Username, c.User.ID, c.IsEnabled, c.CanReply, string(c.Rights))

	case u.BusinessMessage != nil:
		m := u.BusinessMessage
		ownerID, known := connections[m.BusinessConnectionID]
		if !known && m.From.ID != m.Chat.ID {
			// In a 1-to-1 business chat chat.id is the customer; a sender
			// whose id differs from chat.id must be the owner. Cache so we
			// survive restarts without waiting for a fresh
			// business_connection update.
			ownerID = m.From.ID
			connections[m.BusinessConnectionID] = ownerID
			log.Printf("inferred owner=%d for conn=%s", ownerID, m.BusinessConnectionID)
		}
		isOwner := m.From.ID == ownerID
		if isStartCommand(m.Text) {
			src := "customer"
			if isOwner {
				src = "owner"
			}
			log.Printf("business_message [%s /browse]: from=@%s chat=%d — starting nav", src, m.From.Username, m.Chat.ID)
			if err := startBrowse(bot, m.Chat.ID, m.BusinessConnectionID); err != nil {
				log.Printf("startBrowse: %v", err)
			}
			return
		}
		if isOwner {
			log.Printf("business_message [owner skipped]: text=%q", m.Text)
			return
		}
		log.Printf("business_message [customer ignored]: from=@%s chat=%d text=%q",
			m.From.Username, m.Chat.ID, m.Text)

	case u.CallbackQuery != nil:
		cq := u.CallbackQuery
		var bcID string
		if cq.Message != nil {
			bcID = cq.Message.BusinessConnectionID
		}
		log.Printf("callback_query: from=@%s data=%q business_conn=%s", cq.From.Username, cq.Data, bcID)
		if err := handleCallback(bot, cq); err != nil {
			log.Printf("handleCallback: %v", err)
		}

	case u.Message != nil:
		m := u.Message
		log.Printf("message: from=@%s chat=%d text=%q", m.From.Username, m.Chat.ID, m.Text)
		if err := handleMessage(bot, m, ""); err != nil {
			log.Printf("handleMessage(dm): %v", err)
		}
	}
}
