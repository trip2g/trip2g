# Frontend testing ($mol)

> **TL;DR.** Put unit tests in `*.test.ts` next to the code, declare them with
> `$mol_test({ 'name'() { … } })`, assert with `$mol_assert_equal(actual, expected)`.
> Run them with `cd ../mam && node node_modules/.bin/mam trip2g/admin` (exit `0` = all
> green, `135` = a test or type error). Tests run in **Node**: pure functions only — no
> DOM, no `localStorage`, no network. So unit-test *extracted pure logic*
> (e.g. `$trip2g_editor_merge`), not reactive `@$mol_mem` view code.

## Where tests live

A test file sits next to the code it covers and ends in `.test.ts`:

```
assets/ui/editor/merge/merge.ts        ← code  ($trip2g_editor_merge)
assets/ui/editor/merge/merge.test.ts   ← tests
```

| Suffix | Runs in |
|--------|---------|
| `*.test.ts` | Node **and** browser |
| `*.node.test.ts` | Node only |
| `*.web.test.ts` | Browser only (needs DOM) |

Tests are discovered automatically — any `*.test.ts` under the built module tree is
bundled and executed. There is no per-file selection; it is all-or-nothing per app.

## Writing a test

```ts
namespace $ {
	$mol_test({
		'identical content has no markers'() {
			$mol_assert_equal(
				$trip2g_editor_merge('a\nb\nc', 'a\nb\nc'),
				'a\nb\nc',
			)
		},
	})
}
```

Each key is the test name, each value the test body. See
`assets/ui/editor/merge/merge.test.ts` for a real, multi-case example.

### Assertions

From `mam/mol/assert/assert.ts`:

| Helper | Meaning |
|--------|---------|
| `$mol_assert_equal(a, b, …)` | all arguments are **structurally** equal (the one to reach for) |
| `$mol_assert_unique(a, b, …)` | all arguments are **not** equal to each other |
| `$mol_assert_ok(x)` / `$mol_assert_not(x)` | truthy / falsy (deprecated — prefer `_equal`) |
| `$mol_assert_fail(() => …, ErrorOrMessage)` | the callback must throw; optionally match a message string or `Error` subclass |

`$mol_assert_equal` deep-compares strings, arrays and objects, so you can assert whole
structures, not just primitives.

## Running tests

The same command that type-checks the admin app also **runs the Node tests**:

```bash
cd /home/alexes/projects2/mam
node node_modules/.bin/mam trip2g/admin       # exit 0 = compiles + all tests pass; 135 = failure
```

Under the hood it bundles everything into `trip2g/admin/-/node.test.js` and runs it.
To run just that bundle (after a build) and see failures:

```bash
node --enable-source-maps --trace-uncaught trip2g/admin/-/node.test.js
```

A passing run is quiet; a failing assertion prints the diff and the process exits non-zero.

## What runs in a test (and what doesn't)

Node tests run in an **isolated context** (`mam/mol/test`):

- `Math.random()` is **seeded** (deterministic) — don't rely on real randomness.
- `fetch` and `XMLHttpRequest` are **forbidden** — accessing them throws. Tests cannot
  hit the backend or GraphQL.
- There is **no DOM and no `localStorage`** (`$mol_state_local`).
- Each test has a **1 s timeout**; for async, return a `Promise`.

## What to test here

Because of those limits, the unit-testable surface is **pure, deterministic logic**.
The pattern: when a view needs non-trivial logic, **extract it into a plain function**
(its own module) and test that — not the `@$mol_mem` view container.

- ✅ Good: `$trip2g_editor_merge(mine, theirs)` — pure string→string LCS merge → `merge.test.ts`.
- ❌ Hard: a `@$mol_mem` getter that reads `$mol_state_local` / GraphQL / DOM — reactive
  and side-effectful; cover it in Playwright e2e instead (see `TESTING.md`).

Good next candidates (pure or easily extractable):
- `wikilink_at(text, offset)` in `editor/pane/pane.view.ts` — regex wikilink detection
  (extract to a helper first, like merge).
- The self-echo baseline comparison (`versionId <= baseline`) — pure once isolated.

## See also

- `TESTING.md` — end-to-end testing with Playwright (the other half: real backend, DOM, network).
- `mam/mol/test/`, `mam/mol/assert/` — the framework source and its own tests.
