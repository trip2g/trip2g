package coderun

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The static pins in fleetkit_test.go only read the source. These run it: both
// twins are executed by their real interpreter, so a helper that parses, quotes
// or emits wrongly fails here rather than inside a delivery.
//
// Under FLEET_SANDBOX_MUST_ENFORCE=1 a missing interpreter or YAML package is a
// hard failure, matching the sandbox tests — otherwise CI could go green on
// tests that never ran.

func fleetkitSrc(t *testing.T, rel string) string {
	t.Helper()

	abs, err := filepath.Abs(filepath.Join("..", "..", "fleetkit", rel))
	require.NoError(t, err)
	return abs
}

// runPython executes src with the python fleetkit on PYTHONPATH.
func runPython(t *testing.T, src string) string {
	t.Helper()

	exe, err := exec.LookPath("python3")
	if err != nil {
		skipOrFail(t, "python3 not installed: %v", err)
	}
	return run(t, exe, t.TempDir(), "probe.py", src,
		[]string{"PYTHONPATH=" + fleetkitSrc(t, "")})
}

// runNode executes src against the node fleetkit. Dockerfile.codellm installs
// the module as .../node_modules/fleetkit, so the test reproduces that name:
// the source directory is called "node", and resolution keys on the directory
// name, not on what is inside it. yaml is linked in beside it because the
// module imports it, exactly as the image's global install provides it.
func runNode(t *testing.T, src string) string {
	t.Helper()

	exe, err := exec.LookPath("node")
	if err != nil {
		skipOrFail(t, "node not installed: %v", err)
	}

	dir := t.TempDir()
	mods := filepath.Join(dir, "node_modules")
	require.NoError(t, os.MkdirAll(mods, 0o755))
	require.NoError(t, os.Symlink(fleetkitSrc(t, "node"), filepath.Join(mods, "fleetkit")))

	yamlDir := findNodeYAML(t, exe)
	require.NoError(t, os.Symlink(yamlDir, filepath.Join(mods, "yaml")))

	return run(t, exe, dir, "probe.js", src, nil)
}

// findNodeYAML locates the yaml package the node twin imports. The runtime
// image installs it globally; a checkout may have it under node_modules. Not
// finding it is a skip locally and a hard failure in the enforcing lane, so CI
// cannot report green on a twin that never ran.
func findNodeYAML(t *testing.T, nodeExe string) string {
	t.Helper()

	candidates := []string{}
	if repo, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "node_modules", "yaml")); err == nil {
		candidates = append(candidates, repo)
	}
	if out, err := exec.Command(filepath.Join(filepath.Dir(nodeExe), "npm"), "root", "-g").Output(); err == nil {
		candidates = append(candidates, filepath.Join(strings.TrimSpace(string(out)), "yaml"))
	}
	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	skipOrFail(t, "node yaml package not found in %v; npm i -g yaml", candidates)
	return ""
}

// run writes src into dir and executes it, returning combined output.
func run(t *testing.T, exe, dir, name, src string, env []string) string {
	t.Helper()

	file := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(file, []byte(src), 0o600))

	cmd := exec.Command(exe, file)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil && strings.Contains(string(out), "yaml") {
		skipOrFail(t, "%s has no yaml package: %s", exe, strings.TrimSpace(string(out)))
	}
	require.NoError(t, err, "helper run failed: %s", out)
	return string(out)
}

// parseFrontmatterYAML reads the frontmatter block back with the repo's YAML
// library — the same one that parses a note after it is written.
func parseFrontmatterYAML(t *testing.T, content string) map[string]any {
	t.Helper()

	require.True(t, strings.HasPrefix(content, "---\n"), "no frontmatter: %q", content)
	end := strings.Index(content[3:], "\n---")
	require.GreaterOrEqual(t, end, 0, "unterminated frontmatter: %q", content)

	var out map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content[4:end+3]), &out))
	return out
}

// emitted is the stdout contract both twins must produce.
type emitted struct {
	Changes []map[string]string `json:"changes"`
	Answer  string              `json:"answer"`
}

func decodeEmitted(t *testing.T, stdout string) emitted {
	t.Helper()

	var got emitted
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got), "stdout: %q", stdout)
	return got
}

const pyProbe = `
import fleetkit
fleetkit.emit([
    fleetkit.note("a.md", {"title": "T: q", "created_at": "2026-01-15T10:00:00+00:00",
                           "truthy": "yes", "padded": "007"}, "# T\n"),
    fleetkit.patch("b.md", "old", "new"),
], "done")
`

const jsProbe = `
const fleetkit = require('fleetkit');
fleetkit.emit([
  fleetkit.note('a.md', {title: 'T: q', created_at: '2026-01-15T10:00:00+00:00',
                         truthy: 'yes', padded: '007'}, '# T\n'),
  fleetkit.patch('b.md', 'old', 'new'),
], 'done');
`

