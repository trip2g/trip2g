package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"trip2g/internal/mdloader"

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
	Issues []AIIssue `json:"issues"`
}

type WikilinkCheckResponse struct {
	NeedsFix bool   `json:"needs_fix"`
	Fix      string `json:"fix,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

var checkWikilinkPrompt = `
# Task: Check if Single Wikilink Needs Grammatical Correction

You receive a sentence with context and ONE specific wikilink to analyze.

## Context
- Text has been pre-processed with automatic normalization: [[Link]] → [[Link|link]]  
- You get the full sentence for context and ONE specific wikilink from that sentence
- Focus ONLY on whether THIS wikilink has a grammatical case error

## Examples of what to fix:
1. Sentence: "Выполнить [[Зарядка|зарядка]]" → Wikilink: "[[Зарядка|зарядка]]" → Fix to: "[[Зарядка|зарядку]]" (винительный падеж)
2. Sentence: "Думать о [[Работа|работу]]" → Wikilink: "[[Работа|работу]]" → Fix to: "[[Работа|работе]]" (предложный падеж)  
3. Sentence: "Использовать [[Шаблон|шаблон]]" → Wikilink: "[[Шаблон|шаблон]]" → Fix to: "[[Шаблон|шаблона]]" (винительный падеж)
4. Sentence: "Идти к [[Врач]]" → Wikilink: "[[Врач]]" → Fix to: "[[Врач|врачу]]" (дательный падеж)

## What NOT to fix:
1. Complete phrases: [[Легче никогда не станет]]
2. Already correct cases: Из [[Дом|дома]]
3. Already normalized: [[Дневник сновидений|дневник сновидений]]
4. Subject in nominative case: Когда [[Мыслетоплево|мыслетоплево]] в достатке
5. Nominative after "это": Это [[Метод|метод]] работает
6. Subject of sentence: [[Человек|человек]] пришел
7. Correct plural forms: Прочитать [[Аффирмации|аффирмации]] (винительный падеж множественного числа)
8. Plural after verbs: Делать [[Упражнения|упражнения]], писать [[Заметки|заметки]]

## Specific Examples:
- "из [[Шаблон дневной заметки|шаблон дневной заметки]]" → Fix to: "[[Шаблон дневной заметки|шаблона дневной заметки]]" (родительный падеж после "из")
- "использовать [[Шаблон|шаблон]]" → Fix to: "[[Шаблон|шаблона]]" (винительный падеж после "использовать")

## Input Format
You receive:
- Sentence: [full sentence text]
- Wikilink: [specific wikilink to check]

## Output Format
Return ONLY raw JSON without markdown formatting:

If needs correction:
{"needs_fix": true, "fix": "[[corrected wikilink]]", "comment": "объяснение на русском"}

If no correction needed:
{"needs_fix": false}

**Important:**
- comment must be in Russian language
- DO NOT wrap response in json md blocks  
- Return raw JSON only
- fix must be the COMPLETE corrected wikilink, ready for replacement
- CHANGE the label part to correct grammatical case (шаблон → шаблона, работу → работе, etc.)
- If the wikilink is already correct, return {"needs_fix": false}
- NEVER return the same text as fix - if you can't determine the correct form, return {"needs_fix": false}
- Nominative case (именительный падеж) is correct for subjects and after "когда", "если", "это"
- PRESERVE singular/plural form - do NOT change аффирмации→аффирмацию, упражнения→упражнение
- Focus ONLY on grammatical case (падеж), not on number (число)
`

func resolveAI(content string) (*AIResponse, error) {
	// Split content into sentences BEFORE normalization to preserve original wikilinks
	sentences := splitIntoSentences(content)

	// Filter sentences that contain wikilinks
	var wikilinkSentences []string
	for _, sentence := range sentences {
		if containsWikilinks(sentence) {
			wikilinkSentences = append(wikilinkSentences, sentence)
		}
	}

	// Limit to first 10 sentences for testing
	if len(wikilinkSentences) > 10 {
		wikilinkSentences = wikilinkSentences[:10]
	}

	var allIssues []AIIssue

	// Process each sentence separately
	for _, sentence := range wikilinkSentences {
		issues, err := processSentence(sentence)
		if err != nil {
			fmt.Printf("Error processing sentence: %v\n", err)
			continue
		}
		allIssues = append(allIssues, issues...)
	}

	return &AIResponse{Issues: allIssues}, nil
}

func splitIntoSentences(content string) []string {
	// Split by sentences (., !, ?) and paragraphs
	sentences := []string{}

	// First split by paragraphs
	paragraphs := strings.Split(content, "\n\n")

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		// Split paragraph by sentence endings
		current := ""
		for i, char := range paragraph {
			current += string(char)

			// Check if this is a sentence ending
			if char == '.' || char == '!' || char == '?' {
				// Look ahead to see if next char is space or end of string
				if i+1 >= len(paragraph) || paragraph[i+1] == ' ' || paragraph[i+1] == '\n' {
					sentences = append(sentences, strings.TrimSpace(current))
					current = ""
				}
			}
		}

		// Add remaining text if any
		if strings.TrimSpace(current) != "" {
			sentences = append(sentences, strings.TrimSpace(current))
		}
	}

	return sentences
}

func containsWikilinks(text string) bool {
	return strings.Contains(text, "[[") && strings.Contains(text, "]]")
}

func extractWikilinks(text string) []string {
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := re.FindAllString(text, -1)
	return matches
}

func processSentence(sentence string) ([]AIIssue, error) {
	// Extract original wikilinks from the sentence (before normalization)
	originalWikilinks := extractWikilinks(sentence)

	// Normalize sentence for AI processing
	normalizedSentence := string(mdloader.NormalizeWikilinks([]byte(sentence)))
	normalizedWikilinks := extractWikilinks(normalizedSentence)

	var issues []AIIssue

	// Process each wikilink (match original with normalized)
	for i, originalWikilink := range originalWikilinks {
		// Get corresponding normalized wikilink
		var normalizedWikilink string
		if i < len(normalizedWikilinks) {
			normalizedWikilink = normalizedWikilinks[i]
		} else {
			normalizedWikilink = originalWikilink // fallback
		}

		issue, err := checkWikilink(normalizedSentence, normalizedWikilink)
		if err != nil {
			fmt.Printf("Error checking wikilink %s: %v\n", normalizedWikilink, err)
			continue
		}

		if issue != nil {
			// Use original wikilink as marker for replacement
			issue.Marker = originalWikilink
			issues = append(issues, *issue)
		}
	}

	return issues, nil
}

func checkWikilink(sentence, wikilink string) (*AIIssue, error) {
	prompt := fmt.Sprintf("Sentence: %s\nWikilink: %s", sentence, wikilink)

	myThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(checkWikilinkPrompt).Format(types.M{}),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent(prompt),
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
	fmt.Printf("Response for wikilink %s: %s\n", wikilink, responseText)

	var checkResp WikilinkCheckResponse
	err = json.Unmarshal([]byte(responseText), &checkResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	// If no fix needed, return nil
	if !checkResp.NeedsFix {
		return nil, nil
	}

	// Return the issue with original wikilink as marker and AI's fix
	return &AIIssue{
		Marker:  wikilink,
		Fix:     checkResp.Fix,
		Comment: checkResp.Comment,
	}, nil
}
