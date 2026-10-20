package mdloader_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

func validatorNote(jsonnet string) string {
	return "Checks headings.\n\n```jsonnet\n" + jsonnet + "\n```\n"
}

func loadWarnings(t *testing.T, sources map[string]string, path string) []model.NoteWarning {
	t.Helper()

	srcs := make([]mdloader.SourceFile, 0, len(sources))
	for p, content := range sources {
		srcs = append(srcs, mdloader.SourceFile{Path: p, Content: []byte(content)})
	}

	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log, Validate: true})
	require.NoError(t, err)

	note := nvs.PathMap[path]
	require.NotNil(t, note, path)

	return note.Warnings
}

func TestValidateOnlyWhenAsked(t *testing.T) {
	srcs := []mdloader.SourceFile{
		{Path: "rules.md", Content: []byte(validatorNote(`[{level: 'error', msg: 'ran'}]`))},
		{Path: "note.md", Content: []byte("---\nvalidate: \"[[rules]]\"\n---\nbody\n")},
		{Path: "missing.md", Content: []byte("---\nvalidate: \"[[nope]]\"\n---\nbody\n")},
	}

	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)

	require.Empty(t, nvs.PathMap["note.md"].Warnings)
	require.Empty(t, nvs.PathMap["missing.md"].Warnings)
}

