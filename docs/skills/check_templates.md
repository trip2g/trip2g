# Skill: Check Template Rendering

Use this to verify a layout renders without errors and inspect template variable values.

## How it works

`/_system/renderlayout` compiles and executes a Jet template server-side against a real note from the vault. It returns a `previewURL` — a direct link to the rendered HTML. No sync needed: share this link with the user while they're actively editing, it's faster than waiting for vault sync.

## Tool: scripts/renderlayout.py

The repo ships `scripts/renderlayout.py` — a CLI wrapper that reads the API key automatically from `.obsidian/plugins/trip2g/data.json` (walks up from cwd).

```bash
# run from vault dir (where .obsidian/ lives), e.g. docs/
# smoke test — no files needed
python3 ../scripts/renderlayout.py \
  --layout-src "{{ note.HTMLString() }}" --layout-path "/_debug.html" \
  --note-src "hello"
# → http://localhost:8081/_system/renderlayout?preview_id=abc123

# test a layout file against a real note
python3 ../scripts/renderlayout.py \
  --layout-file _layouts/mesh/index.html \
  --note-path /en/user/templates

# fetch rendered HTML directly
python3 ../scripts/renderlayout.py \
  --layout-src "{{ note.M().Debug() }}" --layout-path "/_debug.html" \
  --note-path /demo/template_meta_test \
  --fetch
```

Warnings and errors go to stderr. Exit code 1 on failure.
Share the printed URL with the user — they can open it in browser.

## Inspect variables with debug()

```jet
{{ debug(note.M()) }}
{* → *templateviews.Meta: &{raw:map[extra_content:[channels prices]]}
      methods: [Debug Get GetBool GetInt GetString GetStrings Has Raw] *}

{{ note.M().Debug() }}
{* → {"extra_content":["channels","prices"],"title":"Sales"} *}
```

> Always use parentheses when passing methods to `debug()`:
> `{{ debug(note.Title()) }}` ✓ → `string: My Title`
> `{{ debug(note.Title) }}` ✗ → `func() string: 0x...` (method reference, not value)

## Rendering a single BEM component

To preview one component in isolation, explicitly import its file and yield the block.
See [[en/user/bem]] for BEM conventions and `@lid`/`@did` naming.

**Important caveats:**

- Import paths must be **relative** to the layout path (after `/_layouts/` prefix is stripped).
  From `/_layouts/mesh/_preview.html` → import `"bar"` resolves to `/mesh/bar` ✓
- `yield_blocks()` returns **empty** in preview (wire phase is skipped). Yield style blocks directly instead.
- Dependencies must be imported manually — autoimport doesn't work in preview.

```bash
# Render mesh/bar.html component with its styles
python3 ../scripts/renderlayout.py \
  --layout-path "/_layouts/mesh/_preview.html" \
  --layout-src '{{ import "_blocks" }}{{ import "bar" }}{{ import "button" }}<style>{{ yield _style_mesh_bar() }}{{ yield _style_mesh_button() }}</style>{{ yield mesh_bar() }}' \
  --note-src "hello"
```

Pattern:
1. `{{ import "_blocks" }}` — shared layout wrapper (if needed)
2. `{{ import "component" }}` — the component file (and its dependencies)
3. `<style>{{ yield _style_component() }}</style>` — inline CSS (instead of `yield_blocks`)
4. `{{ yield component() }}` — the HTML block

**Tip:** `/_system/renderlayout` without params always serves the **latest** render — keep it open in a browser while iterating.

## Jet range gotcha

Single-variable range gives the **index** (0, 1, 2…), not the value:

```jet
{* WRONG — item = 0, 1, 2 *}
{{ range item := note.M().GetStrings("extra_content") }}{{ item }}{{ end }}

{* CORRECT *}
{{ range i, item := note.M().GetStrings("extra_content") }}{{ item }}{{ end }}
```

## scripts/renderlayout.py source

```python
#!/usr/bin/env python3
"""
renderlayout — CLI wrapper for /_system/renderlayout.

Reads API key and URL from .obsidian/plugins/trip2g/data.json
(walks up from cwd) or from TRIP2G_API_KEY / TRIP2G_API_URL env vars.
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
from pathlib import Path


def find_config() -> dict:
    current = Path.cwd()
    for directory in [current, *current.parents]:
        candidate = directory / ".obsidian" / "plugins" / "trip2g" / "data.json"
        if candidate.exists():
            with open(candidate) as f:
                data = json.load(f)
            sync_dirs = data.get("syncDirs", [])
            if sync_dirs:
                return {
                    "api_key": sync_dirs[0].get("apiKey", ""),
                    "api_url": sync_dirs[0].get("apiUrl", "http://localhost:8081"),
                }
    return {}


def main():
    parser = argparse.ArgumentParser(description="Render a Jet layout template")
    parser.add_argument("--layout-path", help="Layout path on server, e.g. /_layouts/foo.html")
    parser.add_argument("--layout-file", help="Read layout src from this local file")
    parser.add_argument("--layout-src",  help="Inline layout template source")
    parser.add_argument("--note-path",   help="Note path on server, e.g. /my-note")
    parser.add_argument("--note-file",   help="Read note markdown from this local file")
    parser.add_argument("--note-src",    help="Inline note markdown source")
    parser.add_argument("--api-key",     help="API key (overrides config)")
    parser.add_argument("--api-url",     help="Server URL (overrides config)")
    parser.add_argument("--fetch", action="store_true", help="Fetch and print rendered HTML")
    args = parser.parse_args()

    config = find_config()
    api_key = args.api_key or os.environ.get("TRIP2G_API_KEY") or config.get("api_key", "")
    api_url = (args.api_url or os.environ.get("TRIP2G_API_URL") or config.get("api_url", "http://localhost:8081")).rstrip("/")

    if not api_key:
        print("Error: no API key found. Set TRIP2G_API_KEY or --api-key.", file=sys.stderr)
        sys.exit(1)

    layout_path = args.layout_path
    if args.layout_file:
        layout_path = layout_path or ("/" + args.layout_file.lstrip("/"))
        with open(args.layout_file) as f:
            layout_src = f.read()
    else:
        layout_src = args.layout_src

    if not layout_path:
        print("Error: --layout-path or --layout-file is required.", file=sys.stderr)
        sys.exit(1)

    layout = {"path": layout_path}
    if layout_src is not None:
        layout["src"] = layout_src

    note = None
    if args.note_path:
        note = {"path": args.note_path}
    elif args.note_file:
        with open(args.note_file) as f:
            note = {"src": f.read()}
    elif args.note_src is not None:
        note = {"src": args.note_src}

    payload = {"layout": layout}
    if note is not None:
        payload["note"] = note

    body = json.dumps(payload).encode()
    req = urllib.request.Request(
        f"{api_url}/_system/renderlayout",
        data=body,
        headers={"Content-Type": "application/json", "X-API-Key": api_key},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        result = json.loads(e.read())

    warnings = result.get("warnings", {}).get("layout", [])
    if warnings:
        print("WARNINGS:", file=sys.stderr)
        for w in warnings:
            print(" ", w, file=sys.stderr)

    if "error" in result:
        print(f"ERROR: {result['error']}", file=sys.stderr)
        sys.exit(1)

    full_url = f"{api_url}{result.get('previewURL', '')}"

    if args.fetch:
        with urllib.request.urlopen(urllib.request.Request(full_url)) as resp:
            print(resp.read().decode())
    else:
        print(full_url)
        if warnings:
            sys.exit(1)


if __name__ == "__main__":
    main()
```
