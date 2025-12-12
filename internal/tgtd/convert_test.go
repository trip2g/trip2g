package tgtd

import (
	"testing"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"
)

func TestConvert_List(t *testing.T) {
	msg := &tg.Message{
		Message: `- Item 1
- Item 2

Paragraph text here.`,
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityItalic{
				Offset: 0,
				Length: 17,
			},
			&tg.MessageEntityBold{
				Offset: 16,
				Length: 1,
			},
		},
	}

	res := Convert(msg)

	expected := ` - *Item 1*
 - *Item **2***

Paragraph text here.`

	if res != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestConvert_Strikethrough(t *testing.T) {
	msg := &tg.Message{
		Message: "Hello strikethrough world",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityStrike{
				Offset: 6,
				Length: 13,
			},
		},
	}

	res := Convert(msg)
	expected := "Hello ~~strikethrough~~ world"

	if res != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestConvert_Underline(t *testing.T) {
	msg := &tg.Message{
		Message: "Hello underline world",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityUnderline{
				Offset: 6,
				Length: 9,
			},
		},
	}

	res := Convert(msg)
	expected := "Hello <u>underline</u> world"

	if res != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestConvert_Spoiler(t *testing.T) {
	msg := &tg.Message{
		Message: "Hello spoiler world",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntitySpoiler{
				Offset: 6,
				Length: 7,
			},
		},
	}

	res := Convert(msg)
	expected := "Hello ||spoiler|| world"

	if res != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestConvert_TextURL(t *testing.T) {
	msg := &tg.Message{
		Message: "Click here for more",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityTextURL{
				Offset: 6,
				Length: 4,
				URL:    "https://example.com",
			},
		},
	}

	res := Convert(msg)
	expected := "Click [here](https://example.com) for more"

	if res != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestConvert_CustomEmoji(t *testing.T) {
	msg := &tg.Message{
		Message: "Hello 🔥 world",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityCustomEmoji{
				Offset:     6,
				Length:     2,
				DocumentID: 12345,
			},
		},
	}

	res := Convert(msg)
	expected := "Hello ![🔥|20x20](https://ce.trip2g.com/12345.webp) world"

	if res != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestConvert_Empty(t *testing.T) {
	msg := &tg.Message{
		Message:  "",
		Entities: nil,
	}

	res := Convert(msg)

	if res != "" {
		t.Errorf("expected empty string, got: %s", res)
	}
}

