# Bare wikilinks on multilingual sites: resolution design

## TL;DR

Recommendation: replace the flat "global basename, shallowest path wins" rule for bare `[[X]]` links with a deterministic preference ladder — **same folder → same language → global shallowest, with a stable tie-break** — implemented in one place (`NoteViews.ResolveWikilinkTarget`) and reused by the loader. Keep explicit paths (`[[en/user/My Doc]]`, `[[./My Doc]]`) as the documented escape hatch; they are untouched.

Two facts make this safer than it sounds:

1. Today's tie-break for equal-depth candidates (exactly the `en/user/X` vs `ru/user/X` case) is **nondeterministic** — it depends on Go map iteration order when `BasenameMap` is built. The comment in `note.go` claims the candidate list is depth-sorted; the builder never sorts it. So for the most common multilingual collision there is no stable behavior to preserve — we are fixing a bug, not changing a contract.
2. The only links whose target can change *deterministically* are bare links where a same-folder or same-lang candidate exists deeper than a different global-shallowest winner. A one-shot doclint migration report can list every such link before the switch.

## 1. Current algorithm, precisely

Resolution lives in two near-duplicate copies:

- `internal/model/note.go` — `NoteViews.ResolveWikilinkTarget(source, target)` (used by GraphQL `schema.resolvers.go:3243` and the template layer via `loader.go:854`);
- `internal/mdloader/loader.go` — `extractInLinks()` (the canonical pass: fills `ResolvedLinks`, `InLinks` backlinks, and broken-link warnings).

Both do:

- **Branch 1** — target starts with `./` or `../`: resolve against `filepath.Dir(source.Path)` via `PathMap`. Fully relative, unambiguous.
- **Branch 2** — target has no `/` (bare link): lowercase it, look up `BasenameMap[basename]`.
  - 1 candidate → use it.
  - N candidates → pick the one with the fewest `/` in `Path` (shallowest). Ties keep `candidates[0]`.
  - 0 → broken link.
- **Branch 3** — target contains `/`: walk up from the source's directory toward root, trying `dir[:i] + "/" + target` against `PathMap` at each level. First hit wins. This is the escape hatch that already works today.

`BasenameMap` is built in `loader.go` `buildBasenameIndex()` by ranging over `nvs.PathMap` — a Go map, so **candidate order is random per process**. The doc comment on `BasenameMap` (`note.go:312`, "sorted by path depth (shallowest first)") is wrong; nothing sorts it. `strings.Count(path, "/")` then picks a strict minimum, so among equal-depth candidates the winner is whichever the map handed over first.

**Why `[[My Doc]]` is ambiguous today.** With `docs/en/user/My Doc.md` and `docs/ru/user/My Doc.md`, both candidates have identical depth. Neither the source note's folder nor its language enters Branch 2 at all — `source` is not even consulted. The link therefore resolves to a coin-flip between the two languages, and the coin can land differently on the next server restart. `NoteView.Lang` (frontmatter `lang`, parsed at `note.go:624`) exists but is unused by resolution.

**How doclint sees it.** `internal/doclint/lint.go` flags "cross-language wikilink leak" only when a bare link has *exactly one* candidate and that candidate sits under the other language prefix (`pathLangPrefix`/`otherLang`). Multi-candidate basenames are deliberately skipped with the comment that shortest-depth resolution is "deterministic but language-blind" — the deterministic half of that comment is false for equal-depth ties, which is precisely the bilingual-pair case. So today the linter is blind to the worst case.

## 2. How others resolve bare links

