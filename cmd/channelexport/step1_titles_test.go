package main

import "testing"

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "1584 - bold marker then dash",
			content: `---
telegram_publish_message_id: 1584
---

*- У тебя не бывало ощущения что ничего хорошего тебя не ждет?
- Да ничего и не было.`,
			expected: "У тебя не бывало ощущения что ничего",
		},
		{
			name: "882 - bold marker then skin tone emoji",
			content: `---
telegram_publish_message_id: 882
---

*👦🏾 Нет, мой совет мужчинам, не будьте уязвимыми с женщинами
👧🏻 Ты видимо никогда не был влюблен`,
			expected: "Нет, мой совет мужчинам, не будьте уязвимыми",
		},
		{
			name: "1317 - custom emoji then bold",
			content: `---
telegram_publish_message_id: 1317
---

![➡️](tg://emoji?id=5204293223538764272) **Сила, уважение, дисциплина

***- Матрос вас** толкнул`,
			expected: "Сила, уважение, дисциплина",
		},
		{
			name: "1536 - malformed custom emoji with HTML",
			content: `---
telegram_publish_message_id: 1536
---

![<u](tg://emoji?id=5204293223538764272)>➡️</u><u>**Куда продпадает дисциплин</u>а**

Часто мне говорят`,
			expected: "Куда продпадает дисциплина",
		},
		{
			name: "simple text",
			content: `---
telegram_publish_message_id: 123
---

Простой заголовок без форматирования

Второй параграф`,
			expected: "Простой заголовок без форматирования",
		},
		{
			name: "leading emoji",
			content: `---
telegram_publish_message_id: 123
---

➡️ Базовая потребность человека

Текст`,
			expected: "Базовая потребность человека",
		},
		{
			name: "markdown link in title",
			content: `---
telegram_publish_message_id: 123
---

Суть метода [Второй мозг](https://t.me/ryspaisensei/769) очень проста

Текст`,
			expected: "Суть метода Второй мозг очень проста",
		},
		{
			name: "wine emoji in middle",
			content: `---
telegram_publish_message_id: 123
---

Вино🍷 это напиток богов

Текст`,
			expected: "Вино🍷 это напиток богов",
		},
		{
			name: "trailing punctuation",
			content: `---
telegram_publish_message_id: 123
---

Это важный вопрос...

Текст`,
			expected: "Это важный вопрос",
		},
		{
			name: "multiple dashes at start",
			content: `---
telegram_publish_message_id: 123
---

- - - Могу я вам как то помочь?

Текст`,
			expected: "Могу я вам как то помочь",
		},
		{
			name: "618 - italic quote dash",
			content: `---
telegram_publish_message_id: 618
---

*"- Мы больше не дети играющие в пиратов"* **Ронанао Зорро

**#ванпис`,
			expected: "Мы больше не дети играющие в пиратов",
		},
		{
			name: "1563 - timecode at start",
			content: `---
telegram_publish_message_id: 1563
---

00:00 - о закрытом канале
00:40 - каким я вижу мой закрытый канал`,
			expected: "о закрытом канале",
		},
		{
			name: "numbered emoji prefix",
			content: `---
telegram_publish_message_id: 1234
---

![1️⃣](https://ce.trip2g.com/5461128548397884758.webp). **Нейротипология

**Начал смотреть курс по Нейротипологии Ивана #Лимарев.`,
			expected: "Нейротипология",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content)
			if got != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}