func TestConvert_CustomEmojiBold(t *testing.T) {
	msg := &tg.Message{
		Message: `- А кто ваш отец?
- У моей семьи есть завод в Бернштейне
- Какой?
- Там производят седла
- Седла. Такое всегда продается. Вас ждет светлое будущее. 
- Да, мне повезло.
- Иии, вы ждете возвращение домой? Когда я вас отпущу?
- Да, у меня уже все расписано. Мне передадут семейное дело. А вы?
- А я солдат. Мой отец был офицером в этом полку. Участвовал в трех войнах при Бисмарке и все выиграл. В 71 - ом он пошел на Париж, вернулся в Германию героем. Я родился слишком поздно Бриксдорф. 50 лет без войны. А что же солдат без войны. 
- Вы были близки с отцом?
- Может только в детстве. Человек рождается один, живет один, один и умирает. 

Сложный фильм. Мне понравился. Генерал яркая демонстрация человека который до конца не встал взрослым, и не отделился от фигуры отца чтобы идти своей дорогой. Но с другой стороны, фигура отца сделала его генералом. 

Х/Ф "На западном фронте без перемен".

🟠Отношения с отцом
🟠Обязан ли отец любить сына?
🟠Я покажу своему сыну путь

⚡️ Фрагменты: Дай человеку власть, он станет животным

💎 Кино 23 |  #кино #отец #сепарация`,
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityItalic{Offset: 0, Length: 638},
			&tg.MessageEntityBold{Offset: 584, Length: 54},
			&tg.MessageEntityCustomEmoji{Offset: 894, Length: 2, DocumentID: 5460736117236048513},
			&tg.MessageEntityTextURL{Offset: 896, Length: 18, URL: "https://t.me/ryspaisensei/883"},
			&tg.MessageEntityCustomEmoji{Offset: 914, Length: 2, DocumentID: 5460736117236048513},
			&tg.MessageEntityTextURL{Offset: 916, Length: 28, URL: "https://t.me/ryspaisensei/328"},
			&tg.MessageEntityCustomEmoji{Offset: 944, Length: 2, DocumentID: 5460736117236048513},
			&tg.MessageEntityTextURL{Offset: 946, Length: 27, URL: "https://t.me/ryspaisensei/954"},
			&tg.MessageEntityBold{Offset: 973, Length: 2},
			&tg.MessageEntityCustomEmoji{Offset: 973, Length: 2, DocumentID: 5463038705038007921},
			&tg.MessageEntityBold{Offset: 975, Length: 11},
			&tg.MessageEntityTextURL{Offset: 987, Length: 41, URL: "https://t.me/ryspaisensei/1054"},
			&tg.MessageEntityBold{Offset: 1028, Length: 2},
			&tg.MessageEntityCustomEmoji{Offset: 1028, Length: 2, DocumentID: 5215478503788520683},
			&tg.MessageEntityBold{Offset: 1030, Length: 1},
			&tg.MessageEntityTextURL{Offset: 1031, Length: 7, URL: "https://t.me/ryspaisensei/1235"},
			&tg.MessageEntityBold{Offset: 1031, Length: 7},
			&tg.MessageEntityHashtag{Offset: 1042, Length: 5},
			&tg.MessageEntityHashtag{Offset: 1048, Length: 5},
			&tg.MessageEntityHashtag{Offset: 1054, Length: 10},
		},
	}

	res := Convert(msg)
	expected := ` - *А кто ваш отец?*
 - *У моей семьи есть завод в Бернштейне*
 - *Какой?*
 - *Там производят седла*
 - *Седла. Такое всегда продается. Вас ждет светлое будущее. *
 - *Да, мне повезло.*
 - *Иии, вы ждете возвращение домой? Когда я вас отпущу?*
 - *Да, у меня уже все расписано. Мне передадут семейное дело. А вы?*
 - *А я солдат. Мой отец был офицером в этом полку. Участвовал в трех войнах при Бисмарке и все выиграл. В 71 - ом он пошел на Париж, вернулся в Германию героем. Я родился слишком поздно Бриксдорф. 50 лет без войны. А что же солдат без войны. *
 - *Вы были близки с отцом?*
 - *Может только в детстве. **Человек рождается один, живет один, один и умирает.*** 

Сложный фильм. Мне понравился. Генерал яркая демонстрация человека который до конца не встал взрослым, и не отделился от фигуры отца чтобы идти своей дорогой. Но с другой стороны, фигура отца сделала его генералом. 

Х/Ф "На западном фронте без перемен".

![🟠|20x20](https://ce.trip2g.com/5460736117236048513.webp)[Отношения с отцом](https://t.me/ryspaisensei/883)
![🟠|20x20](https://ce.trip2g.com/5460736117236048513.webp)[Обязан ли отец любить сына?](https://t.me/ryspaisensei/328)
![🟠|20x20](https://ce.trip2g.com/5460736117236048513.webp)[Я покажу своему сыну путь](https://t.me/ryspaisensei/954)

![⚡️|20x20](https://ce.trip2g.com/5463038705038007921.webp) **Фрагменты**: [Дай человеку власть, он станет животным](https://t.me/ryspaisensei/1054)

![💎|20x20](https://ce.trip2g.com/5215478503788520683.webp) [Кино 23](https://t.me/ryspaisensei/1235) |  #кино #отец #сепарация`

	require.Equal(t, expected, res)
}

func TestConvert_Poll(t *testing.T) {
	msg := &tg.Message{
		Message:  "",
		Entities: nil,
		Media: &tg.MessageMediaPoll{
			Poll: tg.Poll{
				ID:   123,
				Quiz: true,
				Question: tg.TextWithEntities{
					Text: "What is the capital of France?",
				},
				Answers: []tg.PollAnswer{
					{Text: tg.TextWithEntities{Text: "London"}, Option: []byte("0")},
					{Text: tg.TextWithEntities{Text: "Paris"}, Option: []byte("1")},
					{Text: tg.TextWithEntities{Text: "Berlin"}, Option: []byte("2")},
				},
			},
			Results: tg.PollResults{
				Results: []tg.PollAnswerVoters{
					{Option: []byte("0"), Correct: false, Voters: 10},
					{Option: []byte("1"), Correct: true, Voters: 50},
					{Option: []byte("2"), Correct: false, Voters: 5},
				},
			},
		},
	}

	res := Convert(msg)
	expected := `**What is the capital of France?**

- [ ] London
- [x] Paris
- [ ] Berlin`

	require.Equal(t, expected, res)
}

