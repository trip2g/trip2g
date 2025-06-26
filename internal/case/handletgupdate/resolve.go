package handletgupdate

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	TgUserStateByBotIDAndChatID(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error)
	InsertTgUserProfile(ctx context.Context, arg db.InsertTgUserProfileParams) error
	UpsertTgUserState(ctx context.Context, arg db.UpsertTgUserStateParams) error
	UpsertTgBotChat(ctx context.Context, arg db.UpsertTgBotChatParams) error
	MarkTgBotChatRemoved(ctx context.Context, id int64) error
	CalculateSha256(s string) string
	PublicURL() string
	LatestNoteViews() *model.NoteViews // TODO: read LiveNoteViews for production users
	Logger() logger.Logger
	BotID() int64
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type Question struct {
	ID       int
	Text     string
	Category string
}

type MBTIResult struct {
	Name       string             `json:"name"`
	Categories map[string]float32 `json:"categories"`
}

type QuizState struct {
	Answers map[int]int `json:"answers"`
}

type UserStateData struct {
	QuizStates map[string]QuizState `json:"quiz_states"`
}

type UserState struct {
	*UserStateData

	ChatID int64
	Value  string

	UpdateCount int64
}

type request struct {
	chatID    int64
	update    tgbotapi.Update
	userState *UserState
	env       Env
	questions []Question
}

func Resolve(ctx context.Context, env Env, update tgbotapi.Update) error {
	// Update user profile if we have a message with user info
	if update.Message != nil && update.Message.From != nil {
		profileParams := db.InsertTgUserProfileParams{
			ChatID:    update.Message.Chat.ID,
			BotID:     env.BotID(),
			FirstName: toNullString(update.Message.From.FirstName),
			LastName:  toNullString(update.Message.From.LastName),
			Username:  toNullString(update.Message.From.UserName),
		}

		hashValue, err := json.Marshal(profileParams)
		if err != nil {
			return fmt.Errorf("failed to marshal user profile params: %w", err)
		}

		profileParams.Sha256Hash = env.CalculateSha256(string(hashValue))

		err = env.InsertTgUserProfile(ctx, profileParams)
		if err != nil {
			return fmt.Errorf("failed to insert user profile: %w", err)
		}
	}

	var err error

	req := request{
		env:    env,
		update: update,
	}

	if update.CallbackQuery != nil {
		req.chatID = update.CallbackQuery.Message.Chat.ID
	} else if update.Message != nil {
		req.chatID = update.Message.Chat.ID
	}

	req.userState, err = req.UserState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user state: %w", err)
	}

	defer func() {
		updateErr := req.updateUserState(ctx)
		if err != nil {
			env.Logger().Error("failed to update user state", "error", updateErr)
		}
	}()

	// Handle bot being added/removed from chats
	if update.MyChatMember != nil {
		return req.handleMyChatMember(ctx)
	}

	if update.CallbackQuery != nil {
		return req.handleCallbackQuery(ctx)
	}

	if update.Message != nil && update.Message.IsCommand() {
		return req.handleCommands(ctx)
	}

	return nil
}

func (req *request) handleMyChatMember(ctx context.Context) error {
	log := req.env.Logger()
	chatMember := req.update.MyChatMember
	
	// Only track channels, groups, and supergroups
	chat := chatMember.Chat
	if chat.Type != "channel" && chat.Type != "group" && chat.Type != "supergroup" {
		return nil
	}
	
	newStatus := chatMember.NewChatMember.Status
	oldStatus := chatMember.OldChatMember.Status
	
	log.Info("bot chat member status changed",
		"chat_id", chat.ID,
		"chat_type", chat.Type,
		"chat_title", chat.Title,
		"old_status", oldStatus,
		"new_status", newStatus,
	)
	
	// Bot was added to the chat
	if (newStatus == "member" || newStatus == "administrator") && 
	   (oldStatus == "left" || oldStatus == "kicked") {
		
		err := req.env.UpsertTgBotChat(ctx, db.UpsertTgBotChatParams{
			ID:        chat.ID,
			ChatType:  chat.Type,
			ChatTitle: chat.Title,
		})
		if err != nil {
			log.Error("failed to upsert bot chat", "error", err, "chat_id", chat.ID)
			return fmt.Errorf("failed to upsert bot chat: %w", err)
		}
		
		log.Info("bot added to chat", "chat_id", chat.ID, "chat_title", chat.Title)
	}
	
	// Bot was removed from the chat
	if (newStatus == "left" || newStatus == "kicked") && 
	   (oldStatus == "member" || oldStatus == "administrator") {
		
		err := req.env.MarkTgBotChatRemoved(ctx, chat.ID)
		if err != nil {
			log.Error("failed to mark bot chat as removed", "error", err, "chat_id", chat.ID)
			return fmt.Errorf("failed to mark bot chat as removed: %w", err)
		}
		
		log.Info("bot removed from chat", "chat_id", chat.ID, "chat_title", chat.Title)
	}
	
	return nil
}

