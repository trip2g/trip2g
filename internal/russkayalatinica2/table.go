package russkayalatinica2

// cell defines a bidirectional transliteration rule.
// All characters/strings are lowercase; case is resolved separately.
//
// The typ field determines how this character is seen by neighboring cells
// for context matching. It is derived from the FIRST cell for each character
// (matching Swift's allScriptTable[element]?.first?.type behavior).
type cell struct {
	cyrl    rune        // lowercase Cyrillic character
	latn    string      // lowercase Latin equivalent
	typ     elementType // element type for context matching
	prefix  contextType // required prefix context (zero = any)
	postfix contextType // required postfix context (zero = any)
}

// ruTable defines transliteration rules for Russkaya Latinica.
// Faithfully ported from LinguisticKit ScriptTable.ru.swift.
//
// Ordering contract: for characters with multiple rules, the FIRST cell
// listed determines the element type used for context matching by neighbors.
// Context rules are mutually exclusive so matching order within a character
// does not affect correctness.
//
//nolint:gochecknoglobals
var ruTable = [...]cell{
	// Simple vowels (type: vowel)
	{cyrl: 'а', latn: "a", typ: vowel},
	{cyrl: 'и', latn: "i", typ: vowel},
	{cyrl: 'о', latn: "o", typ: vowel},
	{cyrl: 'у', latn: "u", typ: vowel},

	// Consonants
	{cyrl: 'б', latn: "b", typ: consonant},
	{cyrl: 'в', latn: "v", typ: consonant},
	{cyrl: 'г', latn: "g", typ: consonant},
	{cyrl: 'д', latn: "d", typ: consonant},
	{cyrl: 'ж', latn: "zh", typ: consonant},
	{cyrl: 'з', latn: "z", typ: consonant},
	{cyrl: 'к', latn: "k", typ: consonant},
	{cyrl: 'л', latn: "l", typ: consonant},
	{cyrl: 'м', latn: "m", typ: consonant},
	{cyrl: 'н', latn: "n", typ: consonant},
	{cyrl: 'п', latn: "p", typ: consonant},
	{cyrl: 'р', latn: "r", typ: consonant},
	{cyrl: 'с', latn: "s", typ: consonant},
	{cyrl: 'т', latn: "t", typ: consonant},
	{cyrl: 'ф', latn: "f", typ: consonant},
	{cyrl: 'х', latn: "kh", typ: consonant},
	{cyrl: 'ц', latn: "c", typ: consonant},
	{cyrl: 'ч', latn: "ch", typ: consonant},
	{cyrl: 'ш', latn: "sh", typ: consonant},
	{cyrl: 'щ', latn: "sjh", typ: consonant},

	// Compound/iotated vowels (type: other per Swift)
	// These are NOT typed as vowel because in the Swift ru table
	// their cells use default type (.other). This matters for
	// postfixContext matching on ъ (nonVowel vs vowel).
	{cyrl: 'я', latn: "ya", typ: other},
	{cyrl: 'ю', latn: "yu", typ: other},
	{cyrl: 'ё', latn: "yo", typ: other},
	{cyrl: 'ы', latn: "yi", typ: other},

	// Context-dependent: э
	// First cell → element type: vowel.
	// After non-consonant: "e". After consonant: "ye".
	{cyrl: 'э', latn: "e", typ: vowel, prefix: ctxNonConsonant},
	{cyrl: 'э', latn: "ye", typ: other, prefix: ctxConsonant},

	// Context-dependent: е
	// First cell → element type: other (NOT vowel — matches Swift).
	// After consonant: "e". After non-consonant: "ye".
	{cyrl: 'е', latn: "e", typ: other, prefix: ctxConsonant},
	{cyrl: 'е', latn: "ye", typ: other, prefix: ctxNonConsonant},

	// Context-dependent: й
	// First cell → element type: vowel.
	{cyrl: 'й', latn: "j", typ: vowel, prefix: ctxNonConsonant},
	{cyrl: 'й', latn: "yj", typ: other, prefix: ctxConsonant},

	// Context-dependent: ь (soft sign)
	// First cell → element type: vowel (matches Swift).
	{cyrl: 'ь', latn: "j", typ: vowel, prefix: ctxConsonant},
	{cyrl: 'ь', latn: "hj", typ: vowel, prefix: ctxNonConsonant},

	// Context-dependent: ъ (hard sign)
	// 4 rules based on prefix (consonant/non) × postfix (vowel/non).
	// First cell → element type: other.
	{cyrl: 'ъ', latn: "y", typ: other, prefix: ctxConsonant, postfix: ctxNonVowel},
	{cyrl: 'ъ', latn: "hy", typ: other, prefix: ctxNonConsonant, postfix: ctxNonVowel},
	{cyrl: 'ъ', latn: "yh", typ: other, prefix: ctxConsonant, postfix: ctxVowel},
	{cyrl: 'ъ', latn: "hyh", typ: other, prefix: ctxNonConsonant, postfix: ctxVowel},

	// Historical
	{cyrl: 'ѵ', latn: "y", typ: other},
}
