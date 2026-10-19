// Package frontmatteredit sets and removes top-level keys of a note's YAML
// frontmatter by splicing text: the lines of every key it does not touch, the
// comments between them and the note body come out byte for byte as they went in.
package frontmatteredit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

var (
	ErrNothingToChange = errors.New("set or unset must name at least one key")
	ErrEmptyKey        = errors.New("frontmatter key must not be empty")
	ErrControlInKey    = errors.New("frontmatter key must not contain control characters or line breaks")
	ErrSetAndUnset     = errors.New("a key cannot be both set and unset")
	ErrUnclosed        = errors.New("frontmatter has no closing --- line")
	ErrInvalidYAML     = errors.New("frontmatter is not valid YAML")
	ErrNotMapping      = errors.New("frontmatter is not a key: value mapping")
	ErrFlowMapping     = errors.New("frontmatter written as {...} cannot be edited key by key")
	ErrIndentedMapping = errors.New("frontmatter keys are indented; move them to the start of the line to edit them")
	ErrAlias           = errors.New("frontmatter uses YAML aliases (*name); edit it as a whole note instead")
	ErrDuplicateKey    = errors.New("frontmatter declares the key more than once")
	ErrUnsafeEdit      = errors.New("frontmatter edit would not read back as intended; nothing was changed")
)

const bom = "\uFEFF"

// Apply returns content with each key of set added or replaced and each key of
// unset removed. Only top-level keys are addressed: "a.b" names a key literally
// called "a.b". Keys new to the note are appended after its last key in sorted
// order. A note without frontmatter gets a block when set is not empty.
//
// The result is parsed back before it is returned; when any key other than the
// ones named would read differently, Apply returns ErrUnsafeEdit and no content.
func Apply(content string, set map[string]any, unset []string) (string, error) {
	err := Validate(set, unset)
	if err != nil {
		return "", err
	}

	prefix := ""
	if strings.HasPrefix(content, bom) {
		prefix = bom
		content = content[len(bom):]
	}

	edited, err := edit(content, set, unset)
	if err != nil {
		return "", err
	}

	err = verify(content, edited, set, unset)
	if err != nil {
		return "", err
	}
	return prefix + edited, nil
}

