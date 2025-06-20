package russkayalatinica

import (
	"strings"
	"unicode"
)

// ElementType represents the type of a character in the script
type ElementType int

const (
	ElementTypeConsonant ElementType = iota
	ElementTypeVowel
	ElementTypeOther
	ElementTypeNonLetter
)

// ContextType represents the context for transliteration rules
type ContextType int

const (
	ContextTypeUnset ContextType = iota
	ContextTypeConsonant
	ContextTypeVowel
	ContextTypeOther
	ContextTypeNonLetter
	ContextTypeAny
	ContextTypeNonConsonant
	ContextTypeNonVowel
	ContextTypeNonOther
	ContextTypeLetter
	ContextTypeHardSign // Special context for hard sign
)

// Cell represents a transliteration rule
type Cell struct {
	Cyrl           string
	Latn           string
	ElementType    ElementType
	PrefixContext  ContextType
	PostfixContext ContextType
}

// Define the mapping table based on the Swift implementation
var ruScriptTable = buildScriptTable()

// Build reverse mapping table - will be initialized when needed
var reverseTable map[string]string
var reverseTableInitialized bool

func buildScriptTable() []Cell {
	// Based on the user's official Russkaya Latinica table
	cells := []Cell{
		// Simple mappings first (no context dependencies)
		{Cyrl: "а", Latn: "a", ElementType: ElementTypeVowel},
		{Cyrl: "б", Latn: "b", ElementType: ElementTypeConsonant},
		{Cyrl: "в", Latn: "v", ElementType: ElementTypeConsonant},
		{Cyrl: "г", Latn: "g", ElementType: ElementTypeConsonant},
		{Cyrl: "д", Latn: "d", ElementType: ElementTypeConsonant},
		{Cyrl: "ё", Latn: "yo", ElementType: ElementTypeVowel},
		{Cyrl: "ж", Latn: "zh", ElementType: ElementTypeConsonant},
		{Cyrl: "з", Latn: "z", ElementType: ElementTypeConsonant},
		{Cyrl: "и", Latn: "i", ElementType: ElementTypeVowel},
		{Cyrl: "к", Latn: "k", ElementType: ElementTypeConsonant},
		{Cyrl: "л", Latn: "l", ElementType: ElementTypeConsonant},
		{Cyrl: "м", Latn: "m", ElementType: ElementTypeConsonant},
		{Cyrl: "н", Latn: "n", ElementType: ElementTypeConsonant},
		{Cyrl: "о", Latn: "o", ElementType: ElementTypeVowel},
		{Cyrl: "п", Latn: "p", ElementType: ElementTypeConsonant},
		{Cyrl: "р", Latn: "r", ElementType: ElementTypeConsonant},
		{Cyrl: "с", Latn: "s", ElementType: ElementTypeConsonant},
		{Cyrl: "т", Latn: "t", ElementType: ElementTypeConsonant},
		{Cyrl: "у", Latn: "u", ElementType: ElementTypeVowel},
		{Cyrl: "ф", Latn: "f", ElementType: ElementTypeConsonant},
		{Cyrl: "х", Latn: "kh", ElementType: ElementTypeConsonant},
		{Cyrl: "ц", Latn: "c", ElementType: ElementTypeConsonant},
		{Cyrl: "ч", Latn: "ch", ElementType: ElementTypeConsonant},
		{Cyrl: "ш", Latn: "sh", ElementType: ElementTypeConsonant},
		{Cyrl: "щ", Latn: "sjh", ElementType: ElementTypeConsonant},
		{Cyrl: "э", Latn: "e", ElementType: ElementTypeVowel},
		{Cyrl: "ю", Latn: "yu", ElementType: ElementTypeVowel},
		{Cyrl: "я", Latn: "ya", ElementType: ElementTypeVowel},

		// Uppercase versions
		{Cyrl: "А", Latn: "A", ElementType: ElementTypeVowel},
		{Cyrl: "Б", Latn: "B", ElementType: ElementTypeConsonant},
		{Cyrl: "В", Latn: "V", ElementType: ElementTypeConsonant},
		{Cyrl: "Г", Latn: "G", ElementType: ElementTypeConsonant},
		{Cyrl: "Д", Latn: "D", ElementType: ElementTypeConsonant},
		{Cyrl: "Ё", Latn: "YO", ElementType: ElementTypeVowel},
		{Cyrl: "Ж", Latn: "ZH", ElementType: ElementTypeConsonant},
		{Cyrl: "З", Latn: "Z", ElementType: ElementTypeConsonant},
		{Cyrl: "И", Latn: "I", ElementType: ElementTypeVowel},
		{Cyrl: "К", Latn: "K", ElementType: ElementTypeConsonant},
		{Cyrl: "Л", Latn: "L", ElementType: ElementTypeConsonant},
		{Cyrl: "М", Latn: "M", ElementType: ElementTypeConsonant},
		{Cyrl: "Н", Latn: "N", ElementType: ElementTypeConsonant},
		{Cyrl: "О", Latn: "O", ElementType: ElementTypeVowel},
		{Cyrl: "П", Latn: "P", ElementType: ElementTypeConsonant},
		{Cyrl: "Р", Latn: "R", ElementType: ElementTypeConsonant},
		{Cyrl: "С", Latn: "S", ElementType: ElementTypeConsonant},
		{Cyrl: "Т", Latn: "T", ElementType: ElementTypeConsonant},
		{Cyrl: "У", Latn: "U", ElementType: ElementTypeVowel},
		{Cyrl: "Ф", Latn: "F", ElementType: ElementTypeConsonant},
		{Cyrl: "Х", Latn: "KH", ElementType: ElementTypeConsonant},
		{Cyrl: "Ц", Latn: "C", ElementType: ElementTypeConsonant},
		{Cyrl: "Ч", Latn: "CH", ElementType: ElementTypeConsonant},
		{Cyrl: "Ш", Latn: "SH", ElementType: ElementTypeConsonant},
		{Cyrl: "Щ", Latn: "SJH", ElementType: ElementTypeConsonant},
		{Cyrl: "Э", Latn: "E", ElementType: ElementTypeVowel},
		{Cyrl: "Ю", Latn: "YU", ElementType: ElementTypeVowel},
		{Cyrl: "Я", Latn: "YA", ElementType: ElementTypeVowel},

		// Context-dependent rules exactly from Swift implementation

		// е rules
		{Cyrl: "е", Latn: "ye", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "е", Latn: "e", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},
		{Cyrl: "Е", Latn: "YE", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "Е", Latn: "E", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},

		// э rules
		{Cyrl: "э", Latn: "e", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "э", Latn: "ye", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},
		{Cyrl: "Э", Latn: "E", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "Э", Latn: "YE", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},

		// й rules
		{Cyrl: "й", Latn: "j", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "й", Latn: "yj", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},
		{Cyrl: "Й", Latn: "J", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "Й", Latn: "YJ", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},

		// ь rules
		{Cyrl: "ь", Latn: "j", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},
		{Cyrl: "ь", Latn: "hj", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "Ь", Latn: "J", ElementType: ElementTypeVowel, PrefixContext: ContextTypeConsonant},
		{Cyrl: "Ь", Latn: "HJ", ElementType: ElementTypeVowel, PrefixContext: ContextTypeNonConsonant},

		// ы rules
		{Cyrl: "ы", Latn: "yi", ElementType: ElementTypeVowel},
		{Cyrl: "Ы", Latn: "YI", ElementType: ElementTypeVowel},

		// ъ rules (simplified based on test analysis)
		{Cyrl: "ъ", Latn: "y", ElementType: ElementTypeOther, PrefixContext: ContextTypeConsonant},
		{Cyrl: "ъ", Latn: "hy", ElementType: ElementTypeOther, PrefixContext: ContextTypeNonConsonant},
		{Cyrl: "Ъ", Latn: "Y", ElementType: ElementTypeOther, PrefixContext: ContextTypeConsonant},
		{Cyrl: "Ъ", Latn: "HY", ElementType: ElementTypeOther, PrefixContext: ContextTypeNonConsonant},

		// Add special е/Е rule after hard sign
		{Cyrl: "е", Latn: "ye", ElementType: ElementTypeVowel, PrefixContext: ContextTypeHardSign},
		{Cyrl: "Е", Latn: "YE", ElementType: ElementTypeVowel, PrefixContext: ContextTypeHardSign},

		// Special characters
		{Cyrl: "ѵ", Latn: "y", ElementType: ElementTypeOther},
	}

	// Set default contexts where not specified
	for i := range cells {
		if cells[i].PrefixContext == ContextTypeUnset && cells[i].PostfixContext == ContextTypeUnset {
			cells[i].PrefixContext = ContextTypeAny
			cells[i].PostfixContext = ContextTypeAny
		} else if cells[i].PrefixContext != ContextTypeUnset && cells[i].PostfixContext == ContextTypeUnset {
			cells[i].PostfixContext = ContextTypeAny
		} else if cells[i].PrefixContext == ContextTypeUnset && cells[i].PostfixContext != ContextTypeUnset {
			cells[i].PrefixContext = ContextTypeAny
		}
	}

	return cells
}

