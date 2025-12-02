package main

import (
	"fmt"
	"testing"
)

func TestDebugUTF16Positions(t *testing.T) {
	text := "- А кто ваш отец?\n- У моей семьи есть завод в Бернштейне\n- Какой?\n- Там производят седла\n- Седла. Такое всегда продается. Вас ждет светлое будущее. \n- Да, мне повезло.\n- Иии, вы ждете возвращение домой? Когда я вас отпущу?\n- Да, у меня уже все расписано. Мне передадут семейное дело. А вы?\n- А я солдат. Мой отец был офицером в этом полку. Участвовал в трех войнах при Бисмарке и все выиграл. В 71 - ом он пошел на Париж, вернулся в Германию героем. Я родился слишком поздно Бриксдорф. 50 лет без войны. А что же солдат без войны. \n- Вы были близки с отцом?\n- Может только в детстве. Человек рождается один, живет один, один и умирает. \n\nСложный фильм."

	// Count UTF-16 code units
	utf16pos := 0
	for i, r := range text {
		if utf16pos == 584 {
			end := i + 50
			if end > len(text) {
				end = len(text)
			}
			fmt.Printf("Position 584 (UTF-16) = byte %d, text: %q\n", i, text[i:end])
		}
		if utf16pos == 638 {
			end := i + 50
			if end > len(text) {
				end = len(text)
			}
			fmt.Printf("Position 638 (UTF-16) = byte %d, text: %q\n", i, text[i:end])
		}
		if r > 0xFFFF {
			utf16pos += 2
		} else {
			utf16pos++
		}
	}
	fmt.Printf("Total UTF-16 units: %d, Total bytes: %d\n", utf16pos, len(text))
}
