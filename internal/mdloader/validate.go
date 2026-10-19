package mdloader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
	jsonnetast "github.com/google/go-jsonnet/ast"
	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"go.abhg.dev/goldmark/wikilink"

	"trip2g/internal/image"
	"trip2g/internal/jsonneteval"
	"trip2g/internal/model"
	"trip2g/internal/yamlutil"
)

// validatePrelude gives validators the same meta and path locals as
// frontmatter patches, plus the note's nodes. It shares the block's first
// line, so jsonnet error line numbers match the block.
const validatePrelude = `local meta = std.parseJson(std.extVar("meta")), path = std.extVar("path"), nodes = std.native("nodes")(); `

const nodeTypeImage = "image"

var errBadValidateResult = errors.New("result must be an array of {level, msg, line?}")

type validateMessage struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Line  int    `json:"line"`
}

// validateNotes runs the jsonnet validators each note lists in its "validate"
// frontmatter and records their messages as note warnings. It never blocks a
// note: an "error" message only marks the note with a critical warning.
func (ldr *loader) validateNotes() {
	run := &validateRun{
		blocks:    map[*model.NoteView][]string{},
		blockErrs: map[*model.NoteView]error{},
	}

	paths := make([]string, 0, len(ldr.nvs.PathMap))
	for path := range ldr.nvs.PathMap {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	validated := map[string]bool{}
	defer func() { ldr.validateCache.retain(validated) }()

	for _, path := range paths {
		note := ldr.nvs.PathMap[path]
		raw, ok := note.RawMeta["validate"]
		if !ok {
			continue
		}

		targets, err := validateTargets(raw)
		if err != nil {
			note.AddWarning(model.NoteWarningCritical, "validate: %s", err)
			continue
		}

		validators := ldr.resolveValidators(note, targets)
		if len(validators) == 0 {
			continue
		}

		validated[path] = true
		key := validateKey(note, validators, ldr.patchSetHash)
		if cached, hit := ldr.validateCache.get(path, key); hit {
			note.Warnings = append(note.Warnings, cached...)
			continue
		}

		before := len(note.Warnings)
		run.validate(note, validators)
		ldr.validateCache.put(path, key, note.Warnings[before:])
	}
}

// resolveValidators turns targets into validator notes, warning about the ones
// that can't be used. A validator listed twice is kept once.
func (ldr *loader) resolveValidators(note *model.NoteView, targets []string) []*model.NoteView {
	var validators []*model.NoteView
	seen := map[*model.NoteView]bool{}

	for _, target := range targets {
		name, _, _ := strings.Cut(stripWikilink(target), "#")
		name = strings.TrimSuffix(name, ".md")
		validator := ldr.nvs.ResolveWikilinkTarget(note, name)
		if validator == nil {
			note.AddWarning(model.NoteWarningCritical, "validate: target not found: %s", target)
			continue
		}
		if seen[validator] {
			continue
		}
		seen[validator] = true

		// Canvas, base and excalidraw files resolve like notes but have no AST.
		if validator.Ast() == nil {
			note.AddWarning(model.NoteWarningCritical, "validate: %s is not a markdown note", validator.Path)
			continue
		}

		validators = append(validators, validator)
	}

	return validators
}

// validateRun holds what one Load shares across notes: the VM and the jsonnet
// blocks extracted from each validator.
type validateRun struct {
	vm        *jsonnet.VM
	blocks    map[*model.NoteView][]string
	blockErrs map[*model.NoteView]error
}

func (r *validateRun) validate(note *model.NoteView, validators []*model.NoteView) {
	metaJSON, err := json.Marshal(yamlutil.Normalize(note.RawMeta))
	if err != nil {
		note.AddWarning(model.NoteWarningCritical, "validate: %s", err)
		return
	}

	if r.vm == nil {
		r.vm = jsonneteval.NewVM()
		r.vm.NativeFunction(regexMatchNative())
	}
	r.vm.ExtVar("meta", string(metaJSON))
	r.vm.ExtVar("path", note.Path)
	r.vm.NativeFunction(nodesNative(note))

	for _, validator := range validators {
		if _, seen := r.blocks[validator]; !seen {
			r.blocks[validator], r.blockErrs[validator] = extractJsonnetBlocks(validator.Ast(), validator.Content)
		}

		if r.blockErrs[validator] != nil {
			note.AddWarning(model.NoteWarningCritical, "validate %s: %s", validator.Path, r.blockErrs[validator])
			continue
		}

		for _, src := range r.blocks[validator] {
			runValidator(r.vm, note, validator.Path, src)
		}
	}
}

// runValidator evaluates one jsonnet block; the caller has bound meta, path
// and nodes for the note.
func runValidator(vm *jsonnet.VM, note *model.NoteView, validatorPath, src string) {
	fail := func(err error) {
		note.AddWarning(model.NoteWarningCritical, "validate %s: %s", validatorPath, err)
	}

	out, err := vm.EvaluateAnonymousSnippet(validatorPath, validatePrelude+src)
	if err != nil {
		fail(errors.New(strings.TrimSpace(firstLine(err.Error()))))
		return
	}

	// A bare null would unmarshal into a nil slice and pass as valid.
	if !strings.HasPrefix(strings.TrimSpace(out), "[") {
		fail(errBadValidateResult)
		return
	}

	var messages []validateMessage
	err = json.Unmarshal([]byte(out), &messages)
	if err != nil {
		fail(errBadValidateResult)
		return
	}

	for _, m := range messages {
		var level model.NoteWarningLevel
		switch m.Level {
		case "error":
			level = model.NoteWarningCritical
		case "warning":
			level = model.NoteWarningWarning
		default:
			fail(fmt.Errorf("unknown level %q (want error or warning)", m.Level))
			continue
		}

		if m.Msg == "" {
			fail(errors.New("message without msg"))
			continue
		}

		if m.Line > 0 {
			note.AddWarning(level, "validate %s: line %d: %s", validatorPath, m.Line, m.Msg)
		} else {
			note.AddWarning(level, "validate %s: %s", validatorPath, m.Msg)
		}
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// validateTargets reads "validate" as one target or a list of them. An
// unquoted [[x]] is YAML for a nested list, so a one-item list counts as x.
// Null or a blank string lists no validators, so a patch can switch them off.
func validateTargets(raw interface{}) ([]string, error) {
	if s, isString := raw.(string); raw == nil || isString && strings.TrimSpace(s) == "" {
		return nil, nil
	}

	items, ok := raw.([]interface{})
	if !ok {
		items = []interface{}{raw}
	}

	targets := make([]string, 0, len(items))
	for _, item := range items {
		// [[x]] nests twice as a list item: [[["x"]]].
		for range 2 {
			nested, isList := item.([]interface{})
			if !isList || len(nested) != 1 {
				break
			}
			item = nested[0]
		}
		s, isString := item.(string)
		if !isString || strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("expected a note or a list of notes, got %v", raw)
		}
		targets = append(targets, s)
	}

	return targets, nil
}

func regexMatchNative() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "regexMatch",
		Params: jsonnetast.Identifiers{"pattern", "str"},
		Func: func(args []interface{}) (interface{}, error) {
			pattern, ok1 := args[0].(string)
			str, ok2 := args[1].(string)
			if !ok1 || !ok2 {
				return nil, errors.New("regexMatch(pattern, str) takes two strings")
			}
			return regexp.MatchString(pattern, str)
		},
	}
}

