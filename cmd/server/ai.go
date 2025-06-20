package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/thread"
)

const pipelineBuilderPrompt = `
# JSON Pipeline Builder

You are a JSON pipeline configuration builder. Your task is to convert natural language descriptions of AI processing pipelines into structured JSON.

## Pipeline Structure

A pipeline has memory that stores all step outputs. Each step can access previous outputs using variables.

Each step has:
- name: unique identifier
- type: "prompt" or "loop"  
- prompt: the AI instruction (can use {{.step_name.output}} from memory)
- iterations: (only for loop type) how many times to repeat

## JSON Schema:

[
  {
    "name": "step_name",
    "type": "prompt",
    "prompt": "AI instruction text"
  },
  {
    "name": "loop_step", 
    "type": "loop",
    "iterations": 2,
    "prompt": "AI instruction with {{.previous_step.output}}"
  }
]

## Memory Variables:
- {{.input.content}} - always available, the original user input text
- {{.step_name.output}} - result from any previous step named "step_name"
- Variables must use dot notation: {{.step_name.output}}

## Example:

Input: "First extract 3 key topics, then summarize the ORIGINAL text by 50%, then summarize the result by 50% again, keeping the topics"

Output:
[
  {
    "name": "extract_topics",
    "type": "prompt",
    "prompt": "Extract exactly 3 key topics from the text: {{.input.content}}"
  },
  {
    "name": "first_summary",
    "type": "prompt",
    "prompt": "Reduce the ORIGINAL text {{.input.content}} by 50% while keeping these key topics: {{.extract_topics.output}}"
  },
  {
    "name": "second_summary", 
    "type": "prompt",
    "prompt": "Reduce this text {{.first_summary.output}} by 50% while keeping these key topics: {{.extract_topics.output}}"
  }
]

## Rules:
1. Use {{.input.content}} for original text (can be used in any step)
2. Reference previous step outputs with {{.step_name.output}}
3. For loops, each iteration processes {{.input.content}} with context from {{.step_name.output}}
4. Use descriptive step names
5. Return ONLY the JSON array of steps, no wrapper object

Convert the following pipeline description to JSON:
`

// Test pipeline description
const testPipelineDescription = `
Создай пайплайн который:
1. Сначала извлекает 3 ключевые темы из оригинального текста {{.input.content}}
2. Первое сокращение: сокращает ОРИГИНАЛЬНЫЙ текст {{.input.content}} на 50%, сохраняя эти темы из шага 1
3. Второе сокращение: сокращает результат с шага 2 еще на 50%, сохраняя темы из шага 1  
4. После этого анализирует насколько СОКРАЩЕННЫЙ результат с шага 3 отвечает на вопрос "Как выглядит день автора?", используя также темы из шага 1
5. В конце критик дает альтернативный анализ, сравнивая ОРИГИНАЛЬНЫЙ текст {{.input.content}} и результат анализа с шага 4
`

func resolveAI(content string, offset int) ([]byte, error) {
	// Step 1: Build pipeline JSON from description
	myThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(pipelineBuilderPrompt),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent(testPipelineDescription),
		),
	)

	err := openai.New().WithMaxTokens(2048).Generate(context.Background(), myThread)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pipeline: %w", err)
	}

	resMsg := myThread.LastMessage()
	pipelineJSON := resMsg.Contents[0].Data.(string)

	// Step 2: Parse and validate JSON array
	var pipelineSteps []map[string]interface{}
	err = json.Unmarshal([]byte(pipelineJSON), &pipelineSteps)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline JSON: %w", err)
	}

	// Step 3: Pretty print for debug
	prettyJSON, err := json.MarshalIndent(pipelineSteps, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format JSON: %w", err)
	}

	fmt.Println("Generated Pipeline Steps:")
	fmt.Println(string(prettyJSON))
	fmt.Printf("Number of steps: %d\n", len(pipelineSteps))

	// Print step details for debugging
	for i, step := range pipelineSteps {
		fmt.Printf("Step %d: %s (%s)\n", i+1, step["name"], step["type"])
	}

	// For now, just return the JSON
	// Later we will add execution logic
	return prettyJSON, nil
}