func TestValidate(t *testing.T) {
	headingsRule := validatorNote(`
[
  {level: 'error', msg: 'h' + n.level + ' ' + n.text, line: n.line}
  for n in nodes if n.type == 'heading'
] + [
  {level: 'warning', msg: 'title ' + meta.title + ' at ' + path},
]`)

	tests := []struct {
		name     string
		sources  map[string]string
		path     string
		expected []model.NoteWarning
	}{
		{
			name: "messages map to warning levels with lines",
			sources: map[string]string{
				"rules.md": headingsRule,
				"note.md":  "---\nvalidate: \"[[rules]]\"\ntitle: Hello\n---\n# One\n\ntext\n\n## Two\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: line 5: h1 One"},
				{Level: model.NoteWarningCritical, Message: "validate rules.md: line 9: h2 Two"},
				{Level: model.NoteWarningWarning, Message: "validate rules.md: title Hello at note.md"},
			},
		},
		{
			name: "list of path and wikilink targets",
			sources: map[string]string{
				"lint/a.md": validatorNote(`[{level: 'error', msg: 'from a'}]`),
				"lint/b.md": validatorNote(`[{level: 'warning', msg: 'from b'}]`),
				"note.md":   "---\nvalidate:\n  - lint/a.md\n  - \"[[b]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate lint/a.md: from a"},
				{Level: model.NoteWarningWarning, Message: "validate lint/b.md: from b"},
			},
		},
		{
			name: "every jsonnet block of a validator runs",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'error', msg: 'one'}]`) + validatorNote(`[{level: 'error', msg: 'two'}]`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: one"},
				{Level: model.NoteWarningCritical, Message: "validate rules.md: two"},
			},
		},
		{
			name: "empty result means valid",
			sources: map[string]string{
				"rules.md": validatorNote(`[]`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path:     "note.md",
			expected: nil,
		},
		{
			name: "missing validator is critical",
			sources: map[string]string{
				"note.md": "---\nvalidate: \"[[nope]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate: target not found: [[nope]]"},
			},
		},
		{
			name: "validator without jsonnet blocks is critical",
			sources: map[string]string{
				"rules.md": "just text\n",
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: no jsonnet code blocks found"},
			},
		},
		{
			name: "runtime error is critical",
			sources: map[string]string{
				"rules.md": validatorNote(`error 'boom'`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: RUNTIME ERROR: boom"},
			},
		},
		{
			name: "result that is not a message list is critical",
			sources: map[string]string{
				"rules.md": validatorNote(`{ok: true}`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: result must be an array of {level, msg, line?}"},
			},
		},
		{
			name: "unknown level is critical",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'fatal', msg: 'x'}]`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: unknown level \"fatal\" (want error or warning)"},
			},
		},
		{
			name: "non-markdown target is critical, not a crash",
			sources: map[string]string{
				"board.canvas": `{"nodes":[],"edges":[]}`,
				"note.md":      "---\nvalidate: \"[[board.canvas]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate: board.canvas is not a markdown note"},
			},
		},
		{
			name: "unquoted wikilink works",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'error', msg: 'ran'}]`),
				"note.md":  "---\nvalidate: [[rules]]\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: ran"},
			},
		},
		{
			name: "unquoted wikilinks in a list work",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'error', msg: 'ran'}]`),
				"note.md":  "---\nvalidate:\n  - [[rules]]\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: ran"},
			},
		},
		{
			name: "non-string value is critical",
			sources: map[string]string{
				"note.md": "---\nvalidate: 42\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate: expected a note or a list of notes, got 42"},
			},
		},
		{
			name: "empty value means no validators",
			sources: map[string]string{
				"note.md": "---\nvalidate:\n---\nbody\n",
			},
			path:     "note.md",
			expected: nil,
		},
		{
			name: "blank string means no validators",
			sources: map[string]string{
				"note.md": "---\nvalidate: \" \"\n---\nbody\n",
			},
			path:     "note.md",
			expected: nil,
		},
		{
			name: "patch switches validation off with null",
			sources: map[string]string{
				"off.md": "---\ntype: frontmatter-patch\ninclude: [\"note.md\"]\n---\n" +
					"```jsonnet\n{ validate: null }\n```\n",
				"note.md": "---\nvalidate: \"[[nope]]\"\n---\nbody\n",
			},
			path:     "note.md",
			expected: nil,
		},
		{
			name: "patch switches validation off with an empty string",
			sources: map[string]string{
				"off.md": "---\ntype: frontmatter-patch\ninclude: [\"note.md\"]\n---\n" +
					"```jsonnet\n{ validate: '' }\n```\n",
				"note.md": "---\nvalidate: \"[[nope]]\"\n---\nbody\n",
			},
			path:     "note.md",
			expected: nil,
		},
		{
			name: "blank item in a list is critical",
			sources: map[string]string{
				"note.md": "---\nvalidate:\n  - \"\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate: expected a note or a list of notes, got []"},
			},
		},
		{
			name: "same validator listed twice runs once",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'error', msg: 'ran'}]`),
				"note.md":  "---\nvalidate:\n  - \"[[rules]]\"\n  - rules.md\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: ran"},
			},
		},
		{
			name: "null result is critical",
			sources: map[string]string{
				"rules.md": validatorNote(`if false then []`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: result must be an array of {level, msg, line?}"},
			},
		},
		{
			name: "message without msg is critical",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'warning'}]`),
				"note.md":  "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: message without msg"},
			},
		},
		{
			name: "frontmatter patch assigns a validator to a folder",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'error', msg: 'ran'}]`),
				"patch.md": "---\ntype: frontmatter-patch\ninclude: [\"blog/**\"]\n---\n" +
					"```jsonnet\n{ validate: ['[[rules]]'] }\n```\n",
				"blog/post.md": "body\n",
			},
			path: "blog/post.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: ran"},
			},
		},
		{
			name: "appending patches stack with the note's own validate",
			sources: map[string]string{
				"own.md":    validatorNote(`[{level: 'error', msg: 'own'}]`),
				"folder.md": validatorNote(`[{level: 'error', msg: 'folder'}]`),
				"vault.md":  validatorNote(`[{level: 'error', msg: 'vault'}]`),
				"p1.md": "---\ntype: frontmatter-patch\ninclude: [\"blog/**\"]\n---\n```jsonnet\n" +
					"local own = std.get(meta, 'validate', []);\n" +
					"local list = if std.isString(own) then [own] else own;\n" +
					"{ validate: list + ['[[folder]]'] }\n```\n",
				"p2.md": "---\ntype: frontmatter-patch\ninclude: [\"**/*.md\"]\npriority: 1\n---\n```jsonnet\n" +
					"local own = std.get(meta, 'validate', []);\n" +
					"local list = if std.isString(own) then [own] else own;\n" +
					"{ validate: list + ['[[vault]]'] }\n```\n",
				"blog/post.md": "---\nvalidate: \"[[own]]\"\n---\nbody\n",
			},
			path: "blog/post.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate own.md: own"},
				{Level: model.NoteWarningCritical, Message: "validate folder.md: folder"},
				{Level: model.NoteWarningCritical, Message: "validate vault.md: vault"},
			},
		},
		{
			name: "heading fragment in a target is ignored",
			sources: map[string]string{
				"rules.md": validatorNote(`[{level: 'error', msg: 'ran'}]`),
				"note.md":  "---\nvalidate: \"[[rules#Section]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: ran"},
			},
		},
		{
			name: "regexMatch native",
			sources: map[string]string{
				"rules.md": validatorNote(`
local re = std.native('regexMatch');
[{level: 'error', msg: 'bad date'} for d in [meta.date] if !re('^\\d{4}-\\d{2}-\\d{2}$', d)]`),
				"a_good.md": "---\nvalidate: \"[[rules]]\"\ndate: 2026-10-16\n---\nbody\n",
				"b_bad.md":  "---\nvalidate: \"[[rules]]\"\ndate: 16.10.2026\n---\nbody\n",
			},
			path: "b_bad.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: bad date"},
			},
		},
		{
			name: "invalid regexMatch pattern is critical",
			sources: map[string]string{
				"rules.md": validatorNote(`
local re = std.native('regexMatch');
[{level: 'error', msg: 'never'} for x in [1] if re('(', 'a')]`),
				"note.md": "---\nvalidate: \"[[rules]]\"\n---\nbody\n",
			},
			path: "note.md",
			expected: []model.NoteWarning{
				{Level: model.NoteWarningCritical, Message: "validate rules.md: RUNTIME ERROR: error parsing regexp: missing closing ): `(`"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loadWarnings(t, tt.sources, tt.path)
			if diff := pretty.Diff(tt.expected, got); len(diff) > 0 {
				t.Errorf("warnings mismatch:\n%s\ngot: %# v", diff, pretty.Formatter(got))
			}
		})
	}
}

