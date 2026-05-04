# Spike: Throwaway Prototypes for Template Engine Features

A **spike** is a standalone, single-file Go program used to validate a template engine idea before committing to the main codebase. It lives in `../trip2g_jet_blocks/` and reproduces only the relevant subsystem — no database, no HTTP server, just the Jet template mechanics.

## When to Use a Spike

Before implementing a new feature in `internal/layoutloader/`, always write a spike first:

- Adding new placeholder syntax (e.g., `@lid`, `@did`)
- Designing new block collection mechanisms (e.g., `yield_blocks` transitive CSS)
- Testing new asset patterns (e.g., `_js_` blocks, auto-import strategies)
- Validating edge cases before landing implementation code

If the spike doesn't work, neither will the real implementation.

## Where: `../trip2g_jet_blocks/main.go`

The spike is a minimal single-file Go program (~300 lines) that:

1. Sets up a Jet template loader with test fixtures
2. Registers template files (HTML, CSS blocks, JS blocks)
3. Loads and executes templates
4. Prints debug output and collected results

No external dependencies except Jet and Go stdlib. No build steps, no docker, no config files.

Example structure:

```go
package main

import (
	"fmt"
	"github.com/CloudyKit/jet/v6"
)

type autoLoader struct {
	// minimal loader implementation
}

func main() {
	// Load templates from testdata/
	// Execute and print output
	// Verify the idea works
}
```

## How to Run

```bash
cd ../trip2g_jet_blocks
go run main.go
```

Output is printed to stdout: debug logs, collected CSS/JS, rendered HTML.

## Workflow: Spike → Implementation → Test → Ship

### Step 1: Write the spike

Design the feature in isolation. Test edge cases. Print debug output to verify assumptions.

Example: validating that transitive block dependencies are collected correctly:

```go
// Spike: load page.html, which yields comp_nav, which yields comp_button
// Expected: CSS for nav AND button both collected
// Actual: print collected CSS, verify both are present
```

### Step 2: Verify output

Run the spike and inspect the results. If wrong, iterate on the spike code — don't commit to the main codebase yet.

```bash
go run main.go 2>&1 | tee spike-output.txt
# Review spike-output.txt
```

### Step 3: Port to `internal/layoutloader/`

Once the spike works, implement the same logic in the real code. Copy patterns, not code — the spike was throwaway.

Files to modify:
- `loader.go` — main loader logic
- `auto_import.go` — auto-discovery for siblings
- `yield_blocks.go` — block collection and emission
- `registry.go` — block name → file lookup

### Step 4: Write regression test

Add a test case to `blocks_inline_test.go` that validates the feature. Use the spike's test fixtures as `testdata/blocks_inline/*`:

```go
func TestYieldBlocks_MyNewFeature(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/page.html", Path: "testdata/blocks_inline/page.html", Content: readFixture(t, "...")},
		{ID: "/comp.html", Path: "testdata/blocks_inline/comp.html", Content: readFixture(t, "...")},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/page.html")
	require.Contains(t, out, ".expected-css")
}
```

### Step 5: Delete the spike (or keep as reference)

Once the feature lands in the main codebase and tests pass, you can:

- **Delete** `main.go` — spike is disposable by design
- **Keep** the `testdata/` templates — they become regression test fixtures

## Key Rule

**If you need to change `internal/layoutloader/`, spike first.**

The spike is insurance: it isolates the core idea from the app complexity. Problems found during spike validation are cheap to fix. Problems found after merging to main are expensive.

## Example: `yield_blocks` Feature

This project's `yield_blocks` system was validated via spike before implementation:

1. **Spike validated**: block discovery by file prefix, transitive dependency following, CSS/JS collection deduplication
2. **Implementation**: ported patterns to `loader.go`, `auto_import.go`, `yield_blocks.go`
3. **Regression tests**: `blocks_inline_test.go` tests prefix patterns, transitive deps, deduplication
4. **Shipped**: feature is in production

The spike proved the core BFS yield-following algorithm before committing to the full loader refactor.

## Testing the Spike Itself

The spike is not a unit test — it's an executable design document. Testing happens via:

1. **Eye-balling the output** — does it look right?
2. **Debug prints** — trace the algorithm, verify assumptions
3. **Fixture inspection** — manually check `testdata/` files
4. **Manual execution** — run `go run main.go` and read stdout

Once the feature works, formalize it with proper unit tests in `blocks_inline_test.go`.
