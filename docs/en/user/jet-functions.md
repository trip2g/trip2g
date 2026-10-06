---
title: Jet functions reference
free: true
lang: en
lang_redirect: "[[ru/user/jet-functions]]"
---

Every function, filter and construct you can use in a custom Jet layout in `_layouts/`, checked against the Jet version trip2g ships (CloudyKit/jet v6.3.1) and the trip2g renderer.

The page has two parts. The first is the full reference for **Jet itself**: built-in functions, escaping, and language constructs. The second is an **index of what trip2g adds** (`note`, `nvs`, `asset()` and the rest), with a link to the page that documents each item.

New to custom layouts? Start with [[en/user/templates|Templates]], then build pages from components as [[en/user/components|Template components]] recommends. When something renders wrong, see [[en/user/jet-debugging|Debugging Jet templates]].

### What Jet is like

Jet is a template engine for Go. Its syntax is close to Go's `text/template`, with features borrowed from Jinja2 and Twig: layout inheritance with `extends` and blocks, filters written with `|`, and a ternary operator. If you know either, read Jet as one of them with the differences below.

Like Go templates: `{{ … }}` delimiters, `if` / `else if` / `else` / `end`, `range` over lists and maps with an `else` branch for an empty collection, and `if` that declares a variable.

Unlike Go templates:

| | Go `text/template` | Jet |
|---|---|---|
| Variables | `$x := .Title` | `x := note.Title()`, no `$` |
| Methods | `.Title`, called automatically | `note.Title()`, parentheses required |
| Operators | functions: `eq`, `and`, `not` | `==`, `&&`, `!`, `+`, `cond ? a : b` |
| `if` with a declaration | `if $x := f`, tests `$x` | `if x := f(); cond`, any condition |
| `range` with one variable | gives the element | gives the **index**; write `range i, x :=` |
| Pipe `a \| f b` | `a` becomes the **last** argument | `a` becomes the **first** argument |
| Reusable fragment | `define` + `template "name" .` | `block name(p="")` + `yield name(p=…)` |
| Layout inheritance | no `extends` | `extends`; the page overrides blocks |
| Comments | `{{/* … */}}` | `{* … *}` |
| Escaping | `html/template` escapes by context | the same HTML escaping everywhere |

Like Jinja2 and Twig: `extends` with blocks that a page overrides, `import` and `include`, filters with arguments (`"a-b" | replace: "-", " ", -1`), and `cond ? a : b` as in Twig. Unlike them, every tag uses `{{ }}` (there is no `{% %}`), loops are `range`, not `for x in list`, and a block takes named parameters like a Jinja macro.

Escaping doesn't look at context. Inside `<script>` use `json` or `safeJs`, described below.

### Output is escaped by default

Read this first. `{{ value }}` HTML-escapes the value, so a title like `Q&A <draft>` reaches the page as text, not as tags:

```jet
{{ "<b>bold</b>" }}            {* → &lt;b&gt;bold&lt;/b&gt; *}
{{ "<b>bold</b>" | unsafe }}   {* → <b>bold</b> *}
```

What follows from this:

- Methods and fields that return HTML the server built print as is, without a filter: `note.HTMLString()`, `FirstListHTML()`, `FormSpecJSON()`, `SubgraphNamesJSON()`, section `TitleHTML` / `ContentHTML`, code block `HTML`, the `defaultTemplate.*` functions and `asset()`.
- `| unsafe` and `| raw` print a plain string without escaping. You need them only for markup the template builds itself or keeps in a string, and for `injection.Content` of the site's [[en/user/templates#HTML injections|HTML injections]].
- `| html` is no longer needed for text. It escapes once and marks the result as safe, so `{{ title | html }}` in an older template prints the same as `{{ title }}`, not a double-escaped `&amp;amp;`.
- A string function or `+` applied to safe output returns a plain string, and that string is escaped: `{{ upper(note.HTMLString()) }}` prints the tags as text.