// Validate checks set and unset alone, before any note is read: at least one
// key, no blank key, no control character or line break in a key, no key named
// in both.
func Validate(set map[string]any, unset []string) error {
	if len(set) == 0 && len(unset) == 0 {
		return ErrNothingToChange
	}
	for key := range set {
		err := validateKey(key)
		if err != nil {
			return err
		}
		if slices.Contains(unset, key) {
			return fmt.Errorf("%w: %s", ErrSetAndUnset, key)
		}
	}
	for _, key := range unset {
		err := validateKey(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return ErrEmptyKey
	}
	if strings.ContainsFunc(key, isControl) {
		return fmt.Errorf("%w: %q", ErrControlInKey, key)
	}
	return nil
}

func isControl(r rune) bool {
	return unicode.IsControl(r) || r == '\u2028' || r == '\u2029'
}

// frontmatter is a note cut where goldmark-meta cuts it. lines is the YAML
// document, split on every line break YAML counts; tail runs from a "..."
// document end marker, if any, through the closing fence.
type frontmatter struct {
	fence string
	lines []string
	tail  string
	body  string
}

func (f frontmatter) String() string {
	return f.fence + strings.Join(f.lines, "") + f.tail + f.body
}

func (f frontmatter) yaml() string {
	return strings.Join(f.lines, "")
}

func split(content string) (frontmatter, error) {
	lines := strings.SplitAfter(content, "\n")
	if !isFence(lines[0]) {
		return frontmatter{body: content}, nil
	}

	closing := slices.IndexFunc(lines[1:], isFence)
	if closing == -1 {
		return frontmatter{}, ErrUnclosed
	}

	yamlLines := splitYAMLLines(strings.Join(lines[1:closing+1], ""))
	end := slices.IndexFunc(yamlLines, isDocumentEnd)
	if end == -1 {
		end = len(yamlLines)
	}

	f := frontmatter{
		fence: lines[0],
		lines: yamlLines[:end],
		tail:  strings.Join(yamlLines[end:], "") + lines[closing+1],
		body:  strings.Join(lines[closing+2:], ""),
	}
	return f, nil
}

// isFence mirrors goldmark-meta, which decides what a note's meta is: a line of
// dashes only, surrounded by optional whitespace.
func isFence(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && strings.Trim(trimmed, "-") == ""
}

// isDocumentEnd reports a "..." marker: both YAML parsers stop reading there.
func isDocumentEnd(line string) bool {
	rest, ok := strings.CutPrefix(line, "...")
	return ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t' || breakLen(rest) > 0)
}

// breakLen is the length of the line break s starts with, 0 when it starts with
// none: the breaks yaml.v3 counts when it numbers lines, CRLF as one.
func breakLen(s string) int {
	for _, b := range [...]string{"\r\n", "\r", "\n", "\u0085", "\u2028", "\u2029"} {
		if strings.HasPrefix(s, b) {
			return len(b)
		}
	}
	return 0
}

// splitYAMLLines splits after each line break, so line n here is the line
// yaml.v3 reports as n+1.
func splitYAMLLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		size := breakLen(s[i:])
		if size == 0 {
			i++
			continue
		}
		i += size
		lines = append(lines, s[start:i])
		start = i
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func lineEnding(line string) string {
	if strings.HasSuffix(line, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func edit(content string, set map[string]any, unset []string) (string, error) {
	f, err := split(content)
	if err != nil {
		return "", err
	}

	if f.fence == "" {
		if len(set) == 0 {
			return content, nil
		}
		nl := lineEnding(strings.SplitAfter(content, "\n")[0])
		added, addErr := encodeNewKeys(set, nil, nl)
		if addErr != nil {
			return "", addErr
		}
		f.fence = "---" + nl
		f.lines = []string{added}
		f.tail = "---" + nl
		return f.String(), nil
	}

	edited, err := editLines(f.lines, set, unset, lineEnding(f.fence))
	if err != nil {
		return "", err
	}
	f.lines = []string{edited}
	return f.String(), nil
}

// entry is one top-level key and the lines [start, end) it occupies.
type entry struct {
	key   *yaml.Node
	value *yaml.Node
	start int
	end   int
}

func editLines(lines []string, set map[string]any, unset []string, nl string) (string, error) {
	entries, err := parseEntries(lines)
	if err != nil {
		return "", err
	}

	seen := map[string]bool{}
	var out strings.Builder
	next := 0
	for _, e := range entries {
		out.WriteString(strings.Join(lines[next:e.start], ""))
		next = e.end
		name := e.key.Value
		seen[name] = true

		if slices.Contains(unset, name) {
			continue
		}
		value, ok := set[name]
		if !ok {
			out.WriteString(strings.Join(lines[e.start:e.end], ""))
			continue
		}
		text, encErr := encodeReplacement(e, value, nl)
		if encErr != nil {
			return "", encErr
		}
		if text == "" {
			text = strings.Join(lines[e.start:e.end], "")
		}
		out.WriteString(text)
	}

	added, err := encodeNewKeys(set, seen, nl)
	if err != nil {
		return "", err
	}
	out.WriteString(added)
	out.WriteString(strings.Join(lines[next:], ""))
	return out.String(), nil
}

func parseRoot(text string) (*yaml.Node, error) {
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(text), &doc)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidYAML, err)
	}
	if len(doc.Content) == 0 {
		return nil, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, ErrNotMapping
	}
	if root.Style&yaml.FlowStyle != 0 {
		return nil, ErrFlowMapping
	}
	if root.Column != 1 {
		return nil, ErrIndentedMapping
	}
	if hasAlias(root) {
		return nil, ErrAlias
	}
	return root, nil
}

func hasAlias(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode {
		return true
	}
	return slices.ContainsFunc(node.Content, hasAlias)
}

// parseEntries finds each top-level key's lines. A key runs from its own line to
// the next key's line, less the blank and comment lines just before the next key:
// those belong to the next key or to the block's end, and stay put when this key
// changes. A trailing line is only let go while the key, read alone, keeps its
// value, so the blank lines of a |+ scalar and the "# ..." lines inside a block
// scalar stay with it.
func parseEntries(lines []string) ([]entry, error) {
	root, err := parseRoot(strings.Join(lines, ""))
	if err != nil || root == nil {
		return nil, err
	}

	entries := make([]entry, 0, len(root.Content)/2)
	names := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		if names[key.Value] {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateKey, key.Value)
		}
		names[key.Value] = true
		entries = append(entries, entry{key: key, value: root.Content[i+1], start: key.Line - 1})
	}
	for i := range entries {
		end := len(lines)
		if i+1 < len(entries) {
			end = entries[i+1].start
		}
		want, decErr := decodeValue(entries[i].value)
		if decErr != nil {
			return nil, decErr
		}
		for end > entries[i].start+1 && isGap(lines[end-1]) &&
			spanReadsAs(lines[entries[i].start:end-1], want) {
			end--
		}
		entries[i].end = end
	}
	return entries, nil
}

