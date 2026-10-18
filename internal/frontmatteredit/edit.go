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

	"gopkg.in/yaml.v3"
)

var (
	ErrNothingToChange = errors.New("set or unset must name at least one key")
	ErrEmptyKey        = errors.New("frontmatter key must not be empty")
	ErrSetAndUnset     = errors.New("a key cannot be both set and unset")
	ErrUnclosed        = errors.New("frontmatter has no closing --- line")
	ErrInvalidYAML     = errors.New("frontmatter is not valid YAML")
	ErrNotMapping      = errors.New("frontmatter is not a key: value mapping")
	ErrFlowMapping     = errors.New("frontmatter written as {...} cannot be edited key by key")
	ErrDuplicateKey    = errors.New("frontmatter declares the key more than once")
)

const bom = "\uFEFF"

// Apply returns content with each key of set added or replaced and each key of
// unset removed. Only top-level keys are addressed: "a.b" names a key literally
// called "a.b". Keys new to the note are appended after its last key in sorted
// order. A note without frontmatter gets a block when set is not empty.
func Apply(content string, set map[string]any, unset []string) (string, error) {
	err := validate(set, unset)
	if err != nil {
		return "", err
	}

	prefix := ""
	if strings.HasPrefix(content, bom) {
		prefix = bom
		content = content[len(bom):]
	}

	lines := strings.SplitAfter(content, "\n")
	if !isFence(lines[0]) {
		if len(set) == 0 {
			return prefix + content, nil
		}
		nl := lineEnding(lines[0])
		added, addErr := encodeNewKeys(set, nil, nl)
		if addErr != nil {
			return "", addErr
		}
		return prefix + "---" + nl + added + "---" + nl + content, nil
	}

	closing := slices.IndexFunc(lines[1:], isFence)
	if closing == -1 {
		return "", ErrUnclosed
	}
	body := lines[1 : closing+1]

	edited, err := editBody(body, set, unset, lineEnding(lines[0]))
	if err != nil {
		return "", err
	}

	return prefix + lines[0] + edited + strings.Join(lines[closing+1:], ""), nil
}

func validate(set map[string]any, unset []string) error {
	if len(set) == 0 && len(unset) == 0 {
		return ErrNothingToChange
	}
	for key := range set {
		if strings.TrimSpace(key) == "" {
			return ErrEmptyKey
		}
		if slices.Contains(unset, key) {
			return fmt.Errorf("%w: %s", ErrSetAndUnset, key)
		}
	}
	for _, key := range unset {
		if strings.TrimSpace(key) == "" {
			return ErrEmptyKey
		}
	}
	return nil
}

// isFence mirrors goldmark-meta, which decides what a note's meta is: a line of
// dashes only, surrounded by optional whitespace.
func isFence(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && strings.Trim(trimmed, "-") == ""
}

func lineEnding(line string) string {
	if strings.HasSuffix(line, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// entry is one top-level key and the body lines [start, end) it occupies.
type entry struct {
	key   *yaml.Node
	value *yaml.Node
	start int
	end   int
}

func editBody(body []string, set map[string]any, unset []string, nl string) (string, error) {
	entries, err := parseEntries(body)
	if err != nil {
		return "", err
	}

	seen := map[string]bool{}
	var out strings.Builder
	next := 0
	for _, e := range entries {
		out.WriteString(strings.Join(body[next:e.start], ""))
		next = e.end
		name := e.key.Value
		seen[name] = true

		if slices.Contains(unset, name) {
			continue
		}
		value, ok := set[name]
		if !ok {
			out.WriteString(strings.Join(body[e.start:e.end], ""))
			continue
		}
		text, encErr := encodeReplacement(e, value, nl)
		if encErr != nil {
			return "", encErr
		}
		if text == "" {
			text = strings.Join(body[e.start:e.end], "")
		}
		out.WriteString(text)
	}

	added, err := encodeNewKeys(set, seen, nl)
	if err != nil {
		return "", err
	}
	out.WriteString(added)
	out.WriteString(strings.Join(body[next:], ""))
	return out.String(), nil
}

// parseEntries finds each top-level key's lines. A key runs from its own line to
// the next key's line, less the blank and unindented comment lines just before
// the next key: those belong to the next key or to the block's end, and stay put
// when this key changes.
func parseEntries(body []string) ([]entry, error) {
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(strings.Join(body, "")), &doc)
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
		end := len(body)
		if i+1 < len(entries) {
			end = entries[i+1].start
		}
		for end > entries[i].start+1 && isGap(body[end-1]) {
			end--
		}
		entries[i].end = end
	}
	return entries, nil
}

func isGap(line string) bool {
	return strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#")
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
	err := node.Encode(normalize(value))
	if err != nil {
		return nil, fmt.Errorf("frontmatteredit: encode value: %w", err)
	}
	return &node, nil
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
