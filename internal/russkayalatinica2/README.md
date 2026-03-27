# russkayalatinica2

Bidirectional Cyrillic ↔ Latin transliteration using **Russkaya Latinica**.
Ported from [LinguisticKit](https://github.com/nicklama/LinguisticKit) (Swift) `ScriptTable.ru`.

## Mapping table

### Simple (context-free)

| Cyrillic | Latin | Type |
|----------|-------|------|
| а | a | vowel |
| б | b | consonant |
| в | v | consonant |
| г | g | consonant |
| д | d | consonant |
| ж | zh | consonant |
| з | z | consonant |
| и | i | vowel |
| к | k | consonant |
| л | l | consonant |
| м | m | consonant |
| н | n | consonant |
| о | o | vowel |
| п | p | consonant |
| р | r | consonant |
| с | s | consonant |
| т | t | consonant |
| у | u | vowel |
| ф | f | consonant |
| х | kh | consonant |
| ц | c | consonant |
| ч | ch | consonant |
| ш | sh | consonant |
| щ | sjh | consonant |
| я | ya | other |
| ю | yu | other |
| ё | yo | other |
| ы | yi | other |

### Context-dependent

| Cyrillic | Latin | Prefix context | Postfix context |
|----------|-------|---------------|-----------------|
| **э** | e | non-consonant | — |
| **э** | ye | consonant | — |
| **е** | e | consonant | — |
| **е** | ye | non-consonant | — |
| **й** | j | non-consonant | — |
| **й** | yj | consonant | — |
| **ь** | j | consonant | — |
| **ь** | hj | non-consonant | — |
| **ъ** | y | consonant | non-vowel |
| **ъ** | hy | non-consonant | non-vowel |
| **ъ** | yh | consonant | vowel |
| **ъ** | hyh | non-consonant | vowel |

## Element types

Types determine how a character is "seen" by neighboring context rules.
Derived from the first cell for each character (matching Swift behavior).

- **vowel**: а, и, о, у, э, й, ь
- **consonant**: б, в, г, д, ж, з, к, л, м, н, п, р, с, т, ф, х, ц, ч, ш, щ
- **other**: я, ю, ё, ы, е, ъ, ѵ

Key insight: compound/iotated vowels (я, ю, ё, ы, е) are `other`, not `vowel`.
This matters for ъ postfix matching — e.g. `пэнъю`: ю is `other` (non-vowel) → ъ→"y", not "yh".

## Case algorithm

Multi-pass resolution (from Swift `StringProtocol.applyingTransform`):

1. Single uppercase letter → ambiguous
2. **Pass 1**: interior ambiguous — resolve by next neighbor (lowercase→capitalize, other→uppercase)
3. **Pass 2**: consecutive uppercase/ambiguous sequences > 1 → all uppercase
4. **Pass 3**: remaining ambiguous — nearest cased neighbor is uppercase → uppercase, else capitalize

Examples: `КамАЗ`→`KamAZ`, `ПЕТРОВ Ю. Я.`→`PETROV YU. YA.`, `в МХАТе`→`v MKHATe`

## API

```go
russkayalatinica2.Translit("Русская Латиница")    // "Russkaya Latinica"
russkayalatinica2.RevertTranslit("Russkaya Latinica") // "Русская Латиница"
```