The [[#Escaping]] section below compares the filters.

### Jet built-in functions

| Function | Returns |
|---|---|
| `len(x)` | Length of a string (in bytes), slice, array or map; number of fields of a struct |
| `isset(a, b, …)` | `true` if every argument is defined and not nil |
| `lower(s)`, `upper(s)` | The string in lower / upper case |
| `hasPrefix(s, prefix)`, `hasSuffix(s, suffix)` | `bool` |
| `trimSpace(s)` | The string without leading and trailing whitespace |
| `repeat(s, n)` | `s` repeated `n` times |
| `replace(s, old, new, n)` | `s` with the first `n` matches replaced; `n = -1` replaces all |
| `split(s, sep)` | `[]string` |
| `map(k1, v1, k2, v2, …)` | `map[string]interface{}` |
| `slice(a, b, …)`, `array(a, b, …)` | `[]interface{}` |
| `ints(from, to)` | A rangeable sequence `from … to-1` |
| `json(v)` | JSON of `v`, compact, printed without HTML escaping |
| `writeJson(v)` | Writes JSON of `v` to the output, followed by a newline |
| `includeIfExists(path, ctx?)` | Renders the template if it exists; returns a hidden `true`/`false` |
| `exec(path, ctx?)` | The value another layout file `return`s; its output is discarded |
| `dump()`, `dump("name")` | A text dump of the context, variables and globals in scope |

Escaping filters (`html`, `url`, `safeHtml`, `safeJs`, `raw`, `unsafe`) have [[#Escaping|their own table]].

#### Calling a function: three forms

A function can be called directly, or applied with a pipe. In a pipe the piped value becomes the **first** argument, and the rest go after a colon:

```jet
{{ upper("hello") }}                {* → HELLO *}
{{ "hello" | upper }}               {* → HELLO *}
{{ "a-b-c" | replace: "-", " ", -1 }}
{* replace("a-b-c", "-", " ", -1) → a b c *}
{{ "Hello" | hasPrefix: "He" }}     {* → true *}
{{ "a-b" | replace: "-", " ", -1 | upper }}
{* pipes chain → A B *}
```

`{{ 3 | repeat: "ab" }}` fails, because it calls `repeat(3, "ab")`. Write `repeat("ab", 3)`.

#### Strings

```jet
{{ lower("ABC") }}                  {* → abc *}
{{ trimSpace("  x  ") }}            {* → x *}
{{ repeat("ab", 3) }}               {* → ababab *}
{{ replace("aaa", "a", "b", 2) }}   {* → bba *}

{{ range i, part := split("a,b,c", ",") }}
  [{{ part }}]
{{ end }}
{* → [a] [b] [c] *}

{{ if hasSuffix(note.Permalink(), "/faq") }}
  <a href="/faq/contact">Ask a question</a>
{{ end }}
```

`len` counts bytes, not letters: `len("héllo")` is `6`.

#### `isset`

```jet
{{ m := map("a", 1) }}
{{ isset(m["a"]) }}        {* → true *}
{{ isset(m["z"]) }}        {* → false: missing map key *}
{{ isset(undefinedVar) }}  {* → false, no error *}
```

`isset` is the one place where an undefined variable is not an error. Everywhere else `{{ undefinedVar }}` stops the render with `identifier "undefinedVar" not available`.

#### `map`, `slice`, `ints`

```jet
{{ links := map("docs", "/docs", "blog", "/blog") }}
{{ links["blog"] }}  {{ links.blog }}   {* both → /blog *}

{{ range i, n := ints(1, 4) }}
  {{ n }}
{{ end }}
{* → 1 2 3 *}
```

- `map` keys must be strings, and the argument count must be even.
- `ints(from, to)` excludes `to`, and `from` must be smaller than `to`. `ints(3, 1)` is a render error.
- A plain number is not rangeable: `{{ range i := 3 }}` fails. Use `ints(0, 3)`.

#### Index and slice

`list[from:to]` takes the elements from `from` up to, but not including, `to`. Either bound can be left out, and both can be variables or expressions:

```jet
{{ colors := slice("red", "green", "blue", "black") }}
{{ colors[1] }}             {* → green *}
{{ colors[1:3] }}           {* → [green blue] *}

{{ from := 1 }}
{{ to := len(colors) - 1 }}
{{ colors[from:to] }}       {* → [green blue] *}
{{ colors[from:] }}         {* → [green blue black] *}
{{ colors[:to] }}           {* → [red green blue] *}
```

Strings slice the same way, by bytes: `{{ word := "hello" }}{{ word[1:4] }}` prints `ell`. A string literal can't be sliced directly: `"hello"[1:4]` doesn't load.

Put spaces around `-` in an index: `colors[len(colors) - 2:]` works, `colors[len(colors)-2:]` doesn't load, because Jet reads `-2` as a number.

#### `json` and `writeJson`

```jet
<script>
  window.pageData = {{ json(map(
    "title", note.Title(),
    "tags", note.Tags()
  )) }};
</script>
```

```jet
<script type="application/json" id="data">
  {{ writeJson(note.M().Raw()) }}
</script>
```

Both escape `<`, `>` and `&` as `<`… so the output is safe inside `<script>`, and the page's escaping leaves it alone. `writeJson` adds a trailing newline. `json` does not.

A tag can span several lines inside parentheses, as the `json(map(…))` call above does: break a long argument list after a comma. A chain of method calls can't break before a `.`. Split a long chain into variables instead (see the [[#Worked example: latest posts with a count|worked example]]).

#### `includeIfExists` and `exec`

```jet
{{ includeIfExists("partials/banner") }}

{{ if !includeIfExists("partials/sidebar", note) }}
  <p>No sidebar</p>
{{ end }}
```

`includeIfExists` renders the template when it exists and returns a boolean that prints nothing. The optional second argument becomes `.` inside the included template.

`exec("lib/featured", data)` runs another layout file, throws away what it prints and gives you the value of its `{{ return }}`. Inside that file `data` is `.`, and `note`, `nvs` and the other page variables work. The path starts at `_layouts/` and has **no extension**: `exec("lib/featured")` works, `exec("lib/featured.html")` fails with `template /lib/featured.html could not be found`. Use it for data (a list, a map, a number), and components for markup. Examples and path rules: [[en/user/templates#Data from another layout: exec and return|Templates: exec and return]].

#### `dump`

```jet
<pre>{{ dump() }}</pre>       {* context, variables in scope, globals, blocks *}
<pre>{{ dump("x") }}</pre>    {* one variable: "x:=5 // float64" *}
```

`dump` lists *names*. To see an object's type, value and methods, use trip2g's [[en/user/jet-debugging|debug()]] instead.

### Escaping

| Filter | Does | Use for |
|---|---|---|
| `html` | A function: escapes `<`, `>`, `&`, `'`, `"` once and returns a safe value | Showing HTML source as text; keeping an escaped value in a variable |
| `safeHtml` | The same escaping, written straight to the output | Nothing `{{ value }}` doesn't already do |
| `url` | A function: query escaping (`a b&c` → `a+b%26c`), returns a safe value | One query-string value: `?q={{ term \| url }}` |
| `safeJs` | JavaScript string escaping (`'` → `\'`, `<` → `<`) | A value inside a JS string literal |
| `raw` / `unsafe` | No escaping, written straight to the output | Markup the template builds itself or keeps in a plain string |

```jet
{{ s := "<b>Tom & Jerry</b>" }}
{{ s }}               {* → &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; *}
{{ s | html }}        {* the same *}
{{ s | safeHtml }}    {* the same *}
{{ s | unsafe }}      {* → <b>Tom & Jerry</b> *}
{{ s | raw }}         {* the same as unsafe *}
```

**`html` or `safeHtml`.** For text, neither: `{{ s }}` already escapes. They differ in what they are. `html` is a function that returns a value, so it works anywhere an expression does, and the result is not escaped a second time: `{{ e := html(s) }}` can be printed later, in text or in an attribute. `safeHtml`, `safeJs`, `raw` and `unsafe` are writers: they work only as the last filter of an output tag. `{{ t := s | unsafe }}` doesn't load, and `safeHtml(s)` fails at render. The one job `html` still has is showing markup as text: `{{ note.HTMLString() }}` renders the note, `{{ html(note.HTMLString()) }}` prints its tags so a reader sees them.

```jet
{{ e := html(s) }}
<p title="{{ e }}">{{ e }}</p>
{* both escaped once: &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; *}

<pre>{{ html(note.HTMLString()) }}</pre>
{* the note's HTML as visible tags *}
```

**`url`** escapes one query-string value. It escapes `/`, `:` and `&` too, so it is not for a whole URL. Like `html`, it is a function and its result is not escaped again:

```jet
{{ term := "a b&c/d" }}
<a href="/search?q={{ term | url }}">Search</a>
{* → href="/search?q=a+b%26c%2Fd" *}
```

**`raw` and `unsafe`** are the same filter. Use one only for markup you trust: the template's own strings, `injection.Content`. Never for text an author or a visitor wrote.

**`safeJs`** escapes a value for a JavaScript string literal:

```jet
<script>
  var title = '{{ note.Title() | safeJs }}';
</script>
```

### Language constructs

#### Output, comments, whitespace

```jet
{{ note.Title() }}                  {* print an expression *}
{* a comment, never rendered *}
<li>   {{- note.Title() -}}   </li>
{* {{- and -}} trim whitespace on that side *}
```

Call methods with parentheses. `{{ note.Title }}` without `()` prints a function address, not the title.

#### Variables: `:=` and `=`

```jet
{{ count := 0 }}                 {* declare *}
{{ count = count + 1 }}          {* assign to an existing variable *}
```

- `=` on a variable that was never declared is a render error: `could not assign "x" … variable "x" is uninitialised`.
- `:=` inside `if` or `range` declares a **new** variable that hides the outer one until `{{ end }}`. To change the outer value, use `=`:

```jet
{{ x := 1 }}{{ if true }}{{ x := 2 }}{{ end }}{{ x }}   {* → 1 *}
{{ x := 1 }}{{ if true }}{{ x = 2 }}{{ end }}{{ x }}    {* → 2 *}
```

#### Operators and values

| Kind | Operators |
|---|---|
| Arithmetic | `+ - * / %` |
| Comparison | `== != < > <= >=` |
| Logic | `&& \|\| !` |
| Ternary | `cond ? a : b` |
| Index / slice | `m["key"]`, `m.key`, `list[0]`, `list[1:3]`, `list[from:to]`, `str[1:3]` |

- Number literals are floats: `{{ 7 / 2 }}` prints `3.5`.
- `+` with a string concatenates: `{{ "page " + 2 }}` → `page 2`.
- `""`, `0`, `false` and `nil` are false in `if` and `?:`: `{{ s ? s : "none" }}`. An **empty list is true**, so test lists with `len(list) > 0`.
- `||` returns a boolean, not the first non-empty value. For a fallback value use trip2g's [[#Global functions|coalesce()]].

#### `if`

```jet
{{ if n == 1 }}
  one
{{ else if n == 2 }}
  two
{{ else }}
  many
{{ end }}

{{ if v, ok := links["blog"]; ok }}
  <a href="{{ v }}">Blog</a>
{{ end }}
```

#### `range`

```jet
{{ range i, tag := note.Tags() }}
  <span>{{ tag }}</span>
{{ end }}

{{ range key, value := links }}
  {{ key }} → {{ value }}
{{ end }}

{{ range i, post := posts }}
  <a href="{{ post.Permalink() }}">{{ post.Title() }}</a>
{{ else }}
  <p>Nothing here.</p>
{{ end }}
```

- **With one variable, `range` gives the index, not the value.** `{{ range tag := note.Tags() }}` yields `0, 1, 2…`. Always write `range i, value :=`.
- `{{ else }}` renders when the collection is empty.
- Use `_` for a variable you don't need: `range _, tag := note.Tags()`.

#### `block` and `yield`

```jet
{{ block card(title="", url="") }}
  <a class="card" href="{{ url }}">{{ title }}</a>
{{ end }}

{{ yield card(title="Docs", url="/docs") }}
{{ yield card(url="/blog", title="Blog") }}
```

- Pass arguments **by name**. A positional argument, `yield card("Docs")`, is ignored, and the parameter keeps its default.
- Give every parameter a default. A block defined in the page itself also renders where it is defined, with no arguments. A parameter without a default then fails with `missing name for block parameter`.
- `content` is reserved. Don't use it as a parameter name.

A block can wrap content. The block outputs the caller's content with `{{ yield content }}`:

```jet
{{ block panel(title="") }}
  <section>
    <h2>{{ title }}</h2>
    {{ yield content }}
  </section>
{{ end }}

{{ yield panel(title="Related") content }}
  <p>Anything here becomes the panel body.</p>
{{ end }}
```

A value after the call becomes `.` inside the block: `{{ yield menu() items }}`.

#### `import`, `include`, `extends`

```jet
{{ import "blocks" }}
{* load the blocks of _layouts/blocks.html, render nothing *}

{{ include "partials/footer" }}
{* render another template here *}

{{ include "partials/card" note }}
{* … with `note` as `.` inside it *}
```

`extends` makes a page inherit a base layout and fill its blocks:

```jet
{{ extends "base" }}

{{ block main() }}
  <p>This replaces the main block of base.html</p>
{{ end }}
```

- Paths are relative to `_layouts/`, without `.html`.
- `extends` must be the first tag in the file, then any `import`s.
- How a base and a page fit together, with a tested example: [[en/user/templates#Layout inheritance: extends|Templates: layout inheritance]].
- trip2g also imports blocks automatically when a page yields a block defined in another layout file. A template reached through `include` or `extends` doesn't get that and needs its own `{{ import }}`. See [[en/user/components#Auto-import|Template components: auto-import]].

#### `return` and `try` / `catch`

```jet
{* lib/featured.html: the value exec gets *}
{{ return nvs.ByGlob(. + "/*.md").Public().Limit(3).All() }}
```

```jet
{{ try }}
  {{ stats := exec("lib/stats", "blog") }}
  <p>{{ stats.count }} posts</p>
{{ catch err }}
  {{ if currentUser.IsAdmin() }}
    <p>{{ err.Error() }}</p>
  {{ end }}
{{ end }}
```

- `return` gives `exec` its value. In a page rendered normally it stops nothing and prints nothing.
- `try` renders its body; if anything inside fails, the body's output is dropped and `catch` renders instead. Without `catch` a failed `try` prints nothing. It doesn't catch a layout that fails to load.

Worked examples: [[en/user/templates#Data from another layout: exec and return|exec and return]], [[en/user/templates#Isolating failures: try and catch|try and catch]].

### What trip2g adds

trip2g registers eight global functions, passes a set of variables to every custom layout, and replaces two placeholders in layout source. Each item below links to the page that documents it in full. Short descriptions are given here for items that have no other page.

#### Global functions

| Function | Returns | Documented in |
|---|---|---|
| `asset("file.css")` | URL of a file next to the layout; the argument unchanged if no such asset exists | [[en/user/yield_blocks\|yield_blocks]], [[en/user/templates#Assets across layout files\|Templates]] |
| `debug(expr)` | Go type, value and method list of `expr` | [[en/user/jet-debugging\|Debugging Jet templates]] |
| `yield_blocks("prefix")` | Renders every block whose name starts with the prefix (or matches `/regex/`) | [[en/user/yield_blocks\|yield_blocks]] |
| `coalesce(a, b, …)` | The first argument that is set and non-empty; otherwise the last argument | Below |
| `arg_type("param", "type", "comment")` | Nothing; describes a block parameter for tooling | Below |
| `parseJSON(s)`, `parseYAML(s)`, `parseCSV(s)` | The text as maps and lists (CSV: a list of rows); `nil` on invalid input | [[en/user/templates#parse-data\|Templates: parsing data]] |

`coalesce` treats a missing map key, `nil`, `""`, an empty list and an empty map as empty. `0` and `false` count as values:

```jet
{{ videos := map("en", "intro-en.mp4") }}
{{ coalesce(videos[note.Lang()], videos["en"]) }}
{{ coalesce(
  note.M().GetString("subtitle", ""),
  note.Description(),
  note.Title()
) }}
```

`arg_type` renders nothing. trip2g reads it at load time to describe a block's parameters (type and comment) to tools that list blocks:

```jet
{{ block hero(title="", image="") }}
  {{ arg_type("title", "string", "Heading text") }}
  {{ arg_type("image", "asset", "Background image file") }}
  …
{{ end }}
```

#### Variables in every custom layout

| Variable | What it is | Documented in |
|---|---|---|
| `note` | The page being rendered | Below |
| `nvs` | All notes of the site, for lookups and queries | Below |
| `title` | The page title after the site title template is applied, ready for `<title>` | Below |
| `publicURL` | The main domain's address, for example `https://example.com` | [[en/user/templates#SEO tags in a custom layout\|Templates]] |
| `htmlInjectionsHead`, `htmlInjectionsBodyEnd` | The snippets an admin adds in Admin → SEO & URLs → HTML Injections; print `injection.Content` | [[en/user/templates#HTML injections\|Templates: HTML injections]] |
| `defaultTemplate.Header()`, `.Footer()` | HTML of the default template's site header / footer, `""` if the page has none | Below |
| `defaultTemplate.Styles()` | `<link>` tags for the default template's stylesheet | Below |
| `defaultTemplate.UserSpaceScripts()` | The settings `<script>` and script tags that client-side widgets need | Below |
| `currentUser.IsAdmin()` | `true` when the visitor is the site admin | Below; used in [[en/user/themes\|Themes]] |

To put the default template's header, footer and styles around your own content:

```jet
<head>
  <title>{{ title }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main>{{ note.HTMLString() }}</main>
  {{ defaultTemplate.Footer() }}
  {{ if currentUser.IsAdmin() }}
    <a href="/admin">Admin</a>
  {{ end }}
</body>
```

`UserSpaceScripts()` writes the settings object only on its first call, so calling it twice doesn't duplicate it. In the layout preview (`/_system/renderlayout`) all four `defaultTemplate` functions return `""` and `currentUser.IsAdmin()` returns `false`.

#### Placeholders in layout source

`@lid` and `@did` in a layout file are replaced with the file's id before parsing (`mesh/bar.html` → `mesh_bar` / `mesh-bar`, `my-theme/card.html` → `my_theme_card` / `my-theme-card`), so block and CSS names stay unique. See [[en/user/yield_blocks|yield_blocks]] and [[en/user/bem|BEM naming]].

### The page: `note`

Documented in [[en/user/templates#What's available in a custom template|Templates]]: `Title()`, `HTMLString()`, `Permalink()`, `ReadingTime()`, `CreatedAt()`, `M()`, `PartialRenderer()`; `FormSpecJSON()` in [[en/user/forms#Custom layout|Forms]]. The rest:

| Method | Returns | Description |
|---|---|---|
| `HasH1()` | `bool` | The content starts with an H1 that serves as the title; skip your own `<h1>` |
| `ContentString()` | `string` | Raw markdown source |
| `Description()` | `string` | SEO description, `""` if none |
| `Author()` | `string` | Frontmatter `author`, `""` if unset |
| `UpdatedAt()` | `time.Time` | From `updated_at`, `updated` or `modified`; zero if unset (check `.IsZero()`) |
| `Tags()` | `[]string` | From `tags`, else `keywords`; a list or a comma-separated string. `nil` if unset |
| `OGImageURL()` | `string` | URL of the `og_image` (else `cover`) frontmatter image, `""` if unresolved |
| `FirstImageURL()` | `string` | URL of the first image in the note, `""` if none |
| `FirstListHTML()` | `string` | HTML of the first `<ul>` in the note, `""` if none |
| `TOC()` | list of `{Text, Level, ID}` | Headings for a table of contents; empty when the note's TOC setting hides it |
| `PermalinkEncoded()` | `string` | `Permalink()` with each path segment percent-encoded: `/привет мир` → `/%D0%BF…%20%D0%BC…`. See below |
| `Path()` | `string` | File path in the vault, e.g. `blog/post.md` |
| `PathID()`, `VersionID()` | `int64`, `string` | Stable ids for `data-` attributes |
| `IsSystem()` | `bool` | Any path segment starts with `_` |
| `IsHomePage()` | `bool` | The note is the home page of one of its subgraphs |
| `ReadingComplexity()` | `int` | 0–2 |
| `Lang()`, `LangName()` | `string` | Normalized language code (`en`), native name (`English`) |
| `HasLangAlternatives()` | `bool` | The note has versions in other languages |
| `LangAlternative("ru")` | `note` or nil | The version in that language |
| `LangAlternativesList()` | list of `note` | All language versions, sorted by code |
| `HasCodeLanguage("mermaid")`, `HasAnyCodeBlock()`, `HasCharts()`, `HasTaskListItems()` | `bool` | Content checks, for loading a widget script only when needed |
| `SubgraphNamesJSON()` | `string` | JSON list of the note's subgraphs |
| `LastEditedBy()`, `LastEditedByLabel()` | object / `string` | Who pushed the current version. **Admin-only data:** wrap in `currentUser.IsAdmin()` |

`PermalinkEncoded()` is URL encoding, not HTML escaping. Since output is escaped, `Permalink()` is safe in `href` too. Use `PermalinkEncoded()` when a path has non-ASCII characters or spaces and you want a strictly encoded URL, for example in a sitemap, a feed or a link another program reads.

A language switcher:

```jet
{{ range i, alt := note.LangAlternativesList() }}
  <a href="{{ alt.PermalinkEncoded() }}" hreflang="{{ alt.Lang() }}">
    {{ alt.LangName() }}
  </a>
{{ end }}
```

#### Frontmatter: `note.M()`

Documented in [[en/user/templates#What's available in a custom template|Templates]] and [[en/user/jet-debugging|Debugging Jet templates]]: `GetString`, `GetInt`, `GetBool`, `GetStrings`, `Has`, `Debug`. Details the code adds:

| Method | Returns | Behaviour |
|---|---|---|
| `Get(key)` | raw value | `nil` if missing. You can't do arithmetic on it: `Get("order") + 1` fails; use `GetInt` |
| `GetString(key, def)` | `string` | `def` when missing **or not a string**. `order: 3` gives `def` |
| `GetInt(key, def)` | `int` | Accepts integers and floats (truncated: `2.7` → `2`); `def` otherwise, including for `"3"` |
| `GetBool(key, def)` | `bool` | `true`/`false`; strings `"true"`, `"yes"`, `"1"` are true and any other string is false; numbers: non-zero is true |
| `GetStrings(key)` | `[]string` | A list (non-string items dropped) or a single string as a one-item list; empty list, never nil |
| `Raw()` | map | The whole frontmatter, e.g. for `json(note.M().Raw())` |

YAML dates without quotes (`date: 2024-05-10`) arrive as strings. Read them with `GetString`. `CreatedAt()` is the parsed `created_at` / `created_on` value when set.

### Other notes: `nvs` and queries

Documented in [[en/user/templates#What's available in a custom template|Templates]] and [[en/user/templates#Querying notes in templates|Querying notes]]: `nvs.ByPath`, `nvs.ByGlob` and the query methods `SortBy`, `SortByMeta`, `Asc`, `Desc`, `Limit`, `Offset`, `All`, `First`, `Last`. The rest:

| Method | Returns | Description |
|---|---|---|
| `nvs.ByPermalink("/about")` | `note` or nil | Lookup by URL |
| `nvs.ByWikilink("Page name")` | `note` or nil | Resolves like an Obsidian `[[link]]`; with a `/` it is a path, otherwise the shortest matching path wins |
| `nvs.List()` | list of `note` | All notes except those under `_` paths, sorted by permalink |
| `nvs.Query()` | query | A query over all notes, no glob |
| `nvs.BackLinks(note)` | list of `note` | Notes that link to `note`, without system notes, sorted by title (case-insensitive), then by permalink |
| `nvs.OutLinks(note)` | list of `note` | Notes `note` links to |
| `nvs.Sidebars(note)` | list of `note` | The note whose permalink is in the `sidebar` field (`sidebar: false` gives none), else the subgraph sidebars, else `/_sidebar` |
| `nvs.HomePages(note)` | list of `note` | Home pages of the note's subgraphs |
| `nvs.ResolveURL(note)` | `string` | The note's URL |
| query `.Public()` | query | Keeps only notes an anonymous visitor may read: `free`, not behind sign-in, not under `_`, not `noindex`. Also mentioned in [[en/user/seo\|SEO]] |

Behaviour of queries that the code defines:

- **Everything is included unless you filter it.** `ByGlob` and `Query` return paid and sign-in-only notes and `_` system notes too. On a public page, add `.Public()`.
- **Globs match vault paths without a leading slash.** `"blog/*.md"` matches; `"/blog/*.md"` matches nothing. `*` stays in one folder, `**` descends.
- **Without a sort the order is random**, and it changes between renders. Sorting is stable, but ties keep that random order. Add a tie-breaker: `.SortByMeta("date").Desc().SortBy("Title")`.
- `SortBy` takes any `note` method without arguments: `Title`, `CreatedAt`, `Permalink`, `PathID`, `ReadingTime`, `UpdatedAt`, `Author`… (`title`, `created_at`, `path` also work). An unknown name leaves the order unchanged without an error.
- `SortByMeta` compares strings, integers, floats and times. A missing value sorts first (last with `Desc`). Values of different types, such as `1` and `1.5`, compare as equal.
- `Desc()` / `Asc()` change the sort added just before them.
- `Limit(0)` means no limit. An `Offset` past the end gives an empty list.
- `First()` sets the query's limit to 1 for good. Don't reuse a query after calling `First()` on it.

A lookup that finds nothing (`ByPath`, `ByPermalink`, `ByWikilink`, a query's `First()` / `Last()`, `LangAlternative()`, `Section()`, `FirstList()`) returns `nil`. `{{ if x }}` and `{{ if x == nil }}` both test it. Calling a method on it, `x.Title()`, stops the render. Declare and test in one tag:

```jet
{{ if about := nvs.ByPermalink("/about"); about }}
  <a href="{{ about.PermalinkEncoded() }}">{{ about.Title() }}</a>
{{ end }}
```

### Content in parts: `note.PartialRenderer()`

`Introduce()`, `Sections(level)`, `Section(title or anchor)`, `FirstList()`, `Lists()`, `Images()`, `CodeBlocks(lang)` and the section fields `TitleHTML`, `ContentHTML`, `ID` and `Level` are documented in [[en/user/templates|Templates]]. Also available:

| Method | Returns |
|---|---|
| `FirstImageURL()` | URL of the first image |
| section `.Title` | Heading as plain text |
| section `.Sections(level)`, `.Section("Title")` | Subsections of a section |
| list item `.Task`, `.TaskMark` | `""`, `"todo"` or `"done"`, and the character inside `[ ]` |

### Worked example: latest posts with a count

A listing that takes the five newest public notes from `blog/`, sorted by the `date` frontmatter field, and says how many there are in total:

```jet
{{ query := nvs.ByGlob("blog/*.md").Public() }}
{{ total := len(query.All()) }}
{{ sorted := query.SortByMeta("date").Desc().SortBy("Title") }}
{{ posts := sorted.Limit(5).All() }}
<h2>Latest posts</h2>
<p>Showing {{ len(posts) }} of {{ total }}</p>
<ul>
{{ range i, post := posts }}
  <li>
    <a href="{{ post.PermalinkEncoded() }}">{{ post.Title() }}</a>
    <time>{{ post.M().GetString("date", "undated") }}</time>
  </li>
{{ else }}
  <li>No posts yet.</li>
{{ end }}
</ul>
```

How it works:

- `.Public()` drops paid and draft (`_`) notes, so `total` counts only what a visitor can open.
- `query.All()` doesn't change the query. The same `query` is reused for sorting. Only `First()` would change it.
- The sort is kept in its own variable, `sorted`, because a method chain can't continue on the next line.
- `date: 2024-05-10` is a string, and ISO dates sort correctly as strings. Notes without `date` go last.
- `.SortBy("Title")` breaks ties between posts with the same date. Without it their order would change from render to render.
- A title such as `Q&A` is escaped on output; no filter is needed.

This template is rendered by a test in the trip2g repository (`internal/layoutloader/jet_functions_example_test.go`), so it stays correct as the code changes. The shorter snippets on this page are rendered by `internal/layoutloader/jet_functions_snippets_test.go`.

### See also

- [[en/user/templates|Templates]] — how layouts work
- [[en/user/components|Template components]] — how to structure a layout
- [[en/user/spa|An app on top of trip2g]] — a layout that ships a JavaScript app
- [[en/user/jet-debugging|Debugging Jet templates]]
- [[en/user/yield_blocks|yield_blocks]] — components, assets, `@lid` / `@did`
- [[en/user/renderlayout|Layout preview endpoint]] — try a template without uploading it
- [Jet syntax reference](https://github.com/CloudyKit/jet/blob/master/docs/syntax.md) — upstream docs
