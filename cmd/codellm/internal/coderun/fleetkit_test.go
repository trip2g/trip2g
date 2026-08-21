package coderun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/webhookutil"
)

// fleetkitDest is where Dockerfile.codellm must install the helper module: on
// python's default sys.path AND inside the /usr/lib subtree landlock grants.
const (
	fleetkitPyDest   = "/usr/lib/python3/dist-packages/fleetkit.py"
	fleetkitNodeDest = "/usr/lib/node_modules/fleetkit"
)

// fleetkitAPI is the surface role notes import. Vault notes are outside this
// repo, have no build step and are not rolled back with the image, so removing
// or renaming one of these breaks them at runtime with nothing to catch it.
func fleetkitAPI() []string {
	return []string{
		"render", "note", "write", "patch", "emit", "bag",
		"frontmatter", "secrets", "note_frontmatter", "parse_frontmatter",
	}
}

func repoFile(t *testing.T, rel string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", rel))
	require.NoError(t, err)
	return string(body)
}

// TestFleetkit_ShippedToSandboxPath pins the Dockerfile against the sandbox:
// the helper is worthless if it lands somewhere a block cannot import, and
// /usr/local is denied exactly like the pip packages above it.
func TestFleetkit_ShippedToSandboxPath(t *testing.T) {
	dockerfile := repoFile(t, "Dockerfile.codellm")

	require.Contains(t, dockerfile, fleetkitPyDest,
		"Dockerfile.codellm must install fleetkit on the default sys.path under /usr/lib")
	require.Contains(t, dockerfile, "cmd/codellm/fleetkit/fleetkit.py",
		"the module must be COPYed from its source in the repo")

	// NODE_PATH covers require(); the /tmp/node_modules symlink covers import.
	// Both resolve through /usr/lib/node_modules, so the twin must land there.
	require.Contains(t, dockerfile, fleetkitNodeDest,
		"the node twin must install where NODE_PATH and the ESM symlink both resolve")
	require.Contains(t, dockerfile, "cmd/codellm/fleetkit/node",
		"the node twin must be COPYed from its source in the repo")
}

// TestFleetkit_TwinsAreSymmetric holds the two halves to the same surface. The
// point of the node twin is that a role moving between languages keeps the same
// calls, so a function added to one and forgotten in the other is a defect.
func TestFleetkit_TwinsAreSymmetric(t *testing.T) {
	py := repoFile(t, "cmd/codellm/fleetkit/fleetkit.py")
	js := repoFile(t, "cmd/codellm/fleetkit/node/index.js")

	for _, fn := range fleetkitAPI() {
		require.Contains(t, py, "def "+fn+"(", "python fleetkit must export %q", fn)
		require.Contains(t, js, "function "+fn+"(", "node fleetkit must export %q", fn)
		require.Contains(t, js, fn, "node fleetkit must include %q in module.exports", fn)
	}
}

// TestFleetkit_ToolboxPinsValidators pins the validator packages against the
// README that advertises them: a block importing one that was dropped from the
// image fails at runtime inside a delivery.
func TestFleetkit_ToolboxPinsValidators(t *testing.T) {
	dockerfile := repoFile(t, "Dockerfile.codellm")

	require.Contains(t, dockerfile, "jsonschema", "python needs a JSON Schema validator")
	require.Contains(t, dockerfile, "ajv", "node needs a JSON Schema validator")
	require.Contains(t, dockerfile, " yaml", "the node twin needs a YAML package")
}

// TestFleetkit_SourceExports pins the helper's surface. Role notes live in user
// vaults outside this repo, so removing or renaming one of these breaks them at
// runtime with nothing to catch it first.
func TestFleetkit_SourceExports(t *testing.T) {
	src := repoFile(t, "cmd/codellm/fleetkit/fleetkit.py")

	for _, fn := range fleetkitAPI() {
		require.Contains(t, src, "def "+fn+"(",
			"fleetkit must keep exporting %q — vault role notes depend on it", fn)
	}
}

// TestFleetkit_EmitParsesAsTheContract feeds what emit() prints back through the
// real parser. The helper exists to stop roles re-deriving this shape, so the
// two must be pinned to each other.
func TestFleetkit_EmitParsesAsTheContract(t *testing.T) {
	// Verbatim stdout of:
	//   emit([note("a.md", {"title": "T: q"}, "# T\n"), patch("b.md", "x", "y")], "done")
	const stdout = `{"changes": [{"path": "a.md", "content": "---\ntitle: 'T: q'\n---\n# T\n"}, ` +
		`{"path": "b.md", "find": "x", "replace": "y"}], "answer": "done"}`

	changes, answer, err := ParseCodeOutput(stdout)
	require.NoError(t, err)
	require.Equal(t, "done", answer)
	require.Len(t, changes, 2)

	require.Equal(t, webhookutil.AgentChangeKindWrite, changes[0].Kind)
	require.Equal(t, "a.md", changes[0].Path)
	require.True(t, strings.HasPrefix(changes[0].Content, "---\ntitle: 'T: q'\n---\n"),
		"render() must quote a title carrying a colon, got: %q", changes[0].Content)

	require.Equal(t, webhookutil.AgentChangeKindPatch, changes[1].Kind)
	require.Equal(t, "x", changes[1].Find)
	require.Equal(t, "y", changes[1].Replace)
}

// TestFleetkit_EmptyEmitIsValid: a role that decides there is nothing to write
// must still produce parseable stdout rather than an empty run.
func TestFleetkit_EmptyEmitIsValid(t *testing.T) {
	changes, answer, err := ParseCodeOutput(`{"changes": [], "answer": "nothing to do"}`)
	require.NoError(t, err)
	require.Empty(t, changes)
	require.Equal(t, "nothing to do", answer)
}