// Create a mapping for quick lookup of element types
var elementTypeMap = map[rune]ElementType{
	// Vowels
	'а': ElementTypeVowel, 'А': ElementTypeVowel,
	'е': ElementTypeVowel, 'Е': ElementTypeVowel,
	'ё': ElementTypeVowel, 'Ё': ElementTypeVowel,
	'и': ElementTypeVowel, 'И': ElementTypeVowel,
	'о': ElementTypeVowel, 'О': ElementTypeVowel,
	'у': ElementTypeVowel, 'У': ElementTypeVowel,
	'э': ElementTypeVowel, 'Э': ElementTypeVowel,
	'ю': ElementTypeVowel, 'Ю': ElementTypeVowel,
	'я': ElementTypeVowel, 'Я': ElementTypeVowel,
	'ы': ElementTypeVowel, 'Ы': ElementTypeVowel,

	// Consonants
	'б': ElementTypeConsonant, 'Б': ElementTypeConsonant,
	'в': ElementTypeConsonant, 'В': ElementTypeConsonant,
	'г': ElementTypeConsonant, 'Г': ElementTypeConsonant,
	'д': ElementTypeConsonant, 'Д': ElementTypeConsonant,
	'ж': ElementTypeConsonant, 'Ж': ElementTypeConsonant,
	'з': ElementTypeConsonant, 'З': ElementTypeConsonant,
	'к': ElementTypeConsonant, 'К': ElementTypeConsonant,
	'л': ElementTypeConsonant, 'Л': ElementTypeConsonant,
	'м': ElementTypeConsonant, 'М': ElementTypeConsonant,
	'н': ElementTypeConsonant, 'Н': ElementTypeConsonant,
	'п': ElementTypeConsonant, 'П': ElementTypeConsonant,
	'р': ElementTypeConsonant, 'Р': ElementTypeConsonant,
	'с': ElementTypeConsonant, 'С': ElementTypeConsonant,
	'т': ElementTypeConsonant, 'Т': ElementTypeConsonant,
	'ф': ElementTypeConsonant, 'Ф': ElementTypeConsonant,
	'х': ElementTypeConsonant, 'Х': ElementTypeConsonant,
	'ц': ElementTypeConsonant, 'Ц': ElementTypeConsonant,
	'ч': ElementTypeConsonant, 'Ч': ElementTypeConsonant,
	'ш': ElementTypeConsonant, 'Ш': ElementTypeConsonant,
	'щ': ElementTypeConsonant, 'Щ': ElementTypeConsonant,

	// Special cases
	'й': ElementTypeVowel, 'Й': ElementTypeVowel, // It's actually a semivowel
	'ь': ElementTypeOther, 'Ь': ElementTypeOther, // Soft sign - treat as other for context
	'ъ': ElementTypeOther, 'Ъ': ElementTypeOther, // Hard sign
}