func TestValidateNodes(t *testing.T) {
	rules := validatorNote(`[
  {level: 'warning', msg: std.manifestJsonMinified(n), line: n.line}
  for n in nodes
]`)

	note := "---\nvalidate: \"[[rules]]\"\n---\n" +
		"## Session 1\n\n" + // line 4
		"Intro with [site](https://example.com) and [[other]].\n\n" + // line 6
		"![[cat.png|A cat]]\n\n" + // line 8
		"![dog](dog.jpg)\n\n" + // line 10
		"- item\n\n" + // line 12
		"> quote\n\n" + // line 14
		"```go\nfmt.Println()\n```\n" // line 16

	got := loadWarnings(t, map[string]string{"rules.md": rules, "note.md": note, "other.md": "x"}, "note.md")

	var msgs []string
	for _, w := range got {
		msgs = append(msgs, w.Message)
	}

	expected := []string{
		`validate rules.md: line 4: {"depth":1,"level":2,"line":4,"text":"Session 1","type":"heading"}`,
		`validate rules.md: line 6: {"depth":1,"line":6,"text":"Intro with site and other.","type":"paragraph"}`,
		`validate rules.md: line 6: {"depth":2,"line":6,"text":"site","type":"link","url":"https://example.com"}`,
		`validate rules.md: line 6: {"depth":2,"embed":false,"line":6,"target":"other","type":"wikilink"}`,
		`validate rules.md: line 8: {"depth":1,"line":8,"text":"A cat","type":"paragraph"}`,
		`validate rules.md: line 8: {"alt":"A cat","depth":2,"line":8,"src":"cat.png","type":"image"}`,
		`validate rules.md: line 10: {"depth":1,"line":10,"text":"dog","type":"paragraph"}`,
		`validate rules.md: line 10: {"alt":"dog","depth":2,"line":10,"src":"dog.jpg","type":"image"}`,
		`validate rules.md: line 12: {"depth":1,"line":12,"ordered":false,"type":"list"}`,
		`validate rules.md: line 12: {"depth":2,"line":12,"type":"list_item"}`,
		`validate rules.md: line 12: {"depth":3,"line":12,"text":"item","type":"paragraph"}`,
		`validate rules.md: line 14: {"depth":1,"line":14,"type":"blockquote"}`,
		`validate rules.md: line 14: {"depth":2,"line":14,"text":"quote","type":"paragraph"}`,
		`validate rules.md: line 16: {"depth":1,"lang":"go","line":16,"type":"code_block"}`,
	}

	if diff := pretty.Diff(expected, msgs); len(diff) > 0 {
		t.Errorf("nodes mismatch:\n%s\ngot: %# v", diff, pretty.Formatter(msgs))
	}
}