func TestConvert_InlineCode(t *testing.T) {
	msg := &tg.Message{
		Message: "Use docker-compose.yml file and run compose command",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityCode{Offset: 4, Length: 18},
			&tg.MessageEntityCode{Offset: 36, Length: 7},
		},
	}

	res := Convert(msg)
	expected := "Use `docker-compose.yml` file and run `compose` command"

	require.Equal(t, expected, res)
}

func TestConvert_CodeBlock(t *testing.T) {
	msg := &tg.Message{
		Message: "Run this command:\n# Comment\ndocker compose config",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityPre{Offset: 18, Length: 32, Language: "bash"},
		},
	}

	res := Convert(msg)
	expected := "Run this command:\n```bash\n# Comment\ndocker compose config\n```"

	require.Equal(t, expected, res)
}

func TestConvert_CodeBlockNoLanguage(t *testing.T) {
	msg := &tg.Message{
		Message: "Example:\nsome code here",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityPre{Offset: 9, Length: 14, Language: ""},
		},
	}

	res := Convert(msg)
	expected := "Example:\n```\nsome code here\n```"

	require.Equal(t, expected, res)
}

func TestConvert_BoldWithTrailingSpace(t *testing.T) {
	// Bold entity that ends with a space - space should be moved outside the bold
	// Original: "**Главная причина тревоги — это желание преодолеть неопределенность. **Искусственный"
	// Expected: "**Главная причина тревоги — это желание преодолеть неопределенность.** Искусственный"
	msg := &tg.Message{
		Message: "Главная причина тревоги — это желание преодолеть неопределенность. Искусственный интеллект многократно усиливает эту неопределенность, делая будущее еще более туманным.",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityBold{
				Offset: 0,
				Length: 67, // includes trailing space
			},
		},
	}

	res := Convert(msg)
	expected := "**Главная причина тревоги — это желание преодолеть неопределенность.** Искусственный интеллект многократно усиливает эту неопределенность, делая будущее еще более туманным."

	require.Equal(t, expected, res)
}

func TestConvert_PollWithText(t *testing.T) {
	msg := &tg.Message{
		Message:  "Check out this quiz:",
		Entities: nil,
		Media: &tg.MessageMediaPoll{
			Poll: tg.Poll{
				ID:   456,
				Quiz: true,
				Question: tg.TextWithEntities{
					Text: "2 + 2 = ?",
				},
				Answers: []tg.PollAnswer{
					{Text: tg.TextWithEntities{Text: "3"}, Option: []byte("a")},
					{Text: tg.TextWithEntities{Text: "4"}, Option: []byte("b")},
					{Text: tg.TextWithEntities{Text: "5"}, Option: []byte("c")},
				},
			},
			Results: tg.PollResults{
				Results: []tg.PollAnswerVoters{
					{Option: []byte("a"), Correct: false},
					{Option: []byte("b"), Correct: true},
					{Option: []byte("c"), Correct: false},
				},
			},
		},
	}

	res := Convert(msg)
	expected := `Check out this quiz:

**2 + 2 = ?**

- [ ] 3
- [x] 4
- [ ] 5`

	require.Equal(t, expected, res)
}

func TestConvert_BoldWithLeadingPunctuation(t *testing.T) {
	// Telegram bold entity starts with punctuation - must be moved outside
	// CommonMark: opening ** followed by punctuation doesn't work if preceded by word char
	//
	// Input text: "вернулся в субботу, но ментально вернулся к жизни только сегодня, после 2-х часовой сесии на ноги."
	// Bold 1: ", но ментально вернулся к жизни только сегодня" (starts with comma!)
	// Bold 2: "после 2-х часовой сесии на ноги." (control - should work fine)
	//
	// Naive output: "вернулся в субботу**, но ... сегодня**, **после...**"
	// The **,  doesn't open bold because ** is followed by punctuation
	//
	// Expected: comma moves outside bold markers
	msg := &tg.Message{
		Message: "вернулся в субботу, но ментально вернулся к жизни только сегодня, после 2-х часовой сесии на ноги.",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityBold{
				Offset: 18, // ", но ментально вернулся к жизни только сегодня" (UTF-16)
				Length: 47,
			},
			&tg.MessageEntityBold{
				Offset: 66, // "после 2-х часовой сесии на ноги."
				Length: 32,
			},
		},
	}

	res := Convert(msg)
	expected := "вернулся в субботу, **но ментально вернулся к жизни только сегодня**, **после 2-х часовой сесии на ноги.**"

	require.Equal(t, expected, res)
}