// TestFleetkitRuns_EmitsTheContract runs both twins and holds them to the same
// output: the awkward title quoted, the timestamp kept a string, and a patch
// distinguishable from a write by which keys are present.
func TestFleetkitRuns_EmitsTheContract(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T, string) string
		src  string
	}{
		{"python", runPython, pyProbe},
		{"node", runNode, jsProbe},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decodeEmitted(t, tc.run(t, tc.src))

			require.Equal(t, "done", got.Answer)
			require.Len(t, got.Changes, 2)

			require.Equal(t, "a.md", got.Changes[0]["path"])
			content := got.Changes[0]["content"]
			require.True(t, strings.HasPrefix(content, "---\n"), "content: %q", content)
			require.True(t, strings.HasSuffix(content, "---\n# T\n"), "content: %q", content)

			// Assert the meaning, not the quote character: the twins pick
			// different quoting and both are valid YAML. What must hold is that
			// a colon title and a timestamp-shaped value survive as strings —
			// emitted bare, the latter reads back as a timestamp instead.
			require.Equal(t,
				map[string]any{
					"title": "T: q", "created_at": "2026-01-15T10:00:00+00:00",
					// 'yes' and '007' are where the twins drifted: emitted bare
					// they read back as a boolean and an int.
					"truthy": "yes", "padded": "007",
				},
				parseFrontmatterYAML(t, content))

			require.Equal(t, "b.md", got.Changes[1]["path"])
			require.Equal(t, "old", got.Changes[1]["find"])
			require.Equal(t, "new", got.Changes[1]["replace"])
			require.NotContains(t, got.Changes[1], "content", "a patch must not carry content")

			// The emitted JSON is what the real parser consumes, so close the loop.
			parsed, answer, err := ParseCodeOutput(strings.TrimSpace(tc.run(t, tc.src)))
			require.NoError(t, err)
			require.Equal(t, "done", answer)
			require.Len(t, parsed, 2)
		})
	}
}

const pyFrontmatter = `
import json, fleetkit
cases = {
    "plain":    fleetkit.parse_frontmatter("---\ntitle: T\ntags: [a, b]\n---\n# body\n"),
    "none":     fleetkit.parse_frontmatter("# just a body\n"),
    "unclosed": fleetkit.parse_frontmatter("---\ntitle: T\n"),
    "notadict": fleetkit.parse_frontmatter("---\n- a\n- b\n---\nbody"),
}
missing = cases["plain"].nope
print(json.dumps({"cases": {k: dict(v) for k, v in cases.items()}, "missing": missing}))
`

const jsFrontmatter = `
const fleetkit = require('fleetkit');
const cases = {
  plain:    fleetkit.parse_frontmatter('---\ntitle: T\ntags: [a, b]\n---\n# body\n'),
  none:     fleetkit.parse_frontmatter('# just a body\n'),
  unclosed: fleetkit.parse_frontmatter('---\ntitle: T\n'),
  notadict: fleetkit.parse_frontmatter('---\n- a\n- b\n---\nbody'),
};
console.log(JSON.stringify({cases, missing: cases.plain.nope ?? null}));
`

// TestFleetkitRuns_ParseFrontmatter pins the parse half, including the shapes
// that are not a mapping: a lint role iterating a vault meets all of them, and
// any of them raising would take the whole delivery down.
func TestFleetkitRuns_ParseFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T, string) string
		src  string
	}{
		{"python", runPython, pyFrontmatter},
		{"node", runNode, jsFrontmatter},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got struct {
				Cases   map[string]map[string]any `json:"cases"`
				Missing any                       `json:"missing"`
			}
			out := tc.run(t, tc.src)
			require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &got), "stdout: %q", out)

			require.Equal(t, "T", got.Cases["plain"]["title"])
			require.Equal(t, []any{"a", "b"}, got.Cases["plain"]["tags"])
			require.Empty(t, got.Cases["none"], "a note with no frontmatter yields no fields")
			require.Empty(t, got.Cases["unclosed"], "an unterminated block yields no fields")
			require.Empty(t, got.Cases["notadict"], "a non-mapping document yields no fields")
			require.Nil(t, got.Missing, "a missing key must read as empty, not raise")
		})
	}
}

// pyRoleFrontmatter / jsRoleFrontmatter exercise frontmatter(): the role's own
// config, read from the delivery bag. Each probe writes its own bag and points
// FLEET_INPUT at it, since bag() resolves the env var per call.
const pyRoleFrontmatter = `
import json, os, tempfile, fleetkit

path = os.path.join(tempfile.mkdtemp(), "bag.json")
with open(path, "w") as fh:
    json.dump({"frontmatter": {"krisp_base_url": "https://api.krisp.ai", "max": "3"}}, fh)
os.environ["FLEET_INPUT"] = path

fm = fleetkit.frontmatter()
print(json.dumps({"url": fm.krisp_base_url, "max": fm.max, "missing": fm.nope}))
`

const jsRoleFrontmatter = `
const fs = require('fs'), os = require('os'), path = require('path');
const fleetkit = require('fleetkit');

const p = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'bag')), 'bag.json');
fs.writeFileSync(p, JSON.stringify({frontmatter: {krisp_base_url: 'https://api.krisp.ai', max: '3'}}));
process.env.FLEET_INPUT = p;

const fm = fleetkit.frontmatter();
console.log(JSON.stringify({url: fm.krisp_base_url, max: fm.max, missing: fm.nope ?? null}));
`

func TestFleetkitRuns_RoleFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T, string) string
		src  string
	}{
		{"python", runPython, pyRoleFrontmatter},
		{"node", runNode, jsRoleFrontmatter},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got struct {
				URL     string `json:"url"`
				Max     string `json:"max"`
				Missing any    `json:"missing"`
			}
			out := tc.run(t, tc.src)
			require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &got), "stdout: %q", out)

			require.Equal(t, "https://api.krisp.ai", got.URL)
			require.Equal(t, "3", got.Max, "trip2g stringifies note meta; values arrive as text")
			require.Nil(t, got.Missing, "a missing key must read as empty, not raise")
		})
	}
}
