package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"trip2g/internal/logger"

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
- deps: array of step names that must complete before this step can run

## JSON Schema:

[
  {
    "name": "step_name",
    "type": "prompt",
    "prompt": "AI instruction text",
    "deps": []
  },
  {
    "name": "dependent_step", 
    "type": "prompt",
    "prompt": "AI instruction with {{.previous_step.output}}",
    "deps": ["previous_step"]
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
    "prompt": "Extract exactly 3 key topics from the text: {{.input.content}}",
    "deps": []
  },
  {
    "name": "first_summary",
    "type": "prompt",
    "prompt": "Reduce the ORIGINAL text {{.input.content}} by 50% while keeping these key topics: {{.extract_topics.output}}",
    "deps": ["extract_topics"]
  },
  {
    "name": "second_summary", 
    "type": "prompt",
    "prompt": "Reduce this text {{.first_summary.output}} by 50% while keeping these key topics: {{.extract_topics.output}}",
    "deps": ["first_summary", "extract_topics"]
  }
]

## Rules:
1. Use {{.input.content}} for original text (can be used in any step)
2. Reference previous step outputs with {{.step_name.output}}
3. For loops, each iteration processes {{.input.content}} with context from {{.step_name.output}}
4. Use descriptive step names
5. **IMPORTANT**: Add "deps" array for each step listing which steps must complete first
6. If a step uses {{.step_name.output}}, add "step_name" to deps array
7. Steps with empty deps [] can run in parallel
8. Return ONLY the JSON array of steps, no wrapper object

Convert the following pipeline description to JSON:
`

// Step represents a single pipeline step
type Step struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Prompt string   `json:"prompt"`
	Deps   []string `json:"deps"`
}

// StepResult holds the result of a step execution
type StepResult struct {
	StepName string
	Output   string
	Error    error
}

// PipelineMemory stores outputs from completed steps
type PipelineMemory struct {
	data map[string]string
}

func NewPipelineMemory(inputContent string) *PipelineMemory {
	return &PipelineMemory{
		data: map[string]string{
			"input.content": inputContent,
		},
	}
}

func (pm *PipelineMemory) Set(key, value string) {
	pm.data[key] = value
}

func (pm *PipelineMemory) Get(key string) (string, bool) {
	value, exists := pm.data[key]
	return value, exists
}

func (pm *PipelineMemory) GetAll() map[string]string {
	result := make(map[string]string)
	for k, v := range pm.data {
		result[k] = v
	}
	return result
}

// ExecutePipeline runs the pipeline with dependency management and parallelism.
func ExecutePipeline(ctx context.Context, steps []Step, inputContent string, log logger.Logger) (*PipelineMemory, error) {
	memory := NewPipelineMemory(inputContent)

	// Build dependency map
	depMap := make(map[string][]string)
	stepMap := make(map[string]Step)

	for _, step := range steps {
		depMap[step.Name] = step.Deps
		stepMap[step.Name] = step
	}

	// Execute steps in dependency order
	completed := make(map[string]bool)

	for len(completed) < len(steps) {
		// Find steps ready to execute (all dependencies completed)
		var readySteps []Step

		for _, step := range steps {
			if completed[step.Name] {
				continue
			}

			allDepsReady := true
			for _, dep := range step.Deps {
				if !completed[dep] {
					allDepsReady = false
					break
				}
			}

			if allDepsReady {
				readySteps = append(readySteps, step)
			}
		}

		if len(readySteps) == 0 {
			return nil, fmt.Errorf("dependency cycle detected or missing dependencies")
		}

		// Execute ready steps in parallel
		resultChan := executeStepsInParallel(ctx, readySteps, memory, log)

		// Collect results
		for i := 0; i < len(readySteps); i++ {
			result := <-resultChan
			if result.Error != nil {
				return nil, fmt.Errorf("step %s failed: %w", result.StepName, result.Error)
			}

			memory.Set(result.StepName+".output", result.Output)
			completed[result.StepName] = true

			log.Debug("Step completed", "step", result.StepName)
		}
	}

	return memory, nil
}

// executeStepsInParallel runs multiple steps concurrently using native Go concurrency.
func executeStepsInParallel(ctx context.Context, steps []Step, memory *PipelineMemory, log logger.Logger) <-chan StepResult {
	resultChan := make(chan StepResult, len(steps))

	var wg sync.WaitGroup

	// Execute each step in its own goroutine
	for _, step := range steps {
		wg.Add(1)
		go func(s Step) {
			defer wg.Done()

			result, err := executeStep(ctx, s, memory, log)
			if err != nil {
				log.Error("Step failed", "step", s.Name, "error", err)
				result = StepResult{StepName: s.Name, Error: err}
			}

			resultChan <- result
		}(step)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

// executeStep executes a single step with variable substitution.
func executeStep(ctx context.Context, step Step, memory *PipelineMemory, log logger.Logger) (StepResult, error) {
	log.Debug("Executing step", "step", step.Name)

	// Get available variables
	vars := memory.GetAll()
	for k, v := range vars {
		displayValue := v
		if v == "" {
			displayValue = "<empty>"
		} else {
			displayValue = truncateString(v, 50)
		}
		log.Debug("Variable available", "step", step.Name, "key", k, "value", displayValue)
	}

	// Replace variables in prompt
	prompt, err := replaceVariables(step.Prompt, vars)
	if err != nil {
		return StepResult{StepName: step.Name, Error: err}, err
	}

	log.Debug("Formatted prompt", "step", step.Name, "prompt", truncateString(prompt, 100))

	// Execute AI call
	myThread := thread.New().AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent(prompt),
		),
	)

	err = openai.New().WithMaxTokens(2048).Generate(ctx, myThread)
	if err != nil {
		return StepResult{StepName: step.Name, Error: err}, err
	}

	resMsg := myThread.LastMessage()
	if resMsg.Contents[0].Type != thread.ContentTypeText {
		return StepResult{StepName: step.Name, Error: fmt.Errorf("unexpected content type")}, fmt.Errorf("unexpected content type")
	}

	result, ok := resMsg.Contents[0].Data.(string)
	if !ok {
		return StepResult{StepName: step.Name, Error: fmt.Errorf("unexpected content data type")}, fmt.Errorf("unexpected content data type")
	}

	return StepResult{
		StepName: step.Name,
		Output:   result,
	}, nil
}

// replaceVariables substitutes {{.variable.name}} with actual values
func replaceVariables(promptTemplate string, variables map[string]string) (string, error) {
	// Simple string replacement for our specific format {{.key}}
	result := promptTemplate

	// Replace each variable in the template
	for key, value := range variables {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Check for any remaining unreplaced variables
	if strings.Contains(result, "{{.") && strings.Contains(result, "}}") {
		// Return error for unreplaced variables
		return result, fmt.Errorf("template contains unreplaced variables: %s", result)
	}

	return result, nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Test pipeline description
const testPipelineDescription = `
Создай пайплайн который:
1. Сначала извлекает 3 ключевые темы из оригинального текста {{.input.content}}
2. Первое сокращение: сокращает ОРИГИНАЛЬНЫЙ текст {{.input.content}} на 50%, сохраняя эти темы из шага 1
3. Второе сокращение: сокращает результат с шага 2 еще на 50%, сохраняя темы из шага 1  
4. После этого анализирует насколько СОКРАЩЕННЫЙ результат с шага 3 отвечает на вопрос "Как выглядит день автора?", используя также темы из шага 1
5. В конце критик дает альтернативный анализ, сравнивая ОРИГИНАЛЬНЫЙ текст {{.input.content}} и результат анализа с шага 4. Нужно вывести резутаты четвертого шага + критический анализ в конце.

