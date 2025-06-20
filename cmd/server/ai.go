package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/thread"
	"github.com/henomis/lingoose/types"
)

type AIIssue struct {
	Marker  string `json:"marker"`
	Fix     string `json:"fix"`
	Comment string `json:"comment"`
}

type AIResponse struct {
	Issues     []AIIssue `json:"issues"`
	TotalCount int       `json:"totalCount"`
	PageSize   int       `json:"pageSize"`
}

type WikilinkCheckResponse struct {
	NeedsFix bool   `json:"needs_fix"`
	Fix      string `json:"fix,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type WikilinkResult struct {
	Issue *AIIssue
	Error error
	Index int
}

var checkWikilinkPrompt = `
You analyze a Russian text fragment with ONE wikilink and decide if it needs grammatical correction.

Context: You receive ~10 words before the wikilink, the wikilink itself, and ~10 words after.

Task: Determine if the wikilink needs a display text with correct grammatical form.

Rules:
1. If wikilink already has | (pipe), return needs_fix: false
2. If wikilink is grammatically correct as-is, return needs_fix: false  
3. Only fix if the word needs different grammatical case (падеж) for the context
4. Display text should be lowercase
5. Keep plural/singular form unchanged

Examples:
[[Принцип Парето]] помогает фокусироваться > [[Принцип Парето]] помогает фокусироваться // need_fix: false
[[Закон Мерфи]] предупреждает о рисках > [[Закон Мерфи]] предупреждает о рисках // need_fix: false
[[Цикл Деминга]] совершенствует процессы > [[Цикл Деминга]] совершенствует процессы // need_fix: false
[[Диаграмма Ганта]] визуализирует сроки > [[Диаграмма Ганта]] визуализирует сроки // need_fix: false
[[Метод пяти почему]] раскрывает причины > [[Метод пяти почему]] раскрывает причины // need_fix: false

Мы анализировали [[Кривая обучения]] команды > Мы анализировали [[Кривая обучения|кривую обучения]] команды
Я замечаю [[Эффект Даннинга-Крюгера]] у коллег > Я замечаю [[Эффект Даннинга-Крюгера|эффект Даннинга-Крюгера]] у коллег
План выйдет на [[Точка безубыточности]] осенью > План выйдет на [[Точка безубыточности|точку безубыточности]] осенью
Мы получили честную [[Обратная связь]] после демо > Мы получили честную [[Обратная связь|обратную связь]] после демо
Погрузился в [[Фокус глубокая работа]] и продолжил писать > Погрузился в [[Фокус глубокая работа|фокус глубокой работы]] и продолжил писать

Расставляю дела по [[Матрица Эйзенхауэра]]. > Расставляю дела по [[Матрица Эйзенхауэра|матрице Эйзенхауэра]].
Настраиваю дыхание по [[Протокол Делланы]]. > Настраиваю дыхание по [[Протокол Делланы|протоколу Делланы]].
Формируем персонажей через [[Карта эмпатии]]. > Формируем персонажей через [[Карта эмпатии|карту эмпатии]].
Сократили сроки через [[Метод критической цепи]]. > Сократили сроки через [[Метод критической цепи|метод критической цепи]].
Метрики собраны из [[Базовые метрики продукта]]. > Метрики собраны из [[Базовые метрики продукта|базовых метрик продукта]].

Return ONLY JSON:
{"needs_fix": true, "fix": "[[Original|corrected]]", "comment": "причина"}
OR
{"needs_fix": false}
`

type WikilinkContext struct {
	Before   string
	Wikilink string
	After    string
	Position int
}

func extractWikilinksWithContext(content string, wordsBefore, wordsAfter int) []WikilinkContext {
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := re.FindAllStringIndex(content, -1)

	var contexts []WikilinkContext
	words := strings.Fields(content)

	for _, match := range matches {
		start, end := match[0], match[1]
		wikilink := content[start:end]

		// Find word positions
		currentPos := 0
		wikilinkWordIndex := -1

		for i, word := range words {
			if currentPos <= start && currentPos+len(word) >= start {
				wikilinkWordIndex = i
				break
			}
			currentPos += len(word) + 1 // +1 for space
		}

		if wikilinkWordIndex == -1 {
			continue
		}

		// Get context words
		beforeStart := wikilinkWordIndex - wordsBefore
		if beforeStart < 0 {
			beforeStart = 0
		}

		afterEnd := wikilinkWordIndex + wordsAfter + 1
		if afterEnd > len(words) {
			afterEnd = len(words)
		}

		beforeWords := words[beforeStart:wikilinkWordIndex]
		afterWords := words[wikilinkWordIndex+1 : afterEnd]

		contexts = append(contexts, WikilinkContext{
			Before:   strings.Join(beforeWords, " "),
			Wikilink: wikilink,
			After:    strings.Join(afterWords, " "),
			Position: start,
		})
	}

	return contexts
}

func checkSingleWikilink(ctx WikilinkContext) (*AIIssue, error) {
	// Build the context string
	contextStr := fmt.Sprintf("%s %s %s", ctx.Before, ctx.Wikilink, ctx.After)

	myThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(checkWikilinkPrompt).Format(types.M{}),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent(contextStr),
		),
	)

	err := openai.New().WithModel("gpt-4o-mini").WithMaxTokens(512).Generate(context.Background(), myThread)
	if err != nil {
		return nil, err
	}

	resMsg := myThread.LastMessage()
	if resMsg.Contents[0].Type != thread.ContentTypeText {
		return nil, fmt.Errorf("unexpected content type: %s", resMsg.Contents[0].Type)
	}

	responseText := resMsg.Contents[0].Data.(string)

	var checkResp WikilinkCheckResponse
	err = json.Unmarshal([]byte(responseText), &checkResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !checkResp.NeedsFix {
		return nil, nil
	}

	return &AIIssue{
		Marker:  ctx.Wikilink,
		Fix:     checkResp.Fix,
		Comment: checkResp.Comment,
	}, nil
}

func resolveAI(content string, offset int) ([]byte, error) {
	// Extract wikilinks with 10 words context
	contexts := extractWikilinksWithContext(content, 10, 10)
	totalCount := len(contexts)

	// Apply offset and limit
	pageSize := 5
	start := offset
	end := offset + pageSize
	
	if start >= len(contexts) {
		// Return empty result if offset is beyond available wikilinks
		response := AIResponse{Issues: []AIIssue{}, TotalCount: totalCount, PageSize: pageSize}
		responseJSON, err := json.Marshal(response)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}
		return responseJSON, nil
	}
	
	if end > len(contexts) {
		end = len(contexts)
	}
	
	contexts = contexts[start:end]

	// Process wikilinks in parallel using goroutines
	var wg sync.WaitGroup
	results := make(chan WikilinkResult, len(contexts))

	// Launch goroutines for each wikilink
	for i, ctx := range contexts {
		wg.Add(1)
		go func(index int, context WikilinkContext) {
			defer wg.Done()
			
			issue, err := checkSingleWikilink(context)
			results <- WikilinkResult{
				Issue: issue,
				Error: err,
				Index: index,
			}
		}(i, ctx)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and preserve order
	var results_slice []WikilinkResult
	for result := range results {
		results_slice = append(results_slice, result)
	}

	// Sort results by original index to preserve order
	sort.Slice(results_slice, func(i, j int) bool {
		return results_slice[i].Index < results_slice[j].Index
	})

	// Extract issues in original order
	var allIssues []AIIssue
	for _, result := range results_slice {
		if result.Error != nil {
			fmt.Printf("Error checking wikilink at index %d: %v\n", result.Index, result.Error)
			continue
		}

		if result.Issue != nil {
			allIssues = append(allIssues, *result.Issue)
		}
	}

	// Build response
	response := AIResponse{Issues: allIssues, TotalCount: totalCount, PageSize: pageSize}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return responseJSON, nil
}