func (req *request) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(req.chatID, text)

	_, err := req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (req *request) handleCallbackQuery(ctx context.Context) error {
	log := req.env.Logger()

	actionParts := strings.SplitN(req.update.CallbackQuery.Data, ":", 3)

	switch actionParts[0] {
	case "start_mbti":
		callback := tgbotapi.NewCallback(req.update.CallbackQuery.ID, req.update.CallbackQuery.Data)

		_, err := req.env.Request(callback)
		if err != nil {
			return fmt.Errorf("failed to send callback: %w", err)
		}

		return req.sendNextQuestion(ctx)

	case "mbti_answer":
		return req.handleMBTIAnswer(ctx, actionParts)

	default:
		log.Warn("unknown action", "action", req.update.CallbackQuery.Data)

		msg := tgbotapi.NewMessage(req.update.CallbackQuery.Message.Chat.ID, "Unknown action")

		_, err := req.env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func (req *request) handleCommands(ctx context.Context) error {
	switch req.update.Message.Command() {
	case "start":
		return req.sendStartMenu(ctx)

	default:
		msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, "Unknown command")

		_, err := req.env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func (req *request) handleMBTIAnswer(ctx context.Context, actionParts []string) error {
	if len(actionParts) < 3 {
		return req.SendMessage("Invalid action format")
	}

	callbackMsg := req.update.CallbackQuery.Message

	// prepare params
	id, err := strconv.Atoi(actionParts[1])
	if err != nil {
		return fmt.Errorf("failed to parse question ID: %w", err)
	}

	answerValue, err := strconv.Atoi(actionParts[2])
	if err != nil {
		return fmt.Errorf("failed to parse answer value: %w", err)
	}

	// remove the callback buttons
	editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackMsg.Chat.ID, callbackMsg.MessageID, tgbotapi.NewInlineKeyboardMarkup())
	editMsg.ReplyMarkup = generateMBTIAnswerKeyboard(id, answerValue+3)

	_, err = req.env.Request(editMsg)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	// process the answer
	state, exists := req.userState.QuizStates["mbti"]
	if !exists {
		return fmt.Errorf("quiz state not found for user %d", req.chatID)
	}

	if state.Answers == nil {
		state.Answers = make(map[int]int)
	}

	state.Answers[id] = answerValue

	// Update the map with the modified state
	req.userState.QuizStates["mbti"] = state

	return req.sendNextQuestion(ctx)
}

func (req *request) sendStartMenu(_ context.Context) error {
	msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, "Welcome to Trip2G!")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Начать тест", "start_mbti"),
			tgbotapi.NewInlineKeyboardButtonData("Подробнее", "more_details"),
		),
	)

	_, err := req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send start menu: %w", err)
	}

	return nil
}

func (req *request) Questions(ctx context.Context) ([]Question, error) {
	if req.questions != nil {
		return req.questions, nil
	}

	notes := req.env.LatestNoteViews()

	re := regexp.MustCompile(`_mbti/(\d+)\.md$`)

	var res []Question

	for _, note := range notes.List {
		m := re.FindStringSubmatch(note.Path)
		if len(m) < 2 {
			continue
		}

		id, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse question ID from path %s: %w", note.Path, err)
		}

		categoryI, ok := note.RawMeta["category"]
		if !ok {
			return nil, fmt.Errorf("category not found in note %s", note.Path)
		}

		category, ok := categoryI.(string)
		if !ok {
			return nil, fmt.Errorf("category is not a string in note %s. %T", note.Path, categoryI)
		}

		question := Question{
			ID:       id,
			Text:     trimYAMLFrontMatter(string(note.Content)),
			Category: category,
		}

		res = append(res, question)
	}

	slices.SortFunc(res, func(a, b Question) int {
		return cmp.Compare(a.ID, b.ID)
	})

	req.questions = res

	return res, nil
}

func trimYAMLFrontMatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return content
	}

	return strings.TrimSpace(parts[2])
}

func (req *request) UserState(ctx context.Context) (*UserState, error) {
	if req.userState != nil {
		return req.userState, nil
	}

	params := db.TgUserStateByBotIDAndChatIDParams{
		BotID:  req.env.BotID(),
		ChatID: req.chatID,
	}

	userState := UserState{
		UserStateData: &UserStateData{
			QuizStates: map[string]QuizState{},
		},

		ChatID: req.chatID,
		Value:  "pending", // Default value if no state found
	}

	row, err := req.env.TgUserStateByBotIDAndChatID(ctx, params)
	if err != nil {
		if db.IsNoFound(err) {
			return &userState, nil
		}

		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	userState.Value = row.Value
	userState.UpdateCount = row.UpdateCount

	err = json.Unmarshal([]byte(row.Data), &userState.UserStateData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state data: %w", err)
	}

	return &userState, nil
}

func (req *request) updateUserState(ctx context.Context) error {
	data, err := json.Marshal(req.userState.UserStateData)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	upsertParams := db.UpsertTgUserStateParams{
		ChatID: req.userState.ChatID,
		BotID:  req.env.BotID(),
		Value:  req.userState.Value,
		Data:   string(data),

		UpdateCount: req.userState.UpdateCount + 1,
	}

	err = req.env.UpsertTgUserState(ctx, upsertParams)
	if err != nil {
		return fmt.Errorf("failed to upsert user state: %w", err)
	}

	return nil
}

func generateMBTIAnswerKeyboard(questionID int, answerIdx int) *tgbotapi.InlineKeyboardMarkup {
	mbtiAnswers := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("😡", "-3"),
			tgbotapi.NewInlineKeyboardButtonData("😠", "-2"),
			tgbotapi.NewInlineKeyboardButtonData("😕", "-1"),
			tgbotapi.NewInlineKeyboardButtonData("😐", "0"),
			tgbotapi.NewInlineKeyboardButtonData("🙂", "1"),
			tgbotapi.NewInlineKeyboardButtonData("😊", "2"),
			tgbotapi.NewInlineKeyboardButtonData("😄", "3"),
		),
	)

	for i, button := range mbtiAnswers.InlineKeyboard[0] {
		v := fmt.Sprintf("mbti_answer:%d:%s", questionID, *button.CallbackData)
		mbtiAnswers.InlineKeyboard[0][i].CallbackData = &v

		if i == answerIdx {
			mbtiAnswers.InlineKeyboard[0][i].Text = "✅"
		}
	}

	return &mbtiAnswers
}

