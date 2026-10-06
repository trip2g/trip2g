---
title: "How much work a template engine hides"
free: true
lang_redirect: "[[ru/thoughts/template-engine-hidden-work]]"
---

*What this is about: in one week we found in trip2g's templates escaping that was switched off, a `nil` that isn't `nil`, anchors that no wikilink lands on, and a reference that said the opposite of the code in three places. A static site generator or a CMS does all of this for you, and you never find out. Read it if you write your own layouts for trip2g or are picking a template engine for your own product.*

For years I used static site generators and CMSs and never once thought about how much work stands behind `{{ title }}`. You write a variable, it shows up on the page. If the title has a `<` in it, the markup doesn't break. If you link to a heading, the browser scrolls to it. If a note is missing, the template shows a fallback.

trip2g has its own template layer, a wrapper around [Jet](https://github.com/CloudyKit/jet). This week I saw that each of those small things is a separate decision someone has to make. In a few places we hadn't made it.

## Escaping and its contexts

The worst one first. trip2g builds layouts with `jet.WithSafeWriter(nil)` ([`internal/layoutloader/loader.go`](https://github.com/trip2g/trip2g/blob/main/internal/layoutloader/loader.go)). That option switches escaping off entirely. Every text value printed as is, and the `| unsafe` and `| raw` filters did nothing: there was nothing to switch off. Meanwhile three docs pages, [[en/user/templates|Templates]] and two Russian pages, said HTML is escaped by default.

What that means is easier to show. Take a kanban board layout that lets you edit a card in place:

```jet
<textarea>{{ note.ContentString() }}</textarea>
```

If any card's text contains `</textarea><script>…`, the browser closes the field and runs the script for everyone who opens the board. I checked it on bare Jet: with escaping off, the string `x&y</textarea><script>` reaches the page unchanged; with escaping on, it becomes `x&amp;y&lt;/textarea&gt;&lt;script&gt;`.

Escaping wasn't switched off by accident. It was the fix for another bug, which I described in [[en/thoughts/anatomy-15-months|the 15-month anatomy]]. `asset()` returns a signed object storage URL whose parameters are separated by `&`. By default Jet turned `&` into `&amp;`. In an `href` attribute that's right: the browser decodes `&amp;` back. Inside `<style>`, in `url(...)`, nothing is decoded, so storage got a signature with `&amp;` in it and refused. Escaping was switched off entirely, the CSS worked, and everything else was left without protection.

The mistake isn't one decision. Escaping that ignores context is wrong one way or the other. The same value needs different treatment in text, in an attribute, in `<style>`, in `<script>` and in a URL. Go's [`html/template`](https://pkg.go.dev/html/template) does exactly that: it parses the HTML around each insertion and picks the escaping for the spot. I ran the same URL through it: in `href` it wrote `&amp;`, in `url()` inside `<style>` it kept `&`. That's why [Hugo](https://gohugo.io/templates/introduction/), built on `html/template`, never has this bug. [Jinja2](https://jinja.palletsprojects.com/en/stable/api/#autoescaping) only knows HTML escaping, and a bare `Environment` has it off until you turn it on. Jet also has one way of escaping for the whole output. A Hugo user never thinks about contexts because the Go authors thought about them.

## A `nil` that isn't `nil`

The second story is about checking for nothing. A template looks a note up by path: `{{ x := nvs.ByPath("/about.md") }}`. When the note didn't exist, the function returned a typed nil pointer, `*Note(nil)`. To Jet that's not the same thing as the `nil` literal. So:

- `{{ if x == nil }}` was **false** for a missing note;
- `{{ if x }}` worked correctly;
- `x.Title()` panicked inside the method and cut the page render short.

I reproduced all three on plain Jet v6.3.1. It's a classic Go trap, and the template engine hands it straight to a layout author who never thought about types. The fix is in [PR #384](https://github.com/trip2g/trip2g/pull/384): lookups that can miss now return an untyped `nil`. Both forms of the check work, and calling a method on a miss is a render error instead of a panic. Along the way we documented the idiomatic Go form that Jet already has: `{{ if x := nvs.ByPath("/about.md"); x }}…{{ end }}`. The variable only lives inside the `if` and the `else`.

## Anchors and ids

Every heading on a page has an `id` so it can be linked to. Someone has to decide how that id is made. trip2g builds it from the text: Cyrillic is transliterated, anything other than letters and digits becomes `_`, it's lowercased, and a repeat gets `-2`. `## Цены и тарифы` becomes `cenyi_i_tarifyi`.

Change the heading text and the id changes, and old links break. So in [PR #385](https://github.com/trip2g/trip2g/pull/385) we turned on goldmark's attribute parser: `## Pricing {#pricing}` sets the id explicitly, and the braces disappear from the text, the table of contents and search. Obsidian shows those braces as is. I used to count that as a reason not to do it. Now the notes on trip2g sites are more and more often written by agents, and braces in the editor don't bother them.

The third part of this story is still open. The wikilink `[[Pricing#Plans and prices]]` passes the fragment through as written, and the link ends in `#Plans%20and%20prices`. There's no such id on the page, so the browser opens it at the top. In Obsidian the same link works, because Obsidian finds the heading by text, not by id. For now we've documented it and advise writing the id after the hash.

## The reference that lied

To catch all of this I sat down to write a full reference of what a layout can call ([PR #383](https://github.com/trip2g/trip2g/pull/383)). There was one rule: check every claim against the code. Besides escaping, this turned up:

- `range` with one variable gives the index, not the value: `{{ range x := list }}` prints `0`, `1`, not the items;
- `note.Title` without parentheses prints a function address, something like `0xc8e90`;
- the docs listed a `GetStringSlice` method that doesn't exist; the real one is `GetStrings`;
- `_` as a loop variable, `return` and `try` stop a layout from loading: trip2g walks the template tree when it loads, and the walker doesn't know these nodes. Because of `return`, `exec` is useless too, since it returns exactly that value;
- `|` filters are only allowed in output: in an assignment or an `if` you need a function call, `parseJSON(s)`, not `s | parseJSON`.

None of these is hard. But each one had to be found, checked and written down by someone. In an SSG, the team and thousands of users before you did that.

## Why this matters more when the author is an agent

A person who copies an example from the docs and sees a function address on the page fixes it in a minute. An agent copies the example exactly as written, and may not notice the error at all: the page rendered, after all. And it does that on every site where it's asked to build a layout.

So for us the template docs are no longer a reference but the source of behaviour. A wrong example there becomes a wrong pattern in many places at once. Hence the requirement on the reference: it has a test that renders the docs' worked example and checks the output, so the example can't go stale silently. Hence also component auto-import: a layout that calls `{{ yield card() }}` gets the component without an explicit `import`. The fewer rules you need to know, the fewer you can break. But each convenience like that is one more piece of work hidden from the author, and it too has to be described honestly.

There's another side. Agents increasingly use templates as consumers of data. A note carries chart data in a code block tagged `mychart`, and the layout pulls it out with `CodeBlocks("mychart")` and parses it with `parseJSON`. That's why #385 adds `parseJSON`, `parseYAML` and `parseCSV`. A template engine that recently printed a title turns into a small programming language. And it has the same problems as any language: empty values, parse errors, scope.

## What we changed

- A missed lookup returns a real `nil` (#384).
- Explicit heading ids with `{#id}`, data from code blocks, JSON, YAML and CSV parsing (#385).
- A Jet functions reference and fixes to three pages that disagreed with the code (#383).
- Next: escaping by default. We're working on it: methods that return ready HTML will be marked safe through Jet's `Renderer` interface, so old layouts with `note.HTMLString()` keep working without `| unsafe`.

All three PRs are still open and waiting for review.

## If you write layouts for trip2g today

- Escape yourself: `{{ value | html }}` for any text that came from a note or a user. Especially inside `<textarea>`, `<script>` and attributes.
- Don't put signed URLs from `asset()` into `<style>` through `| html`: `&amp;` isn't decoded there.
- Check a lookup with `{{ if x := nvs.ByPath(...); x }}`, not `== nil`, until #384 is merged.
- Give `range` two variables: `{{ range i, item := list }}`.
- Call methods with parentheses: `note.Title()`.
- Don't use `_`, `return`, `try` or `exec`.
- Link to headings by id, not by text.

More in [[en/user/templates|Templates]], [[en/user/jet-debugging|Debugging Jet]] and [[en/user/markdown|Markdown syntax]].