func TestConvert_BoldWithTrailingColonSpaceNewline(t *testing.T) {
	// Bold entity ends with ": \n" - colon, space, and newline should all be trimmed
	// Real case from Telegram message 1719
	msg := &tg.Message{
		Message: "➡️Что там есть  помимо этого: \n🟣Эфиры",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityCustomEmoji{Offset: 0, Length: 2, DocumentID: 5974249837439224721},
			&tg.MessageEntityBold{Offset: 2, Length: 29}, // "Что там есть  помимо этого: \n" (29 UTF-16 units)
			&tg.MessageEntityCustomEmoji{Offset: 31, Length: 2, DocumentID: 5404867479002426748},
		},
	}

	res := Convert(msg)
	expected := "![➡️|20x20](https://ce.trip2g.com/5974249837439224721.webp)**Что там есть  помимо этого**: \n![🟣|20x20](https://ce.trip2g.com/5404867479002426748.webp)Эфиры"

	require.Equal(t, expected, res)
}

func TestConvert_EscapeMarkdown(t *testing.T) {
	// Literal markdown chars in text should be escaped
	msg := &tg.Message{
		Message:  "пи**ец, переменная_имя, и `код`",
		Entities: nil,
	}

	res := Convert(msg)
	expected := "пи\\*\\*ец, переменная\\_имя, и \\`код\\`"

	require.Equal(t, expected, res)
}

func TestConvert_NestedBoldItalicTrailingSpace(t *testing.T) {
	// Bold inside Italic, bold ends with trailing space before newline
	// Italic continues after bold ends
	// Should close as *** not ** *
	msg := &tg.Message{
		Message: "intro bold \nmore italic",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityItalic{Offset: 0, Length: 23}, // whole string
			&tg.MessageEntityBold{Offset: 6, Length: 6},    // "bold \n"
		},
	}

	res := Convert(msg)
	// Should be: *intro **bold*** \n*more italic*
	// Not: *intro **bold** *\n*more italic*
	expected := "*intro **bold*** \n*more italic*"

	require.Equal(t, expected, res)
}

func TestConvert_TextURLWithNewline(t *testing.T) {
	// TextURL entity incorrectly spans into next line (Telegram quirk)
	// Cut at first newline, leading emoji moved outside link
	msg := &tg.Message{
		Message: "тут:\n➖ Природа стресса\n➖Три фазы",
		Entities: []tg.MessageEntityClass{
			// "➖ Природа стресса\n➖" - link grabbed emoji and next line
			&tg.MessageEntityTextURL{Offset: 5, Length: 19, URL: "https://example.com/1"},
			&tg.MessageEntityTextURL{Offset: 24, Length: 10, URL: "https://example.com/2"},
		},
	}

	res := Convert(msg)
	// Emoji outside link, newline preserved, rest stays outside
	expected := "тут:\n➖ [Природа стресса](https://example.com/1)\n➖[Три фазы](https://example.com/2)"

	require.Equal(t, expected, res)
}

func TestConvert_HashtagSpacing(t *testing.T) {
	// Hashtag should have space before it if preceded by non-space
	msg := &tg.Message{
		Message: "text⚡️#дневник22",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityHashtag{Offset: 6, Length: 10},
		},
	}

	res := Convert(msg)
	expected := "text⚡️ #дневник22"

	require.Equal(t, expected, res)
}

func TestConvert_TextURLEscapeEquals(t *testing.T) {
	// Equals signs in link text should be escaped to avoid Setext header interpretation
	msg := &tg.Message{
		Message: "text\n====\nmore",
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityTextURL{Offset: 5, Length: 4, URL: "https://example.com"},
		},
	}

	res := Convert(msg)
	expected := "text\n[\\=\\=\\=\\=](https://example.com)\nmore"

	require.Equal(t, expected, res)
}

func TestConvert_EqualsLineNeedsBlankBefore(t *testing.T) {
	// Line of = after text needs blank line before to avoid Setext header
	msg := &tg.Message{
		Message:  "header\n====\ntext",
		Entities: nil,
	}

	res := Convert(msg)
	expected := "header\n\n====\ntext"

	require.Equal(t, expected, res)
}