- **Obsidian.** Resolution is global across the vault, not relative. When a basename is ambiguous, Obsidian doesn't guess at click time — it *writes* disambiguated links: autocomplete inserts the shortest *unique* path (`[[user/My Doc]]`), governed by the "New link format" setting (shortest path when possible / relative to file / absolute in vault) in Settings → Files and links. So a bare `[[My Doc]]` with duplicates basically never survives in an Obsidian-authored vault; the editor prevents the ambiguity at write time. Forum threads on duplicate names all converge on "use the path form" ([disambiguating multiple files with the same name](https://forum.obsidian.md/t/disambiguiting-mutiple-files-with-the-same-name/32110), [problem with links when there are multiple files with same name](https://forum.obsidian.md/t/problem-with-links-when-there-are-multiple-files-with-same-name/68944), [links help](https://obsidian.md/help/links)). trip2g can't rely on the editor doing this, because files also arrive via git push, agents, and hand-written docs.
- **Foam.** Identifiers are "shortest unambiguous path suffix": `[[todo]]` is ambiguous given `projects/house/todo.md` and `work/todo.md`; `[[house/todo]]` is valid. Ambiguous bare links resolve **alphabetically (deterministic) plus a warning diagnostic** telling you to be more specific ([Foam wikilinks](https://docs.foamnotes.com/features/wikilinks/)). Determinism + a lint warning, no source-context preference.
- **Quartz.** Mirrors Obsidian: case-insensitive global matching via the CrawlLinks plugin, expecting Obsidian to have written unique links ([Quartz wikilinks](https://quartz.jzhao.xyz/features/wikilinks)).
- **Dendron.** Sidesteps the problem structurally: notes are addressed by hierarchical dot-paths (`lang.en.user.my-doc`), so a "bare" name *is* a full address; cross-vault links get a `[[vault-name/note]]` prefix.
- **MediaWiki.** Page titles are absolute — `[[My Doc]]` is one global page, duplicates cannot exist. The relative idiom only exists inside subpage trees (`[[/child]]`, `[[../sibling]]`), i.e. relative-to-current-page is the mechanism that makes "nearby" links work.

The pattern: engines either make ambiguity impossible (MediaWiki, Dendron), push disambiguation to write time (Obsidian, Quartz), or resolve deterministically and warn (Foam). Nobody resolves ambiguous bare links *randomly*. And the rule that makes same-folder links "just work" everywhere it exists (MediaWiki subpages, Obsidian's "relative to file" write format, plain Markdown links) is **prefer targets near the source**.

## 3. Options for trip2g

### A. Current-folder first, then global

Bare link checks `dir(source)/X.md` in `PathMap` first; on miss, falls back to today's global lookup.

- Fixes the bilingual-pair case whenever the pair lives in mirrored folders (`en/user/` ↔ `ru/user/`) — which is the docs-vault layout.
- Cheap: one `PathMap` probe before the existing code.
- Does **not** fix cross-folder same-lang links: `en/user/a.md` → `[[Federation]]` living at `en/advanced/federation.md` still races with `ru/advanced/federation.md`.
- Backward compat: a link that used to (deterministically) hit a shallower note in another folder now hits the sibling. Real but enumerable (see migration below).

### B. Language-aware preference

Compute the source's language (frontmatter `Lang`, else first path segment when it matches a known lang prefix) and filter candidates to same-lang before applying shallowest-wins.

- Fixes all cross-folder same-lang links, not just siblings. Ties directly into the existing `lang` frontmatter and the doclint `pathLangPrefix` logic (which should move into `model` and be shared).
- Needs a definition of "the site's languages". Options: reuse doclint's hardcoded en/ru (bad), derive from top-level folders that look like lang codes, or make it config (`languages = ["en", "ru"]` — trip2g already has multilang machinery, `docs/dev/multilang.md`). Deriving from notes that declare `lang:` frontmatter plus top-level `en/`/`ru/`-style folders is enough.
- Alone, it doesn't help monolingual vaults with duplicate names (still random ties) and doesn't prefer the sibling within a language.

### C. Explicit path as escape hatch (status quo, documented)

Branch 3 already resolves `[[user/My Doc]]` and `[[en/user/My Doc]]`; Branch 1 handles `[[./My Doc]]`. Keep, document in user docs, have doclint tell authors to use it when a bare link is ambiguous (Foam's model).

- Zero behavior change, zero risk.
- Pushes the cost onto every author and every agent forever, and existing content stays randomly resolved. Not a fix by itself — but it *is* the right escape hatch and the right doclint suggestion text regardless of which option ships.

### D. Hybrid ladder (recommended): same-folder → same-lang → global shallowest, deterministic tie-break

For a bare link from `source`:

1. **Same folder**: candidate with `filepath.Dir(candidate.Path) == filepath.Dir(source.Path)` wins.
2. **Same language**: among remaining candidates, keep those whose lang (frontmatter `Lang`, else path prefix) matches the source's; pick the shallowest.
3. **Global shallowest** (today's rule) over whatever is left.
4. Every step breaks ties **by lexicographic `Path`** — never by map order. Sorting the `BasenameMap` slices once at build time (depth, then path) makes the existing doc comment true and gives all steps stable input.

Same-folder-first means mirrored bilingual trees just work even without lang detection; same-lang catches cross-folder links; step 3 preserves current behavior for everything that has no nearby/same-lang candidate. Ambiguity that survives to step 3 is exactly what doclint should keep warning about, with "use `[[path/Name]]`" as the suggested fix.

### Backward compatibility, honestly

Three classes of existing bare links:

| Class | Today | Under D |
|---|---|---|
| Unique basename | resolves to the one note | identical — untouched |
| Multi-candidate, equal depth (the en/ru pair) | **random per restart** | deterministic, lang/folder-correct |
| Multi-candidate, distinct depths, and a same-folder/same-lang candidate is *deeper* than the global winner | shallowest wins, stably | **retargets** to the nearby candidate |

Only the third class is a genuine silent change, and it is mechanically enumerable: run both resolvers over every note's bare links and diff. Ship that as a one-shot doclint mode (`lint docs --resolution-diff` or just a temporary check) and eyeball the list before flipping. On the docs vault the expected diff is "links start landing in the right language," which is the point. If any instance needs the old rule, a config knob (`wikilink_resolution = "global" | "scoped"`, default `scoped`) is cheap insurance — but default to the new behavior; the old one was partly random.

### Interaction with doclint

- The cross-lang-leak check (single-candidate, other-lang target) stays as is — under D a single candidate still resolves the same way, and that warning ("you linked a page that only exists in the other language") stays meaningful, usually signalling a missing translation.
- The "multi-candidate is deterministic so we skip it" comment gets rewritten: under D, doclint *can* now flag any bare link that still resolves cross-language after the ladder (real leak, no same-lang candidate exists) and can downgrade surviving global-shallowest ambiguity to an info-level "consider `[[path/Name]]`" hint, Foam-style.
- `pathLangPrefix`/`otherLang` move from `doclint` into `model` (or wherever the lang detection lands) so lint and resolver share one definition of a note's language.

## 4. Recommended change surface

Ship **Option D**, resolver-side, default-on, with the migration diff.

1. **`internal/mdloader/loader.go` — `buildBasenameIndex()`**: after building, sort each `BasenameMap` slice by (depth asc, path asc). Fixes the nondeterminism on its own and makes the `note.go:312` comment true. This is worth doing even if nothing else ships.
2. **`internal/model/note.go` — `ResolveWikilinkTarget` Branch 2**: replace the shortest-scan with a `pickBareCandidate(source, candidates)` helper implementing the ladder. Lang detection: `source.Lang` if set, else first path segment when it is a recognized language folder (shared helper absorbed from doclint's `pathLangPrefix`, generalized past hardcoded en/ru — top-level folders matching declared/known lang codes).
3. **`internal/mdloader/loader.go` — `extractInLinks()`**: delete the duplicated bare-name block and call `nvs.ResolveWikilinkTarget(p, target)`, keeping the surrounding `ResolvedLinks`/`InLinks`/warning bookkeeping. One algorithm, one place; the current duplication is how the two copies would otherwise drift.
4. **`internal/doclint/lint.go`**: update the multi-candidate comment/logic per above; add the one-shot resolution-diff report for migration.
5. **Config (optional)**: `wikilink_resolution` app setting defaulting to the ladder, `global` for strict legacy behavior. Skip it if the migration diff on real instances is clean.
6. **Docs**: user-facing note in the linking docs — bare links prefer same folder, then same language; use `[[path/Name]]` or `[[./Name]]` to pin a target explicitly (bilingual pair, per project convention).

Sources: [Obsidian links help](https://obsidian.md/help/links) · [Obsidian forum: disambiguating duplicate names](https://forum.obsidian.md/t/disambiguiting-mutiple-files-with-the-same-name/32110) · [Obsidian forum: multiple files with same name](https://forum.obsidian.md/t/problem-with-links-when-there-are-multiple-files-with-same-name/68944) · [Foam wikilinks](https://docs.foamnotes.com/features/wikilinks/) · [Quartz wikilinks](https://quartz.jzhao.xyz/features/wikilinks)
