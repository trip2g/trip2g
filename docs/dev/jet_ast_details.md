# Jet v6 AST & Runtime: Key Findings

Findings from a spike (`../trip2g_jet_blocks`) validating the BEM blocks / `yield_blocks` mechanism.
Jet version: `github.com/CloudyKit/jet/v6 v6.3.1`.

## Block Registration

Blocks are registered at **parse time**, not execution time.

```go
// parse.go:417
t.passedBlocks[block.Name] = block
```

Every `{{block name()}}...{{end}}` encountered during parsing is added to `passedBlocks`
regardless of where it appears — including inside `{{if}}`, `{{range}}`, or nested blocks.
At the end of parsing, `addBlocks` merges `passedBlocks` into `processedBlocks`:

```go
// parse.go:242
t.addBlocks(t.passedBlocks)
```

`processedBlocks` is the map that `YieldBlock` and `getBlock` read from at runtime.

**Consequence:** blocks inside `{{if false}}` ARE registered in `processedBlocks`. This is
the mechanism used to suppress default inline render while keeping blocks available for
`YieldBlock`.

## Include Does NOT Share Blocks with Parent

`{{include "file.html"}}` temporarily replaces the current scope's block map with the
included template's `processedBlocks`:

```go
// eval.go:~630
st.newScope()
defer st.releaseScope()
st.blocks = t.processedBlocks
```

After the include finishes, `releaseScope()` restores the parent's block map. Blocks
defined in the included file are NOT available for `{{yield}}` in the parent template.

**Solution:** inline component file content directly into the page template's source
(string concatenation) so all block definitions end up in the same parsed template's
`processedBlocks`.

## Suppressing Default Block Render

`{{block name()}}...{{end}}` always renders its content inline when executed:

```go
// eval.go:NodeBlock case
block, has := st.getBlock(node.Name)
if has == false {
    block = node
}
st.executeYieldBlock(block, ...)
```

To suppress inline render while keeping blocks registered:

```
{{if false}}
  ... component file content ...
{{end}}
```

- Parse time: `addBlocks` walks full AST including inside `{{if}}` → blocks registered ✓
- Execute time: `{{if false}}` branch never taken → no inline render ✓
- `YieldBlock("name", nil)` still works because blocks are in `processedBlocks` ✓

## YieldBlock

`Runtime.YieldBlock(name string, context interface{})` is exported and writes directly
to the current `st.Writer`:

```go
// eval.go:112
func (st *Runtime) YieldBlock(name string, context interface{}) {
    block, has := st.getBlock(name)
    if !has {
        panic(fmt.Errorf("Block %q was not found!!", name))
    }
    // ...
    st.executeList(block.List)
}
```

- Panics if block not found → always wrap in `recover` when block availability is uncertain
- Writes directly to current writer (correct behavior when called from a global func)
- Can be called from inside a `jet.Func` via `a.Runtime().YieldBlock(name, nil)`

## Global Functions (jet.Func)

`jet.Func` signature: `func(Arguments) reflect.Value`. No body capture.

```go
// func.go:204
type Func func(Arguments) reflect.Value
```

`jet.Arguments` exposes: `Get`, `IsSet`, `ParseInto`, `NumOfArguments`,
`RequireNumOfArguments`, `Runtime`, `Panicf`. No `Body`, `Writer`, or block execution.

**Consequence:** `{{funcname "arg"}}...content...{{end}}` is NOT valid — `{{end}}` is
an orphan and causes a parse error. Functions cannot capture template body content.

Global functions access the current writer via `a.Runtime().Writer` (exported field).

## Scope and Block Map

`newScope()` creates a child scope but shares the **same** `blocks` map reference:

```go
// eval.go:88
func (st *Runtime) newScope() {
    st.scope = &scope{parent: st.scope, variables: make(VarMap), blocks: st.blocks}
}
```

Variables are scoped; blocks are template-global. There is no per-yield-invocation block
scoping. `scope.sortedBlocks()` is **unexported** — block name iteration requires a
load-time registry.

## Template Extensions

Blocks from extended/imported templates are merged into `processedBlocks` during parsing:

```go
// parse.go:235-242
if t.extends != nil {
    t.addBlocks(t.extends.processedBlocks)
}
for _, _import := range t.imports {
    t.addBlocks(_import.processedBlocks)
}
t.addBlocks(t.passedBlocks)
```

This is the `{{extends}}` inheritance mechanism. `{{include}}` does NOT trigger this merge.

## Proven Mechanism: BEM Blocks Auto-Wiring

The following mechanism was validated in `../trip2g_jet_blocks/main.go`:

1. **Build block registry** at load time: walk all component files, map `blockName → fileID`.

2. **Parse page template** (alone, no preamble) using a temporary `jet.Set`. Walk AST for
   `YieldNode` to collect all `{{yield name()}}` calls.

3. **BFS over yield graph**: for each yielded block, look up its file in the registry; walk
   that file for its yields; repeat until no new files are found. Handles transitive deps
   (`page → card → button`) and detects cycles via a `visited` set.

4. **Build combined template**: wrap all needed component file contents in `{{if false}}`
   (suppress default render) + original page content. Store temporarily; restore after parse.

5. **Register `yield_blocks` global func** on the `jet.Set`, closed over the list of block
   names from inlined files. At render time: iterate names, match against pattern
   (prefix or `/regexp/`), call `rt.YieldBlock(name, nil)` for each match.

6. **Execute** the combined template normally. `{{yield hero()}}` resolves because `hero`
   is in `processedBlocks`. `{{yield_blocks "_style_"}}` emits CSS for all `_style_*` blocks
   from the inlined components.

## Caveats

- Load-time cost: each page load parses component files once for registry + once for
  dependency walk. Cache the registry and dependency graph across requests.
- Component files must only define `{{block}}` nodes; they must not `{{yield}}` at top
  level (only inside block definitions), otherwise top-level yields in the preamble would
  execute during `{{if false}}` — but `{{if false}}` prevents this.
- `YieldBlock` panics on missing block; always use `recover` when calling with names from
  an external source.
- `scope.sortedBlocks()` is unexported; block name iteration must use the load-time
  registry, not runtime scope introspection.
