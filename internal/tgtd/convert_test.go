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
	expected := "Hello ![](https://ce.trip2g.com/12345.webp) world"

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
 - *Может только в детстве. **Человек рождается один, живет один, один и умирает. ***

Сложный фильм. Мне понравился. Генерал яркая демонстрация человека который до конца не встал взрослым, и не отделился от фигуры отца чтобы идти своей дорогой. Но с другой стороны, фигура отца сделала его генералом. 

Х/Ф "На западном фронте без перемен".

![](https://ce.trip2g.com/5460736117236048513.webp)[Отношения с отцом](https://t.me/ryspaisensei/883)
![](https://ce.trip2g.com/5460736117236048513.webp)[Обязан ли отец любить сына?](https://t.me/ryspaisensei/328)
![](https://ce.trip2g.com/5460736117236048513.webp)[Я покажу своему сыну путь](https://t.me/ryspaisensei/954)

![](https://ce.trip2g.com/5463038705038007921.webp) **Фрагменты:** [Дай человеку власть, он станет животным](https://t.me/ryspaisensei/1054)

![](https://ce.trip2g.com/5215478503788520683.webp) [Кино 23](https://t.me/ryspaisensei/1235) |  #кино #отец #сепарация`

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
