package main

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestList(t *testing.T) {
	msg := &tg.Message{
		Message: `- Item 1
- Item 2

Paragraph text here.`,
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityItalic{
				Offset: 0,
				Length: 17, // "- Item 1\n- Item 2" (без trailing \n)
			},
			&tg.MessageEntityBold{
				Offset: 16, // символ "2"
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

func TestList2(t *testing.T) {
	msg := &tg.Message{
		Message: `- Item 1
- Item 2

Paragraph text here.`,
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityItalic{
				Offset: 0,
				Length: 17, // "- Item 1\n- Item 2" (без trailing \n)
			},
			&tg.MessageEntityBold{
				Offset: 16, // символ "2"
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
