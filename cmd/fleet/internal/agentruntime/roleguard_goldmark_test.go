package agentruntime

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

// goldmarkDeclaresRole reports what trip2g would actually see: internal/mdloader
// parses notes with goldmark-meta into NoteView.RawMeta, the GraphQL meta field
// serves that map, and fleet's DiscoverRoles reads fleet_id out of it.
func goldmarkDeclaresRole(t *testing.T, src string) bool {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(meta.Meta))
	ctx := parser.NewContext()
	require.NoError(t, md.Convert([]byte(src), &bytes.Buffer{}, parser.WithContext(ctx)))
	_, found := meta.Get(ctx)["fleet_id"]
	return found
}

// TestDeclaresRoleMatchesGoldmark is the guard's real contract: every note
// trip2g would treat as a role must be seen as one here. A hand-written table
// cannot establish that — it only encodes what the author already thought of,
// and the first version of this guard shipped with four bypasses (a "----"
// fence, mismatched fence lengths, a longer dash run, and an unterminated block
// that goldmark still closes at EOF) precisely because it was checked against
// such a table instead of against the parser it has to mirror.
//
// The other direction is allowed: seeing a role where goldmark would not is a
// false denial, which surfaces loudly in the run log.
func TestDeclaresRoleMatchesGoldmark(t *testing.T) {
	corpus := map[string]string{
		"exact fence":              "---\nfleet_id: evil\n---\nbody\n",
		"four dashes":              "----\nfleet_id: evil\n----\nbody\n",
		"mismatched fence lengths": "---\nfleet_id: evil\n----------\nbody\n",
		"long dash run":            "--------\nfleet_id: evil\n--------\nbody\n",
		"unterminated at eof":      "---\nfleet_id: evil\n",
		"unterminated no newline":  "---\nfleet_id: evil",
		"trailing spaces on fence": "--- \nfleet_id: evil\n--- \nbody\n",
		"trailing tab on fence":    "---\t\nfleet_id: evil\n---\t\nbody\n",
		"crlf":                     "---\r\nfleet_id: evil\r\n---\r\nbody\r\n",
		"quoted value":             "---\nfleet_id: \"evil\"\n---\n",
		"single quoted value":      "---\nfleet_id: 'evil'\n---\n",
		"empty value":              "---\nfleet_id:\n---\n",
		"null value":               "---\nfleet_id: ~\n---\n",
		"numeric value":            "---\nfleet_id: 42\n---\n",
		"duplicate keys":           "---\nfleet_id: a\nfleet_id: b\n---\n",
		"anchor and alias":         "---\nbase: &a evil\nfleet_id: *a\n---\n",
		"merge key":                "---\nbase: &a\n  fleet_id: evil\nmerged:\n  <<: *a\n---\n",
		"folded value":             "---\nfleet_id: >\n  evil\n---\n",
		"literal value":            "---\nfleet_id: |\n  evil\n---\n",
		"key with trailing space":  "---\nfleet_id : evil\n---\n",
		"document end marker":      "---\nfleet_id: evil\n...\n---\n",
		"comment before key":       "---\n# c\nfleet_id: evil\n---\n",
		"other keys around":        "---\ntitle: x\nfleet_id: evil\nmode: cron\n---\n",
		"body mentions fleet_id":   "---\ntitle: x\n---\n\nfleet_id: evil\n",
		"fenced block in body":     "---\ntitle: x\n---\n\n```yaml\nfleet_id: evil\n```\n",
		"nested only":              "---\nmeta:\n  fleet_id: evil\n---\n",
		"no frontmatter":           "# Title\n\nfleet_id: evil\n",
		"plain frontmatter":        "---\ntitle: x\n---\nbody\n",
		"invalid yaml":             "---\nfleet_id: [unclosed\n---\n",
		"empty":                    "",
		"list body":                "- item\n- item2\n",
		"thematic rule body":       "text\n\n---\n\nmore\n",
	}

	for name, src := range corpus {
		t.Run(name, func(t *testing.T) {
			if goldmarkDeclaresRole(t, src) {
				require.True(t, declaresRole(src),
					"BYPASS: trip2g would run this as a role, the guard would let an agent write it")
			}
		})
	}
}