// nodesNative exposes the note's AST to jsonnet as a flat, document-order list
// of nodes. Jsonnet can't pass callbacks into native code, so instead of a
// walker the validator folds over this list.
func nodesNative(note *model.NoteView) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name: "nodes",
		Func: func(_ []interface{}) (interface{}, error) {
			return flattenNodes(note.Ast(), note.Content), nil
		},
	}
}

func flattenNodes(doc ast.Node, src []byte) []interface{} {
	nodes := []interface{}{}
	if doc == nil {
		return nodes
	}

	depth := 0
	blockLine := 0

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Kind() == ast.KindDocument {
			return ast.WalkContinue, nil
		}

		if !entering {
			depth--
			return ast.WalkContinue, nil
		}
		depth++

		if n.Type() == ast.TypeBlock {
			if line := nodeLine(n, src); line > 0 {
				blockLine = line
			}
		}

		node := map[string]interface{}{"depth": depth, "line": blockLine}

		switch v := n.(type) {
		case *ast.Heading:
			node["type"] = "heading"
			node["level"] = v.Level
			node["text"] = plainText(v, src)
		case *ast.Paragraph, *ast.TextBlock:
			// A tight list item holds a TextBlock; to a validator it's a paragraph.
			node["type"] = "paragraph"
			node["text"] = plainText(v, src)
		case *ast.FencedCodeBlock:
			node["type"] = "code_block"
			node["lang"] = string(v.Language(src))
			// Lines() start below the fence; point at the fence itself.
			switch {
			case v.Info != nil:
				node["line"] = bytes.Count(src[:v.Info.Segment.Start], []byte("\n")) + 1
			case v.Lines().Len() > 0:
				node["line"] = blockLine - 1
			default:
				// An empty bare fence has no position; don't borrow the previous block's.
				node["line"] = 0
			}
		case *ast.CodeBlock:
			node["type"] = "code_block"
			node["lang"] = ""
		case *ast.List:
			node["type"] = "list"
			node["ordered"] = v.IsOrdered()
		case *ast.ListItem:
			node["type"] = "list_item"
		case *ast.Blockquote:
			node["type"] = "blockquote"
		case *extast.Table:
			node["type"] = "table"
		case *ast.Link:
			node["type"] = "link"
			node["url"] = string(v.Destination)
			node["text"] = plainText(v, src)
		case *ast.Image:
			node["type"] = nodeTypeImage
			node["src"] = string(v.Destination)
			node["alt"] = plainText(v, src)
		case *enclavecore.Enclave:
			// The enclave extension swaps every ![](...) for its own node.
			node["type"] = nodeTypeImage
			node["src"] = string(v.Image.Destination)
			node["alt"] = plainText(&v.Image, src)
		case *wikilink.Node:
			target := string(v.Target)
			if v.Embed && image.IsRightExtension(target) {
				node["type"] = nodeTypeImage
				node["src"] = target
				node["alt"] = embedAlt(v, src)
			} else {
				node["type"] = "wikilink"
				node["target"] = target
				node["embed"] = v.Embed
			}
		default:
			return ast.WalkContinue, nil
		}

		nodes = append(nodes, node)
		return ast.WalkContinue, nil
	})

	return nodes
}

// embedAlt mirrors the image renderer: the label is alt text only when it
// differs from the target, minus any size suffix.
func embedAlt(n *wikilink.Node, src []byte) string {
	if n.ChildCount() != 1 {
		return ""
	}
	label := nodeText(src, n.FirstChild())
	if bytes.Equal(label, n.Target) {
		return ""
	}
	alt, _ := parseImageSize(string(label))
	return alt
}

func plainText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

// nodeLine returns the 1-based source line where a block starts, or 0 when the
// block carries no segments (lists and quotes start at their first child).
func nodeLine(n ast.Node, src []byte) int {
	lines := n.Lines()
	if lines != nil && lines.Len() > 0 {
		return bytes.Count(src[:lines.At(0).Start], []byte("\n")) + 1
	}
	for c := n.FirstChild(); c != nil; c = c.FirstChild() {
		if c.Type() != ast.TypeBlock {
			break
		}
		if l := c.Lines(); l != nil && l.Len() > 0 {
			return bytes.Count(src[:l.At(0).Start], []byte("\n")) + 1
		}
	}
	return 0
}