// TestValidateDemoVault runs the demo validators in testdata/validate. They
// live outside docs/ because `trip2g lint docs` in CI would fail on the
// deliberately invalid notes.
func TestValidateDemoVault(t *testing.T) {
	root := "testdata/validate"

	var srcs []mdloader.SourceFile
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, p)
		srcs = append(srcs, mdloader.SourceFile{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	require.NoError(t, err)

	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log, Validate: true})
	require.NoError(t, err)

	expected := map[string][]string{
		"notes/article_ok.md": nil,
		"notes/article_bad.md": {
			"CRITICAL validate validators/required_frontmatter.md: missing frontmatter field: description",
			"CRITICAL validate validators/required_frontmatter.md: tags must be a list",
			`CRITICAL validate validators/seo.md: date must be YYYY-MM-DD, got "16.10.2026"`,
			"CRITICAL validate validators/sections_order.md: sections must be Intro → Steps → FAQ, got Steps → Intro",
			"WARNING validate validators/media_hygiene.md: line 14: image publish.png has no alt text",
			"CRITICAL validate validators/media_hygiene.md: line 16: code block without a language",
			"CRITICAL validate validators/external_links.md: line 22: link to untrusted domain: https://example.com/spam",
		},
		"notes/seo_long_description.md": {
			"WARNING validate validators/seo.md: description is 168 chars, keep it under 160",
		},
		"notes/faq_ok.md": nil,
		"notes/faq_bad.md": {
			`CRITICAL validate validators/only_h3.md: line 4: heading "FAQ" is h1, only h3 is allowed`,
			`CRITICAL validate validators/only_h3.md: line 10: heading "Billing" is h2, only h3 is allowed`,
		},
		"notes/workshop_ok.md": nil,
		"notes/board_ok.md":    nil,
		"notes/board_bad.md": {
			"CRITICAL validate validators/app_config.md: columns must be a non-empty list",
			"CRITICAL validate validators/app_config.md: wip_limit must be a whole number",
			"WARNING validate validators/app_config.md: show_archive must be true or false",
		},
		"notes/workshop_bad.md": {
			`CRITICAL validate validators/session_has_image.md: line 8: "Session 2: publish" has no image`,
		},
		"notes/{blog} {article} Weekly summary – 2026-10-18 – Cursor.md": nil,
		"notes/{blog} {2026-10-18} {article} Weekly summary.md": {
			`CRITICAL validate validators/file_name.md: name must be "{project} {type} title – YYYY-MM-DD – tool"`,
		},
		"notes/Weekly summary – 18.10.2026 – Notepad.md": {
			"CRITICAL validate validators/file_name.md: name must start with {project} {type} and a title",
			`CRITICAL validate validators/file_name.md: date "18.10.2026" must be YYYY-MM-DD`,
			`WARNING validate validators/file_name.md: new tool "Notepad"? add it to the list in this validator`,
		},
		"notes/{blog} {article} Weekly summary – 2026-10-18 – claude-code.md": {
			`CRITICAL validate validators/file_name.md: tool "claude-code" must be a name like Claude Code, Cursor, ChatGPT, Gemini, Windsurf, Manual`,
		},
		"notes/{blog} {article} Weekly summary – 2026-10-17 – Claude Code.md": {
			"CRITICAL validate validators/file_name.md: frontmatter date 2026-10-18 differs from 2026-10-17 in the name",
		},
		"notes/workshop_and_seo.md": {
			`CRITICAL validate validators/session_has_image.md: line 9: "Session 1: setup" has no image`,
		},
	}

	for path, want := range expected {
		note := nvs.PathMap[path]
		require.NotNil(t, note, path)

		var got []string
		for _, w := range note.Warnings {
			if !strings.Contains(w.Message, "validate") {
				continue
			}
			level := "WARNING"
			if w.Level == model.NoteWarningCritical {
				level = "CRITICAL"
			}
			got = append(got, level+" "+w.Message)
		}

		if diff := pretty.Diff(want, got); len(diff) > 0 {
			t.Errorf("%s:\n%s\ngot: %# v", path, diff, pretty.Formatter(got))
		}
	}
}

