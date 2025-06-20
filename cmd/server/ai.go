package main

import (
	"context"

	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/thread"
	"github.com/henomis/lingoose/types"
)

const prompt0 = `
# Задача: Извлечение ключевых тем

Проанализируй предоставленный текст и выдели ровно 3 ключевые темы, которые наиболее точно отражают его основное содержание.

## Требования:
1. Каждая тема должна быть сформулирована как краткая фраза (2-5 слов)
2. Темы должны охватывать разные аспекты текста
3. Избегай повторений и дублирования смысла
4. Фокусируйся на главных идеях, а не на деталях
5. Используй существительные и ключевые понятия из текста

## Формат ответа:
Верни только список из трех пунктов без дополнительных объяснений:

1. [Первая ключевая тема]
2. [Вторая ключевая тема]
3. [Третья ключевая тема]

Анализируй внимательно и выбирай наиболее значимые темы.
`

const prompt1 = `
Сократи текст на половину, сохранив основные ключевые темы:
{{.topics}}
`

const prompt2 = `
# Задача: Анализ релевантности текста

У тебя есть:
1. Ключевые темы текста:
{{.topics}}

2. Сокращенная версия текста:
{{.text}}

## Вопросы для анализа:

1. Насколько процентов (0-100%) данный текст отвечает на вопрос: "Как выглядит день автора базы?"
   - Опиши какие аспекты дня раскрыты, а какие отсутствуют

2. Дай 3-5 конкретных рекомендаций по улучшению текста, чтобы он лучше отвечал на этот вопрос

## Формат ответа:
Релевантность: [X]%

Раскрытые аспекты:
- [аспект 1]
- [аспект 2]

Отсутствующие аспекты:
- [аспект 1]
- [аспект 2]

Рекомендации по улучшению:
1. [конкретная рекомендация]
2. [конкретная рекомендация]
3. [конкретная рекомендация]
`

const prompt3 = `
# Задача: Критический анализ и альтернативная версия

Ты - опытный критик и редактор текстов. Проанализируй оригинальный текст и результат его обработки.

## Исходные данные:

### Оригинальный текст:
{{.original}}

### Результат анализа:
{{.analysis}}

## Твоя задача:

1. Критически оцени результат анализа:
   - Что упущено из виду?
   - Какие выводы кажутся поверхностными?
   - Где анализ мог бы быть глубже?

2. Предложи свою альтернативную версию анализа, которая:
   - Более точно отвечает на вопрос "Как выглядит день автора?"
   - Выделяет неочевидные, но важные детали
   - Дает более практичные рекомендации

## Формат ответа:

### Критика предыдущего анализа:
[Укажи 3-4 конкретных недостатка]

### Мой альтернативный анализ:

Релевантность: [X]% (с обоснованием)

Ключевые инсайты о дне автора:
- [неочевидный, но важный аспект]
- [скрытая рутина или паттерн]
- [эмоциональная составляющая дня]

Практические рекомендации:
1. [очень конкретная и выполнимая рекомендация]
2. [рекомендация с примером реализации]
3. [рекомендация, учитывающая контекст автора]

### Главный вывод:
[Одно предложение о том, что действительно важно понять про день автора]
`

func resolveAI(content string, offset int) ([]byte, error) {
	// Step 1: Extract key topics
	myThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(prompt0),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent(content),
		),
	)

	err := openai.New().WithMaxTokens(2048).Generate(context.Background(), myThread)
	if err != nil {
		panic(err)
	}

	resMsg := myThread.LastMessage()
	topics := resMsg.Contents[0].Data.(string)

	// Step 2: Apply reductions in a loop
	currentText := content
	reductionIterations := 2

	for i := 0; i < reductionIterations; i++ {
		myThread := thread.New().AddMessage(
			thread.NewSystemMessage().AddContent(
				thread.NewTextContent(prompt1).Format(types.M{"topics": topics}),
			),
		).AddMessage(
			thread.NewUserMessage().AddContent(
				thread.NewTextContent(currentText),
			),
		)

		err = openai.New().WithMaxTokens(2048).Generate(context.Background(), myThread)
		if err != nil {
			panic(err)
		}

		resMsg := myThread.LastMessage()
		currentText = resMsg.Contents[0].Data.(string)
	}

	// Step 3: Analyze relevance with topics and reduced text
	finalThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(prompt2).Format(types.M{
				"topics": topics,
				"text":   currentText,
			}),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent("Проанализируй текст"),
		),
	)

	err = openai.New().WithMaxTokens(2048).Generate(context.Background(), finalThread)
	if err != nil {
		panic(err)
	}

	finalMsg := finalThread.LastMessage()
	analysis := finalMsg.Contents[0].Data.(string)

	// Step 4: Critical review with original text and analysis
	criticThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(prompt3).Format(types.M{
				"original": content,
				"analysis": analysis,
			}),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent("Дай критический анализ"),
		),
	)

	err = openai.New().WithMaxTokens(2048).Generate(context.Background(), criticThread)
	if err != nil {
		panic(err)
	}

	criticMsg := criticThread.LastMessage()
	critique := criticMsg.Contents[0].Data.(string)

	return []byte(critique), nil
}
