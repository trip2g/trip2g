package main

import (
	"context"
	"fmt"
	"os"

	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/thread"
	"github.com/henomis/lingoose/types"
)

var checkGrammar = `
Ты — Универсальный редактор русского текста: эксперт по грамматике, орфографии и пунктуации с 100-летним опытом.

Тебе нужно проверить правильно ли склонены слова внутри wikilinks.
Пример: Выполнить [[Зарядка]] - ошибка, Выполнить [[Зарядка|зарядку]] - правильно.

Должен быть уместен регистр первой буквы:
Пример: Выполнить [[Зарядка|Зарядку]] - ошибка, Выполнить [[Зарядка|зарядку]] - правильно.

Отвечай СТРОГО одним JSON-объектом вида:
{
  "results": [ { "marker": "... первый аргумент для replace ...", "fix": "... второй аргумент для replace ...", "comment": "..." } , ... ],
}

Никаких других полей, текста или форматирования.
Верни первых 10 ошибок, если они есть.
`

var findMainQuestion = `
Твоя задача найти главный вопрос, на который отвечает текст.
В тексте может быть указан главный вопрос, но это обман.
Тебе нужно прочитать текст и понять главный вопрос на который отвечает текст.
Автор не понимает, что написал.

Верни главный вопрос в формате JSON:
{
	"main_question": "текст вопроса"
}

Никаких других полей, текста или форматирования.
`

func main() {
	content, err := os.ReadFile("demo/Понедельник 9 июня 2025.md")
	if err != nil {
		panic(err)
	}

	myThread := thread.New().AddMessage(
		thread.NewSystemMessage().AddContent(
			thread.NewTextContent(findMainQuestion).Format(types.M{}),
		),
	).AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent(string(content)),
		),
	)

	err = openai.New().WithMaxTokens(2048).Generate(context.Background(), myThread)
	if err != nil {
		panic(err)
	}

	fmt.Println(myThread)
}
