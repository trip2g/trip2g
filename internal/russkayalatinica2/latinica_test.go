package russkayalatinica2

import (
	"fmt"
	"testing"
)

// All test cases from LinguisticKit ScriptTableRuTests.swift.
var swiftTests = []struct {
	cyrl string
	latn string
}{
	// Pangrams
	{"Съешь же ещё этих мягких французских булок, да выпей чаю.", "Syyeshj zhe yesjhyo etikh myagkikh francuzskikh bulok, da vyipej chayu."},
	{"СЪЕШЬ ЖЕ ЕЩЁ ЭТИХ МЯГКИХ ФРАНЦУЗСКИХ БУЛОК, ДА ВЫПЕЙ ЧАЮ.", "SYYESHJ ZHE YESJHYO ETIKH MYAGKIKH FRANCUZSKIKH BULOK, DA VYIPEJ CHAYU."},

	// Hard sign sequences
	{"ъэ", "hyhe"},
	{"дъэ", "dyhe"},
	{"ъй", "hyhj"},
	{"дъй", "dyhj"},
	{"дъь", "dyhhj"},
	{"ъъ", "hyhy"},
	{"пэнъю", "pyenyyu"},

	// Words
	{"интервьюер", "intervjyuyer"},
	{"Забжэ", "Zabzhye"},
	{"Чанъань", "Chanyhanj"},
	{"секвойе", "sekvojye"},
	{"Йемен", "Jyemen"},
	{"трансйеменский", "transyjyemenskij"},
	{"безйодовый", "bezyjodovyij"},
	{"тэны", "tyenyi"},

	// Acronyms
	{"МКС", "MKS"},
	{"КС", "KS"},
	{"АКС", "AKS"},
	{"ЮСЭКС", "YUSYEKS"},
	{"ЮНЕСКО", "YUNESKO"},
	{"ТЭН", "TYEN"},
	{"ЖКХ", "ZHKKH"},
	{"ЕГЭ", "YEGYE"},
	{"ЭВМ", "EVM"},
	{"ИРЯ", "IRYA"},
	{"ФИДЕ", "FIDE"},
	{"ЦЕРН", "CERN"},
	{"ТЮЗ", "TYUZ"},
	{"МАГАТЭ", "MAGATYE"},
	{"ОБСЕ", "OBSE"},
	{"ТЭС", "TYES"},
	{"ОБЖ", "OBZH"},

	// Mixed case
	{"КамАЗ", "KamAZ"},
	{"в МХАТе", "v MKHATe"},

	// Initials
	{"Петров Ю. Я.", "Petrov Yu. Ya."},
	{"Ю. Я. Петров", "Yu. Ya. Petrov"},
	{"ПЕТРОВ Ю. Я.", "PETROV YU. YA."},
	{"Ю. Я. ПЕТРОВ", "YU. YA. PETROV"},

	// Long mixed
	{"ОБЖ — основы безопасности жизнедеятельности", "OBZH — osnovyi bezopasnosti zhiznedeyateljnosti"},

	// Standalone special chars
	{"ь ъ ѵ", "hj hy y"},

	// Simple
	{"Русская Латиница", "Russkaya Latinica"},
}

func TestTranslit(t *testing.T) {
	for _, tc := range swiftTests {
		got := Translit(tc.cyrl)
		if got != tc.latn {
			t.Errorf("Translit(%q)\n  got  %q\n  want %q", tc.cyrl, got, tc.latn)
		}
	}
}

func TestRevertTranslit(t *testing.T) {
	for _, tc := range swiftTests {
		got := RevertTranslit(tc.latn)
		if got != tc.cyrl {
			t.Errorf("RevertTranslit(%q)\n  got  %q\n  want %q", tc.latn, got, tc.cyrl)
		}
	}
}

func TestSingleCharRoundTrip(t *testing.T) {
	chars := "абвгдеёжзийклмнопрстуфхцчшщъьыэюя"
	for _, r := range chars {
		s := string(r)
		latin := Translit(s)
		back := RevertTranslit(latin)
		if back != s {
			t.Errorf("round-trip failed: %s → %s → %s", s, latin, back)
		}
	}
}

func TestFullRoundTrip(t *testing.T) {
	for _, tc := range swiftTests {
		latin := Translit(tc.cyrl)
		back := RevertTranslit(latin)
		if back != tc.cyrl {
			t.Errorf("round-trip failed for %q:\n  → %q\n  → %q", tc.cyrl, latin, back)
		}
	}
}

func TestPassthrough(t *testing.T) {
	// Non-Cyrillic text should pass through unchanged.
	inputs := []string{
		"Hello, World!",
		"12345",
		"",
		"café résumé",
		"日本語",
	}
	for _, s := range inputs {
		if got := Translit(s); got != s {
			t.Errorf("Translit(%q) = %q, want passthrough", s, got)
		}
	}
}

func BenchmarkTranslit(b *testing.B) {
	text := "Съешь же ещё этих мягких французских булок, да выпей чаю."
	for b.Loop() {
		Translit(text)
	}
}

func BenchmarkRevertTranslit(b *testing.B) {
	text := "Syyeshj zhe yesjhyo etikh myagkikh francuzskikh bulok, da vyipej chayu."
	for b.Loop() {
		RevertTranslit(text)
	}
}

func ExampleTranslit() {
	fmt.Println(Translit("Русская Латиница"))
	// Output: Russkaya Latinica
}

func ExampleRevertTranslit() {
	fmt.Println(RevertTranslit("Russkaya Latinica"))
	// Output: Русская Латиница
}
