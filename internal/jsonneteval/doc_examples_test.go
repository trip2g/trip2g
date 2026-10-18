package jsonneteval_test

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/jsonneteval"
)

var docBlockRe = regexp.MustCompile("(?s)```jsonnet\n(.*?)\n```")

// TestJsonnetDocExamples runs every example on the user Jsonnet pages that
// ends with a "// → <json>" comment and checks the result against it.
func TestJsonnetDocExamples(t *testing.T) {
	for _, page := range []string{"en/user/jsonnet", "ru/user/jsonnet"} {
		doc, err := os.ReadFile("../../docs/" + page + ".md")
		require.NoError(t, err)

		ran := 0
		for _, m := range docBlockRe.FindAllStringSubmatch(string(doc), -1) {
			src := m[1]
			i := strings.LastIndex(src, "\n// → ")
			if i < 0 {
				continue
			}
			// A long result continues on further "//" lines.
			var want strings.Builder
			for _, line := range strings.Split(src[i+len("\n// → "):], "\n") {
				want.WriteString(strings.TrimPrefix(line, "//") + " ")
			}

			got, evalErr := jsonneteval.EvalJSON(src, nil)
			require.NoError(t, evalErr, "%s:\n%s", page, src)
			require.JSONEq(t, want.String(), string(got), "%s:\n%s", page, src)
			ran++
		}
		require.Positive(t, ran, "%s has no runnable examples", page)
	}
}

// The "common mistakes" section quotes these failures.
func TestJsonnetDocMistakes(t *testing.T) {
	_, err := jsonneteval.EvalJSON(`local meta = {}; meta.x`, nil)
	require.ErrorContains(t, err, "Field does not exist: x")

	_, err = jsonneteval.EvalJSON(`'\d'`, nil)
	require.Error(t, err)

	out, err := jsonneteval.EvalJSON(`if false then 1`, nil)
	require.NoError(t, err)
	require.JSONEq(t, "null", string(out))

	_, err = jsonneteval.EvalJSON(`std.extVar('attached_notes')`, map[string]string{"change": "[]"})
	require.ErrorContains(t, err, "Undefined external variable")

	var s string
	out, err = jsonneteval.EvalJSON(`'a' + 1`, nil)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(out, &s))
	require.Equal(t, "a1", s)
}