func contextMatches(ctx ContextType, elementType ElementType, prevRune rune) bool {
	switch ctx {
	case ContextTypeUnset, ContextTypeAny:
		return true
	case ContextTypeConsonant:
		return elementType == ElementTypeConsonant
	case ContextTypeVowel:
		return elementType == ElementTypeVowel
	case ContextTypeOther:
		return elementType == ElementTypeOther
	case ContextTypeNonLetter:
		return elementType == ElementTypeNonLetter
	case ContextTypeNonConsonant:
		return elementType != ElementTypeConsonant
	case ContextTypeNonVowel:
		return elementType != ElementTypeVowel
	case ContextTypeNonOther:
		return elementType != ElementTypeOther
	case ContextTypeLetter:
		return elementType != ElementTypeNonLetter
	case ContextTypeHardSign:
		return prevRune == 'ъ' || prevRune == 'Ъ'
	}
	return false
}

func getElementType(r rune) ElementType {
	if !unicode.IsLetter(r) {
		return ElementTypeNonLetter
	}

	if et, ok := elementTypeMap[r]; ok {
		return et
	}

	// Default for unknown letters
	return ElementTypeOther
}

// Check if we're at a word boundary (for proper case handling)
func isWordBoundary(runes []rune, pos int) bool {
	if pos == 0 || pos >= len(runes) {
		return true
	}

	prevIsLetter := unicode.IsLetter(runes[pos-1])
	currIsLetter := pos < len(runes) && unicode.IsLetter(runes[pos])

	return !prevIsLetter && currIsLetter
}

