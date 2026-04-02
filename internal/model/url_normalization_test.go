package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestURLNormalizationMethod_Valid(t *testing.T) {
	require.True(t, URLNormCyrillic.Valid())
	require.True(t, URLNormSimpleTranslit.Valid())
	require.True(t, URLNormRusskayaLatinica2.Valid())

	// v1 is deprecated, not user-selectable
	require.False(t, URLNormRusskayaLatinica.Valid())
	require.False(t, URLNormalizationMethod("invalid").Valid())
	require.False(t, URLNormalizationMethod("").Valid())
}

func TestURLNormalizationMethod_Apply(t *testing.T) {
	input := "/поиск_астрономия"

	require.Equal(t, "/поиск_астрономия", URLNormCyrillic.Apply(input))
	require.Equal(t, "/poisk_astronomija", URLNormSimpleTranslit.Apply(input))
	require.Equal(t, "/poisk_astronomiya", URLNormRusskayaLatinica.Apply(input))
	require.Equal(t, "/poisk_astronomiya", URLNormRusskayaLatinica2.Apply(input))

	// Default case falls back to rl2
	require.Equal(t, "/poisk_astronomiya", URLNormalizationMethod("unknown").Apply(input))
}

func TestAllURLVariants(t *testing.T) {
	variants := AllURLVariants("/моя_страница")

	require.Len(t, variants, 4)
	require.Equal(t, "/моя_страница", variants[URLNormCyrillic])
	require.Equal(t, "/moja_stranica", variants[URLNormSimpleTranslit])
	require.Equal(t, "/moya_stranica", variants[URLNormRusskayaLatinica])
	require.Equal(t, "/moya_stranica", variants[URLNormRusskayaLatinica2])
}

func TestPermalinkEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ascii path", "/hello_world", "/hello_world"},
		{"cyrillic path", "/поиск", "/%D0%BF%D0%BE%D0%B8%D1%81%D0%BA"},
		{"nested cyrillic", "/раздел/страница", "/%D1%80%D0%B0%D0%B7%D0%B4%D0%B5%D0%BB/%D1%81%D1%82%D1%80%D0%B0%D0%BD%D0%B8%D1%86%D0%B0"},
		{"root path", "/", "/"},
		{"mixed", "/docs/моя_заметка", "/docs/%D0%BC%D0%BE%D1%8F_%D0%B7%D0%B0%D0%BC%D0%B5%D1%82%D0%BA%D0%B0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, PermalinkEncode(tt.input))
		})
	}
}

func TestNoteViewPermalinkEncoded(t *testing.T) {
	n := &NoteView{Permalink: "/поиск_астрономия"}
	require.Equal(t, "/%D0%BF%D0%BE%D0%B8%D1%81%D0%BA_%D0%B0%D1%81%D1%82%D1%80%D0%BE%D0%BD%D0%BE%D0%BC%D0%B8%D1%8F", n.PermalinkEncoded())

	n2 := &NoteView{Permalink: "/hello_world"}
	require.Equal(t, "/hello_world", n2.PermalinkEncoded())
}

func TestPreparePermalink_AlternatePermalinks(t *testing.T) {
	// When config is russkayalatinica2, cyrillic and simple_translit are alternates.
	// russkayalatinica v1 may or may not differ from v2 — if same, it's excluded.
	n := &NoteView{Path: "Поиск Астрономия.md"}
	n.PreparePermalink(URLNormRusskayaLatinica2)

	require.Equal(t, "/poisk_astronomiya", n.Permalink)
	require.Equal(t, "/поиск_астрономия", n.PermalinkOriginal)

	// Cyrillic variant is same as PermalinkOriginal -> excluded from alternates
	// rl1 == rl2 for this input -> excluded (same as Permalink)
	// Only simple_translit differs
	require.NotNil(t, n.AlternatePermalinks)
	require.Equal(t, "/poisk_astronomija", n.AlternatePermalinks[URLNormSimpleTranslit])
	// cyrillic == PermalinkOriginal, so it should NOT be in alternates
	_, hasCyrillic := n.AlternatePermalinks[URLNormCyrillic]
	require.False(t, hasCyrillic)
}

func TestPreparePermalink_CyrillicMethod(t *testing.T) {
	n := &NoteView{Path: "Моя Страница.md"}
	n.PreparePermalink(URLNormCyrillic)

	// Cyrillic method: Permalink == PermalinkOriginal
	require.Equal(t, "/моя_страница", n.Permalink)
	require.Equal(t, "/моя_страница", n.PermalinkOriginal)

	// Alternates should contain the Latin variants
	require.NotNil(t, n.AlternatePermalinks)
	require.Equal(t, "/moja_stranica", n.AlternatePermalinks[URLNormSimpleTranslit])
}

