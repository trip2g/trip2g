# Skill: Check Template Rendering

Use this to verify a layout renders without errors and inspect template variable values.

## How it works

`/_system/renderlayout` compiles and executes a Jet template server-side against a real note from the vault. It returns a `previewURL` — a direct link to the rendered HTML. No sync needed: you can share this link with the user while they're actively editing, it's faster than waiting for vault sync.

## API key

Read from `.obsidian/plugins/trip2g/data.json`:

```bash
API_KEY=$(cat .obsidian/plugins/trip2g/data.json | python3 -c "import json,sys; print(json.load(sys.stdin)['syncDirs'][0]['apiKey'])")
API_URL=$(cat .obsidian/plugins/trip2g/data.json | python3 -c "import json,sys; print(json.load(sys.stdin)['syncDirs'][0]['apiUrl'])")
```

## Check: no errors + preview link

```bash
curl -s -X POST $API_URL/_system/renderlayout \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "layout": { "path": "/_layouts/iiworker/landing.html" },
    "note":   { "path": "/demo/simple" }
  }' | python3 -m json.tool
```

Success: `warnings.layout` is empty, `previewURL` is set. Share `$API_URL + previewURL` with the user.

Errors appear in `warnings.layout`:
```json
{ "warnings": { "layout": ["runtime: Jet Runtime Error..."] } }
```

## Inspect variables with inline template

Override the template source to inspect any value without editing files:

```bash
curl -s -X POST $API_URL/_system/renderlayout \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "layout": { "path": "/_debug.html", "src": "{{ debug(note.M()) }}" },
    "note":   { "path": "/demo/template_meta_test" }
  }' | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('error') or d['previewURL'])" \
  | xargs -I{} curl -s "$API_URL{}"
```

Output example:
```
*templateviews.Meta: &{raw:map[author:John Doe featured:true title:Meta Test Page]}
methods: [Debug Get GetBool GetInt GetString GetStrings Has Raw]
```

> **Note:** always use parentheses when passing method results to `debug()`.
> `{{ debug(note.Title()) }}` → `string: My Title` ✓
> `{{ debug(note.Title) }}` → `func() string: 0x...` (passes method reference, not value) ✗

`note.M().Debug()` gives compact JSON of all frontmatter keys:
```
{"author":"John Doe","featured":true,"title":"Meta Test Page"}
```

## Jet range gotcha

Single-variable range gives the **index**, not the value:

```jet
{* WRONG — item = 0, 1, 2 *}
{{ range item := note.M().GetStrings("extra_content") }}{{ item }}{{ end }}

{* CORRECT *}
{{ range i, item := note.M().GetStrings("extra_content") }}{{ item }}{{ end }}
```
