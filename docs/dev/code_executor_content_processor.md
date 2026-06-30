# Code Executor — content processor (`executor:code`)

`executor:code` is a fleet role kind that extracts fenced code blocks from a note's body, runs each block in a sandboxed child process, and writes the combined stdout/stderr back to the note (or a separate log note).

## How it works

1. **Trigger**: the role fires when the note matches `path_patterns` (glob) and contains at least one fenced code block whose language label is known to the interpreter registry.
2. **Extraction**: `extractAllFencedBlocks(body)` scans the Markdown body for triple-backtick blocks with a language label (```` ```python … ``` ````). All blocks are returned in document order; v1 runs only the first.
3. **Sandboxing**: each block is written to a temp file (`/tmp/trip2g-coderun-*<ext>`); a child process is spawned via `exec.CommandContext` with the interpreter binary and the file path.
4. **Output cap**: stdout is capped at `MaxStdoutBytes` (default 1 MiB, configurable via `--max-stdout-bytes` or `fleet.Config.MaxStdoutBytes`). Bytes beyond the cap are silently discarded.
5. **Result**: stdout, stderr, and exit code are returned to the handler, which writes them back to the note (or a log target defined by `write_patterns`).

## Role YAML

```yaml
roles:
  - name: run-scripts
    kind: executor:code
    path_patterns:
      - scripts/**
    fence_lang: python
    allowed_programs:
      - python3
    write_patterns:
      - logs/**
```

`write_patterns` may be empty for read-only or side-effect-only roles.

## Interpreter registry

The default registry is embedded at build time from `internal/agentruntime/interpreters.json`. You can override it at startup:

```
fleet --interpreters /etc/fleet/interpreters.json …
```

Or from code: `agentruntime.LoadInterpretersFile(path)` / `agentruntime.SetInterpretersJSON(data)`.

Each entry:

| Field | Description |
|-------|-------------|
| `name` | Internal name; must match `fence_lang` in role YAML |
| `cmd` | argv[0…] prefix; file path is appended as last arg |
| `code_block_labels` | Markdown fence labels that map to this interpreter |
| `ext` | Temp-file extension (e.g. `.py`) |

## Security model

- Only programs listed in `allowed_programs` may be spawned (checked in `buildInvokers`).
- There is no network isolation or filesystem namespace (yet).
- `write_patterns` scopes which notes the role may update; an empty list means the role cannot update any note (side-effect-only or log-only).
- Future MCP tools will use the same allowlist mechanism — never add tools unconditionally (see comment in `buildInvokers`).

## Extending with new languages

Add an entry to `interpreters.json` (or a custom override file):

```json
{"name":"lua","cmd":["lua"],"code_block_labels":["lua"],"ext":".lua"}
```

No code changes needed.