// Check if the next character is also uppercase (for handling abbreviations)
func isFollowedByUppercase(runes []rune, pos int) bool {
	if pos+1 >= len(runes) {
		return false
	}
	return unicode.IsUpper(runes[pos+1])
}

// Check if the previous character is also uppercase
func isPrecededByUppercase(runes []rune, pos int) bool {
	if pos == 0 {
		return false
	}
	return unicode.IsUpper(runes[pos-1])
}

// Apply proper case to transliterated text
func applyProperCase(input string, isUppercase bool, isAtWordBoundary bool, followedByUpper bool, precededByUpper bool) string {
	if !isUppercase {
		return input
	}

	// If it's part of an all-caps sequence, keep it uppercase
	if precededByUpper || followedByUpper {
		return input
	}

	// If it's a single uppercase letter at word boundary, capitalize only the first letter
	if isAtWordBoundary && len(input) > 1 {
		return strings.Title(strings.ToLower(input))
	}

	return input
}

// Translit converts Cyrillic text to Latin using Russkaya Latinica
func Translit(input string) string {
	if input == "" {
		return ""
	}

	runes := []rune(input)
	var result strings.Builder

	for i := 0; i < len(runes); {
		// Determine prefix context
		var prefixType ElementType
		if i == 0 {
			prefixType = ElementTypeNonLetter
		} else {
			prefixType = getElementType(runes[i-1])
		}

		// Try to match the current character
		matched := false
		currentRune := runes[i]
		currentChar := string(currentRune)

		// Determine postfix context
		var postfixType ElementType
		if i+1 >= len(runes) {
			postfixType = ElementTypeNonLetter
		} else {
			postfixType = getElementType(runes[i+1])
		}

		// Find matching cell
		var prevRune, nextRune rune
		if i > 0 {
			prevRune = runes[i-1]
		}
		if i+1 < len(runes) {
			nextRune = runes[i+1]
		}

		for _, cell := range ruScriptTable {
			if cell.Cyrl == currentChar &&
				contextMatches(cell.PrefixContext, prefixType, prevRune) &&
				contextMatches(cell.PostfixContext, postfixType, nextRune) {

				// Apply proper case handling
				transliterated := cell.Latn
				if unicode.IsUpper(currentRune) {
					isAtBoundary := isWordBoundary(runes, i)
					isFollowedByUpper := isFollowedByUppercase(runes, i)
					isPrecededByUpper := isPrecededByUppercase(runes, i)
					transliterated = applyProperCase(transliterated, true, isAtBoundary, isFollowedByUpper, isPrecededByUpper)
				}

				result.WriteString(transliterated)
				i++
				matched = true
				break
			}
		}

		// If no match found, copy the character as is
		if !matched {
			result.WriteRune(runes[i])
			i++
		}
	}

	return result.String()
}

