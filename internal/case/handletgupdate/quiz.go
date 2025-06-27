package handletgupdate

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

		content := string(note.Content)

		content = trimYAMLFrontMatter(content)

		content = strings.TrimSpace(content)

		if content == "" {
			return nil, fmt.Errorf("note %s has no content", note.Path)
		}

		res = append(res, Question{
			ID:       id,
			Text:     content,
			Category: category,
		})
	}

	req.questions = res

	return res, nil
}

func (req *request) QuestionMap(ctx context.Context) (map[int]Question, error) {
	questions, err := req.Questions(ctx)
	if err != nil {
		return nil, err
	}

	questionMap := make(map[int]Question, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	return questionMap, nil
}

func (req *request) handleMBTIAnswer(ctx context.Context, actionParts []string) error {
	if len(actionParts) < 3 {
		return req.SendMessage("Invalid action format")
	}

	// prepare params
	id, err := strconv.Atoi(actionParts[1])
	if err != nil {
		return fmt.Errorf("failed to parse question ID: %w", err)
	}

	score, err := strconv.Atoi(actionParts[2])
	if err != nil {
		return fmt.Errorf("failed to parse score: %w", err)
	}

	// Read user state and ensure QuizStates is initialized
	if req.userState.QuizStates == nil {
		req.userState.QuizStates = make(map[string]QuizState)
	}

	// Get the current state or create a new one
	state, exists := req.userState.QuizStates["mbti"]
	if !exists {
		state = QuizState{
			Answers: make(map[int]int),
		}
	}

	// Ensure Answers map is initialized
	if state.Answers == nil {
		state.Answers = make(map[int]int)
	}

	// Validate that the question exists
	questionMap, err := req.QuestionMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to load questions: %w", err)
	}

	if _, questionExists := questionMap[id]; !questionExists {
		return req.SendMessage("❌ Invalid question. Please try again.")
	}

	// Store the answer
	state.Answers[id] = score

	// Update the state back to userState
	req.userState.QuizStates["mbti"] = state

	return req.sendNextQuestion(ctx)
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

func trimYAMLFrontMatter(content string) string {
	// Remove YAML front matter
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "---") {
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "---") {
				return strings.Join(lines[i+1:], "\n")
			}
		}
	}
	return content
}