// TestValidateDocsSnippets keeps the examples on the user pages identical to
// the demo validators that TestValidateDemoVault runs.
func TestValidateDocsSnippets(t *testing.T) {
	examples := []string{"required_frontmatter", "sections_order", "session_has_image", "media_hygiene", "seo", "file_name"}

	pages := map[string][]string{
		"en/user/validators":                 examples,
		"ru/user/validators":                 examples,
		"en/thoughts/file-naming-convention": {"file_name"},
		"ru/thoughts/file-naming-convention": {"file_name"},
		"en/user/spa":                        {"app_config"},
		"ru/user/spa":                        {"app_config"},
	}

	// The folder patch on both user pages appends to the note's own validate.
	appendPatch := "local own = std.get(meta, 'validate', []);\n" +
		"local list = if std.isString(own) then [own] else own;\n" +
		"{ validate: list + ['[[required_frontmatter]]', '[[seo]]'] }"

	for page, names := range pages {
		doc, err := os.ReadFile("../../docs/" + page + ".md")
		require.NoError(t, err)

		for _, name := range names {
			src, readErr := os.ReadFile("testdata/validate/validators/" + name + ".md")
			require.NoError(t, readErr)

			_, block, ok := strings.Cut(string(src), "```jsonnet\n")
			require.True(t, ok, name)
			block, _, _ = strings.Cut(block, "\n```")

			require.Contains(t, string(doc), "```jsonnet\n"+block+"\n```", "docs/%s.md lost the %s example", page, name)
		}
		if len(names) > 1 {
			require.Contains(t, string(doc), appendPatch, "docs/%s.md lost the folder patch", page)
		}
	}
}

// TestValidateOntologyVault runs the typed knowledge base essay's demo: a
// folder patch types the notes, the ontology validator checks them, and the
// essay pages keep both code blocks identical to the demo.
func TestValidateOntologyVault(t *testing.T) {
	root := "testdata/ontology"

	var srcs []mdloader.SourceFile
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, p)
		srcs = append(srcs, mdloader.SourceFile{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	require.NoError(t, err)

	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log, Validate: true})
	require.NoError(t, err)

	expected := map[string]struct {
		nodeType any
		warnings []string
	}{
		"docs/README.md":                     {"index", nil},
		"docs/decisions/drop-firecracker.md": {"decision", nil},
		"docs/decisions/old-sandbox.md":      {"decision", nil},
		"docs/plans/billing-v2.md":           {"plan", nil},
		"docs/notes.md": {nil, []string{
			"CRITICAL validate validators/ontology.md: missing node_type",
		}},
		"docs/ops/restart-db.md": {"runbook", []string{
			"CRITICAL validate validators/ontology.md: a runbook needs description",
			`WARNING validate validators/ontology.md: status "live" is not one of: active, draft, deprecated, archived`,
			`WARNING validate validators/ontology.md: unknown link type "see_also"`,
			`CRITICAL validate validators/ontology.md: verified.by "claude" must be human:<id>, process:<id> or tool/version`,
		}},
	}
	for path, want := range expected {
		note := nvs.PathMap[path]
		require.NotNil(t, note, path)
		require.Equal(t, want.nodeType, note.RawMeta["node_type"], path)
		var got []string
		for _, w := range note.Warnings {
			level := "WARNING"
			if w.Level == model.NoteWarningCritical {
				level = "CRITICAL"
			}
			got = append(got, level+" "+w.Message)
		}
		require.Equal(t, want.warnings, got, path)
	}

	blocks := map[string]string{}
	for _, name := range []string{"validators/ontology", "patches/types_by_folder"} {
		src, readErr := os.ReadFile(root + "/" + name + ".md")
		require.NoError(t, readErr)
		_, block, ok := strings.Cut(string(src), "```jsonnet\n")
		require.True(t, ok, name)
		block, _, _ = strings.Cut(block, "\n```")
		blocks[name] = block
	}
	for _, page := range []string{"en/thoughts/typed-knowledge-base", "ru/thoughts/typed-knowledge-base"} {
		doc, readErr := os.ReadFile("../../docs/" + page + ".md")
		require.NoError(t, readErr)
		for name, block := range blocks {
			require.Contains(t, string(doc), "```jsonnet\n"+block+"\n```", "docs/%s.md lost %s", page, name)
		}
	}
}