func isGap(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.TrimSpace(trimmed) == "" || strings.HasPrefix(trimmed, "#")
}

func spanReadsAs(lines []string, want string) bool {
	root, err := parseRoot(strings.Join(lines, ""))
	if err != nil || root == nil || len(root.Content) != 2 {
		return false
	}
	got, err := decodeValue(root.Content[1])
	return err == nil && got == want
}

// decodeValue renders what a node reads as, so two nodes compare equal when a
// reader would see the same value, NaN included.
func decodeValue(node *yaml.Node) (string, error) {
	var value any
	err := node.Decode(&value)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidYAML, err)
	}
	out, err := yaml.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("frontmatteredit: render value: %w", err)
	}
	return string(out), nil
}

// encodeReplacement renders the key with its new value, carrying the old value's
// line comment over. It returns "" when the value is the same scalar already
// there, so the original text — quoting included — is kept.
func encodeReplacement(e entry, value any, nl string) (string, error) {
	node, err := valueNode(value)
	if err != nil {
		return "", err
	}
	if node.Kind == yaml.ScalarNode && e.value.Kind == yaml.ScalarNode &&
		node.Value == e.value.Value && node.ShortTag() == e.value.ShortTag() {
		return "", nil
	}
	node.LineComment = e.value.LineComment
	if node.LineComment == "" {
		node.LineComment = e.key.LineComment
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: e.key.Tag, Style: e.key.Style, Value: e.key.Value}
	return encodePair(key, node, nl)
}

func encodeNewKeys(set map[string]any, seen map[string]bool, nl string) (string, error) {
	keys := make([]string, 0, len(set))
	for key := range set {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	var out strings.Builder
	for _, key := range keys {
		node, err := valueNode(set[key])
		if err != nil {
			return "", err
		}
		text, err := encodePair(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, node, nl)
		if err != nil {
			return "", err
		}
		out.WriteString(text)
	}
	return out.String(), nil
}

func valueNode(value any) (*yaml.Node, error) {
	var node yaml.Node
	err := node.Encode(quoteUnsafeStrings(normalize(value)))
	if err != nil {
		return nil, fmt.Errorf("frontmatteredit: encode value: %w", err)
	}
	return &node, nil
}

// quoteUnsafeStrings double-quotes the strings a block scalar cannot carry: one
// that starts with a line break, which a block scalar drops, and one with a line
// of dashes only, which goldmark-meta would take for the closing fence.
func quoteUnsafeStrings(value any) any {
	switch v := value.(type) {
	case string:
		if !needsDoubleQuotes(v) {
			return v
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: v}
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = quoteUnsafeStrings(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = quoteUnsafeStrings(item)
		}
		return out
	default:
		return value
	}
}

func needsDoubleQuotes(s string) bool {
	if breakLen(s) > 0 {
		return true
	}
	lines := splitYAMLLines(s)
	return len(lines) > 1 && slices.ContainsFunc(lines, isFence)
}

func encodePair(key, value *yaml.Node, nl string) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{key, value}})
	if err != nil {
		return "", fmt.Errorf("frontmatteredit: encode key %s: %w", key.Value, err)
	}
	err = enc.Close()
	if err != nil {
		return "", fmt.Errorf("frontmatteredit: encode key %s: %w", key.Value, err)
	}
	return strings.ReplaceAll(buf.String(), "\n", nl), nil
}

// normalize turns JSON-decoded numbers into the integers and floats they spell,
// so 42 is written as 42 and not as the string "42".
func normalize(value any) any {
	switch v := value.(type) {
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return i
		}
		f, err := v.Float64()
		if err == nil {
			return f
		}
		return v.String()
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = normalize(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalize(item)
		}
		return out
	default:
		return value
	}
}