func (req *request) sendNextQuestion(ctx context.Context) error {
	questions, err := req.Questions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get questions: %w", err)
	}

	if len(questions) == 0 {
		return req.SendMessage("К сожалению, вопросы теста не найдены.")
	}

	state, exists := req.userState.QuizStates["mbti"]
	if !exists {
		state = QuizState{
			Answers: make(map[int]int),
		}
		req.userState.QuizStates["mbti"] = state
	}

	if state.Answers == nil {
		state.Answers = make(map[int]int)
		req.userState.QuizStates["mbti"] = state
	}

	for _, question := range questions {
		_, ok := state.Answers[question.ID]
		if ok {
			continue
		}

		text := fmt.Sprintf("Вопрос %d/%d:\n\n%s", len(state.Answers)+1, len(questions), question.Text)

		msg := tgbotapi.NewMessage(req.chatID, text)
		msg.ReplyMarkup = generateMBTIAnswerKeyboard(question.ID, -1)

		_, err = req.env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send question: %w", err)
		}

		return nil
	}

	mbtiResult := calculateMBTI(questions, state.Answers)

	// Create horizontal bars for categories
	var categoryBars strings.Builder
	categoryNames := map[string][2]string{
		"IE": {"Интроверсия", "Экстраверсия"},
		"SN": {"Сенсорика", "Интуиция"},
		"TF": {"Мышление", "Чувства"},
		"PJ": {"Восприятие", "Суждение"},
		"AR": {"Турбулентность", "Уверенность"},
	}

	for _, category := range []string{"IE", "SN", "TF", "PJ", "AR"} {
		percentage := mbtiResult.Categories[category]
		names := categoryNames[category]

		// Determine which trait is dominant and flip if needed
		var leftName, rightName string
		var leftLetter, rightLetter string
		var displayPercentage float32

		if percentage < 0.5 {
			// First letter is dominant
			leftName = names[0]
			rightName = names[1]
			leftLetter = string(category[0])
			rightLetter = string(category[1])
			displayPercentage = 1 - percentage
		} else {
			// Second letter is dominant
			leftName = names[1]
			rightName = names[0]
			leftLetter = string(category[1])
			rightLetter = string(category[0])
			displayPercentage = percentage
		}

		bar := generateHorizontalBar(displayPercentage, leftLetter)
		categoryBars.WriteString(fmt.Sprintf("%s (%s) > %s (%s)\n%s\n\n", leftName, leftLetter, rightName, rightLetter, bar))
	}

	text := fmt.Sprintf("Все вопросы теста пройдены!\n\nВаш тип личности: %s\n\n%sСпасибо за участие!", mbtiResult.Name, categoryBars.String())

	url := fmt.Sprintf("%s/mbti/%s?client=tg", req.env.PublicURL(), strings.ToLower(mbtiResult.Name[:4]))

	msg := tgbotapi.NewMessage(req.chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonWebApp("Подробнее о типе", tgbotapi.WebAppInfo{URL: url}),
		),
	)

	_, err = req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send question: %w", err)
	}

	return nil
}

func calculateMBTI(questions []Question, answers map[int]int) MBTIResult {
	// Map questions by ID for quick lookup
	questionMap := make(map[int]Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	// Standard MBTI categories in order (removed BG)
	standardCategories := []string{"IE", "SN", "TF", "PJ", "AR"}

	// Calculate sums and counts for each category
	categorySums := make(map[string]int)
	categoryCounts := make(map[string]int)

	for questionID, answer := range answers {
		question, exists := questionMap[questionID]
		if !exists {
			continue
		}

		category := question.Category

		// Skip BG category
		if category == "BG" || category == "GB" {
			continue
		}

		// Normalize category and calculate answer value
		var normalizedCategory string
		var answerValue int

		// Check if category is in standard form (IE, SN, TF, PJ, AR)
		isStandard := false
		for _, std := range standardCategories {
			if category == std {
				isStandard = true
				normalizedCategory = category
				answerValue = -answer // Negative for standard categories
				break
			}
		}

		if !isStandard {
			// Reverse the category (e.g., "EI" -> "IE")
			normalizedCategory = reverseString(category)
			answerValue = answer // Positive for reversed categories
		}

		categorySums[normalizedCategory] += answerValue
		categoryCounts[normalizedCategory]++
	}

	// Calculate percentages and build result
	categoryPercentages := make(map[string]float32)
	var resultName strings.Builder

	for idx, category := range standardCategories {
		sum := categorySums[category]
		count := categoryCounts[category]

		// Calculate percentage for the second letter (positive direction)
		var percentage float32
		if count > 0 {
			// Normalize sum to range 0-1
			// sum ranges from -3*count to +3*count
			maxPossible := float32(3 * count)
			normalized := (float32(sum) + maxPossible) / (2 * maxPossible)
			percentage = normalized
		} else {
			percentage = 0.5 // Default to middle if no data
		}

		categoryPercentages[category] = percentage

		// Choose letter based on sum
		var letter string
		if sum >= 0 {
			// Take second letter (index 1)
			letter = string(category[1])
		} else {
			// Take first letter (index 0)
			letter = string(category[0])
		}

		if idx == 4 {
			resultName.WriteString("-")
		}

		resultName.WriteString(letter)
	}

	return MBTIResult{
		Name:       resultName.String(),
		Categories: categoryPercentages,
	}
}

func generateHorizontalBar(percentage float32, letter string) string {
	// percentage is now always between 0.5 and 1.0
	// Create a horizontal bar with 20 characters
	barLength := 20
	filled := int(percentage * float32(barLength))

	var bar strings.Builder

	// Add the bar
	for i := range barLength {
		if i < filled {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}

	// Add percentage and letter
	bar.WriteString(fmt.Sprintf(" %.0f%% (%s)", percentage*100, letter))

	return bar.String()
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
