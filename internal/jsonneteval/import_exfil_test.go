package jsonneteval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/jsonneteval"
)

// TestEvalJSON_ImportstrCannotReadFiles is a regression guard for the arbitrary
// server file-read vulnerability: a jsonnet snippet (reachable from note
// frontmatter patches and webhook transforms) must not be able to read files
// from disk via importstr/import. The deny-all importer makes such snippets
// error out instead of leaking file contents into the rendered output.
func TestEvalJSON_ImportstrCannotReadFiles(t *testing.T) {
	secret := "TOP-SECRET-" + t.Name()
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(secretPath, []byte(secret), 0o600))

	src := `{ leaked: importstr "` + secretPath + `" }`

	out, err := jsonneteval.EvalJSON(src, nil)

	// With the deny-all importer the snippet must fail, and the secret must
	// never appear in the output.
	require.Error(t, err, "importstr must be rejected, not read the file")
	require.NotContains(t, string(out), secret, "secret file content leaked into output")
}

// TestEvalJSON_PureJsonnetStillWorks ensures the deny-all importer does not
// break legitimate pure jsonnet transforms.
func TestEvalJSON_PureJsonnetStillWorks(t *testing.T) {
	src := `local p = std.parseJson(std.extVar("payload")); { doubled: p.n * 2 }`
	out, err := jsonneteval.EvalJSON(src, map[string]string{"payload": `{"n":21}`})
	require.NoError(t, err)
	require.Contains(t, string(out), "42", "pure jsonnet must still evaluate")
}