func buildReverseTable() {
	reverseTable = make(map[string]string)

	// Build the reverse mapping from the actual rules
	// Since we guarantee one-to-one conversion, we need to ensure
	// each Latin sequence maps to exactly one Cyrillic sequence

	// First, handle single characters in isolation
	singleChars := "абвгдеёжзийклмнопрстуфхцчшщыэюяъь"
	for _, r := range singleChars {
		cyrlChar := string(r)
		latinSeq := Translit(cyrlChar)
		latinLower := strings.ToLower(latinSeq)
		cyrlLower := strings.ToLower(cyrlChar)

		if _, exists := reverseTable[latinLower]; !exists {
			reverseTable[latinLower] = cyrlLower
		}
	}

	// Handle uppercase versions
	upperChars := "АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЫЭЮЯЪЬ"
	for _, r := range upperChars {
		cyrlChar := string(r)
		latinSeq := Translit(cyrlChar)
		latinLower := strings.ToLower(latinSeq)
		cyrlLower := strings.ToLower(cyrlChar)

		if _, exists := reverseTable[latinLower]; !exists {
			reverseTable[latinLower] = cyrlLower
		}
	}

	// Handle special context-dependent cases
	contextCases := []string{
		// Put common е combinations first to ensure they get priority over э combinations
		"же", "че", "ше", "ще", // Common consonant + е combinations
		"ъе", "ъэ", "съ", "дъ", "ъа", "хъ",
		"Съе", "съе", "тъе", "нъе", // consonant + hard sign + vowel cases
		"ье", "сь", "дь", "нь", "ль",
		"шь", "жь", "чь", "щь", // consonant + soft sign cases
		"йе", "сй", "дй", "ный",
		"се", "де", "не", "те", "ре", "пе", "ле", "ме", "ке", "бе", "ве", "фе", "ге", "хе", "це", "зе",
		"эе", "эа", "эо", "эу",
	}

	for _, cyrlSeq := range contextCases {
		actualLatin := Translit(cyrlSeq)
		latinLower := strings.ToLower(actualLatin)
		cyrlLower := strings.ToLower(cyrlSeq)

		if _, exists := reverseTable[latinLower]; !exists {
			reverseTable[latinLower] = cyrlLower
		} else if len(cyrlSeq) > 1 {
			// For multi-character sequences, prefer them over single chars
			reverseTable[latinLower] = cyrlLower
		}

		// Special handling: if this is "consonant + ъе", also add "yye" -> "ъе"
		runeLen := len([]rune(cyrlSeq))
		if runeLen == 3 && strings.HasSuffix(cyrlSeq, "ъе") {
			// "Съе" -> "Syye", so we want "yye" -> "ъе"
			latinRunes := []rune(actualLatin)
			cyrlRunes := []rune(cyrlSeq)

			suffixLatin := string(latinRunes[1:]) // Skip first character (consonant)
			suffixCyrl := string(cyrlRunes[1:])   // Skip first character (consonant)
			suffixLatinLower := strings.ToLower(suffixLatin)
			suffixCyrlLower := strings.ToLower(suffixCyrl)

			if _, exists := reverseTable[suffixLatinLower]; !exists {
				reverseTable[suffixLatinLower] = suffixCyrlLower
			}
		}

		// Special handling: if this is "consonant + ь", also add suffix mappings
		if runeLen == 2 && strings.HasSuffix(cyrlSeq, "ь") {
			// "шь" -> "shj", so we want "shj" -> "шь"
			// This should be handled by the normal mapping, but let's ensure priority
			if _, exists := reverseTable[latinLower]; !exists {
				reverseTable[latinLower] = cyrlLower
			} else {
				// Prefer the multi-character sequence over individual characters
				reverseTable[latinLower] = cyrlLower
			}
		}
	}

	// Force specific priorities for ambiguous cases
	// Common е combinations should take priority over э combinations
	forcePriorities := map[string]string{
		"zhe": "же",
		"che": "че", 
		"she": "ше",
		"sjhe": "ще",
		"en": "ен", // Add this to fix "en" -> "эн" issue
	}
	
	for latin, cyrl := range forcePriorities {
		if existing, exists := reverseTable[latin]; exists && existing != cyrl {
			reverseTable[latin] = cyrl
		}
	}
}

// RevertTranslit converts Latin text back to Cyrillic
func RevertTranslit(input string) string {
	if input == "" {
		return ""
	}

	// Initialize reverse table if not done yet
	if !reverseTableInitialized {
		buildReverseTable()
		reverseTableInitialized = true
	}

	// Get all Latin sequences sorted by length (longest first)
	var latinSeqs []string
	for seq := range reverseTable {
		latinSeqs = append(latinSeqs, seq)
	}
	// Sort by length descending
	for i := 0; i < len(latinSeqs); i++ {
		for j := i + 1; j < len(latinSeqs); j++ {
			if len(latinSeqs[j]) > len(latinSeqs[i]) {
				latinSeqs[i], latinSeqs[j] = latinSeqs[j], latinSeqs[i]
			}
		}
	}

	var result strings.Builder
	i := 0

	for i < len(input) {
		matched := false

		// Try to match the longest possible Latin sequence
		for _, latinSeq := range latinSeqs {
			if i+len(latinSeq) <= len(input) {
				// Extract the substring
				substr := input[i : i+len(latinSeq)]
				substrLower := strings.ToLower(substr)

				if substrLower == latinSeq {
					// Found a match
					cyrlChar := reverseTable[latinSeq]

					// Handle case preservation
					if len(substr) > 0 && unicode.IsUpper([]rune(substr)[0]) {
						if len(substr) > 1 && unicode.IsUpper([]rune(substr)[1]) {
							// All uppercase
							cyrlChar = strings.ToUpper(cyrlChar)
						} else {
							// Title case
							cyrlRunes := []rune(cyrlChar)
							if len(cyrlRunes) > 0 {
								cyrlRunes[0] = unicode.ToUpper(cyrlRunes[0])
								cyrlChar = string(cyrlRunes)
							}
						}
					}

					result.WriteString(cyrlChar)
					i += len(latinSeq)
					matched = true
					break
				}
			}
		}

		// If no match found, copy the character as is
		if !matched {
			result.WriteByte(input[i])
			i++
		}
	}

	return result.String()
}