func TestPreparePermalink_SimpleTranslitMethod(t *testing.T) {
	n := &NoteView{Path: "Моя Страница.md"}
	n.PreparePermalink(URLNormSimpleTranslit)

	require.Equal(t, "/moja_stranica", n.Permalink)
	require.Equal(t, "/моя_страница", n.PermalinkOriginal)

	// Other Latin variants should be in alternates
	require.NotNil(t, n.AlternatePermalinks)
	_, hasRL1 := n.AlternatePermalinks[URLNormRusskayaLatinica]
	_, hasRL2 := n.AlternatePermalinks[URLNormRusskayaLatinica2]
	// rl1 and rl2 produce "/moya_stranica" which differs from "/moja_stranica"
	require.True(t, hasRL1 || hasRL2)
}

func TestPreparePermalink_SlugNoAlternates(t *testing.T) {
	n := &NoteView{Path: "file.md", Slug: "custom-slug"}
	n.PreparePermalink(URLNormRusskayaLatinica2)

	require.Equal(t, "/custom-slug", n.Permalink)
	require.Nil(t, n.AlternatePermalinks, "slug-based notes must have nil AlternatePermalinks")
}

func TestPreparePermalink_CyrillicSlugNoEncoding(t *testing.T) {
	n := &NoteView{Path: "file.md", Slug: "моя-страница"}
	n.PreparePermalink(URLNormRusskayaLatinica2)

	// Permalink must be decoded unicode, not percent-encoded
	require.Equal(t, "/моя-страница", n.Permalink)
	require.Nil(t, n.AlternatePermalinks)
}

func TestRegisterNote_AlternatePermalinksInMap(t *testing.T) {
	nvs := NewNoteViews()
	note := &NoteView{
		Path:              "Поиск.md",
		Permalink:         "/poisk",
		PermalinkOriginal: "/поиск",
		AlternatePermalinks: map[URLNormalizationMethod]string{
			URLNormCyrillic:          "/поиск_cyrillic",
			URLNormSimpleTranslit:    "/poisk_simple",
			URLNormRusskayaLatinica:  "/poisk_rl1",
			URLNormRusskayaLatinica2: "/poisk_rl2",
		},
	}
	nvs.RegisterNote(note)

	// All variants must resolve to the same note
	require.Equal(t, note, nvs.Map["/poisk"])
	require.Equal(t, note, nvs.Map["/поиск"])
	require.Equal(t, note, nvs.Map["/poisk_simple"])
}

func TestRegisterNote_AlternateCollisionLastWriterWins(t *testing.T) {
	nvs := NewNoteViews()

	note1 := &NoteView{
		Path:              "note1.md",
		Permalink:         "/note1",
		PermalinkOriginal: "/note1",
		AlternatePermalinks: map[URLNormalizationMethod]string{
			URLNormCyrillic:          "/note1_cyrillic",
			URLNormSimpleTranslit:    "/collision",
			URLNormRusskayaLatinica:  "/note1_rl1",
			URLNormRusskayaLatinica2: "/note1_rl2",
		},
	}
	nvs.RegisterNote(note1)

	note2 := &NoteView{
		Path:              "note2.md",
		Permalink:         "/note2",
		PermalinkOriginal: "/note2",
		AlternatePermalinks: map[URLNormalizationMethod]string{
			URLNormCyrillic:          "/note2_cyrillic",
			URLNormSimpleTranslit:    "/collision",
			URLNormRusskayaLatinica:  "/note2_rl1",
			URLNormRusskayaLatinica2: "/note2_rl2",
		},
	}
	nvs.RegisterNote(note2)

	// Last-writer-wins for alternate collisions
	require.Equal(t, note2, nvs.Map["/collision"])
	// Original permalinks are untouched
	require.Equal(t, note1, nvs.Map["/note1"])
	require.Equal(t, note2, nvs.Map["/note2"])
}

func TestConfigDrivenMethodChange(t *testing.T) {
	// Simulate changing config from russkayalatinica2 to cyrillic
	n := &NoteView{Path: "Тест.md"}

	n.PreparePermalink(URLNormRusskayaLatinica2)
	require.Equal(t, "/test", n.Permalink)

	// Re-prepare with different method
	n.AlternatePermalinks = nil // reset
	n.PreparePermalink(URLNormCyrillic)
	require.Equal(t, "/тест", n.Permalink)
	require.Equal(t, "/тест", n.PermalinkOriginal)
}