Результат должен быть на русском языке. Укажи это в промпте на каждом шаге.
`

func resolveAI(content string, offset int, log logger.Logger) ([]byte, error) {
	ctx := context.Background()

	// Validate input
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	log.Info("Received content for AI processing", "content", truncateString(content, 100))

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

	err := openai.New().WithMaxTokens(2048).Generate(ctx, myThread)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pipeline: %w", err)
	}

	resMsg := myThread.LastMessage()
	pipelineJSON, ok := resMsg.Contents[0].Data.(string)
	if !ok {
		return nil, fmt.Errorf("failed to get pipeline JSON from AI response")
	}

	// Step 2: Parse and validate JSON array
	var steps []Step
	err = json.Unmarshal([]byte(pipelineJSON), &steps)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline JSON: %w", err)
	}

	// Step 4: Execute the pipeline
	log.Info("Executing pipeline", "steps", len(steps))
	memory, err := ExecutePipeline(ctx, steps, content, log)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Step 5: Get final result
	finalResults := make(map[string]string)
	for _, step := range steps {
		if output, exists := memory.Get(step.Name + ".output"); exists {
			finalResults[step.Name] = output
		}
	}

	// Return the final step's output (usually the last step)
	if len(steps) > 0 {
		lastStep := steps[len(steps)-1]
		if finalOutput, exists := memory.Get(lastStep.Name + ".output"); exists {
			log.Info("Pipeline completed", "final_step", lastStep.Name)
			return []byte(finalOutput), nil
		}
	}

	// Fallback: return all results as JSON
	allResultsJSON, err := json.MarshalIndent(finalResults, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal final results: %w", err)
	}

	return allResultsJSON, nil
}
