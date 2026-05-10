---
title: "Multilingual sites"
free: true
lang_redirect: "[[ru/user/multilingual]]"
---

Publish notes in multiple languages from a single vault. trip2g uses a folder-based approach: each language lives in its own subfolder, and visitors can switch between them with an automatically generated language toggle.

### How it works

Organize your vault with one folder per language:

```
docs/
  en/
    user/
      getting-started.md
      multilingual.md
  ru/
    user/
      Начало работы.md
      multilingual.md
```

Add a `lang` property to the homepage of each language section to declare which language it represents:

```yaml
---
lang: en
---
```

trip2g uses `lang` to associate notes with a language and render the correct `<html lang="...">` attribute for each page.

### Language redirects

Connect the same page across languages with `lang_redirect`. When this property is set, a language switcher appears automatically in the page header — visitors can click it to jump to the equivalent page in the other language.

```yaml
---
title: "Multilingual sites"
lang_redirect: "[[ru/user/multilingual]]"
---
```

The value must be a wikilink pointing to the corresponding note. On the Russian page, point back to the English one:

```yaml
---
title: "Мультиязычный сайт"
lang_redirect: "[[en/user/multilingual]]"
---
```

The language switcher shows only when the linked note exists and is accessible to the visitor.

### Language switcher

When `lang_redirect` is set, a language switcher appears automatically in the page header. It shows the current language with a flag icon and full native name. Clicking opens a dropdown with the available alternatives.

Flag icons are built in for ten languages:

| `lang` value | Flag | Displayed as |
|---|---|---|
| `en` | 🇺🇸 | English |
| `ru` | 🇷🇺 | Русский |
| `de` | 🇩🇪 | Deutsch |
| `fr` | 🇫🇷 | Français |
| `es` | 🇪🇸 | Español |
| `zh` | 🇨🇳 | 中文 |
| `ja` | 🇯🇵 | 日本語 |
| `pt` | 🇵🇹 | Português |
| `ar` | 🇸🇦 | العربية |
| `ko` | 🇰🇷 | 한국어 |

The `lang` value is case-insensitive. Common aliases are supported — `"English"`, `"RU"`, `"German"` are all normalized to the canonical code.

For other languages, no flag is shown unless you add it via custom CSS:

```css
.lang-switcher__current--uk::before,
.lang-switcher__alt--uk::before {
  content: '';
  display: inline-block;
  width: 18px;
  height: 13px;
  background-image: url('/your-flag.png');
  background-size: cover;
  border-radius: 2px;
}
```

Replace `uk` with your `lang` code (lowercased) and point `background-image` to your flag asset.

### Language-scoped sidebars

Place a `_sidebar.md` inside a language folder and it applies to all notes in that folder and its subfolders:

```
docs/
  en/
    user/
      _sidebar.md    ← applies to all EN user notes
  ru/
    user/
      _sidebar.md    ← applies to all RU user notes
```

Each sidebar can be written in the appropriate language and link only to notes in that language section. Wikilinks in RU sidebars use the short form since the sidebar and notes share the same folder context:

```markdown
**Getting Started**
- [[en/user/getting-started|Install & first sync]]
- [[en/user/multilingual|Multilingual sites]]
```

```markdown
**Начало работы**
- [[Начало работы]]
- [[multilingual|Мультиязычность]]
```

### Language-scoped footers

`_footer.md` works the same way as `_sidebar.md` — place it in the language root folder and it applies to all notes beneath it:

```
docs/
  en/
    _footer.md    ← footer for all EN notes
  ru/
    _footer.md    ← footer for all RU notes
```

### Per-language homepages

Use the magazine system to build a homepage that shows cards only for one language. The `magazine_include_files` property scopes which notes appear as cards, and `lang` marks the homepage language:

```yaml
---
lang: en
content: [self, magazine]
magazine_include_files: en/**/*.md
magazine_include_property: home_position
---
```

```yaml
---
lang: ru
content: [self, magazine]
magazine_include_files: ru/**/*.md
magazine_include_property: home_position
---
```

Control the order of cards with `home_position` on each note:

```yaml
---
home_position: 1
---
```

Lower numbers appear first. Notes without `home_position` are not included in the magazine.

### Tips

- **Consistent slugs** — using the same `slug` value (e.g. `slug: multilingual`) on both language versions makes URLs predictable and easier to link across languages.
- **One sidebar per language** — create a `_sidebar.md` in each language folder so navigation is always in the reader's language.
- **Separate homepages** — give each language its own `_index.md` with `lang` set and `magazine_include_files` scoped to that language's folder.
- **`lang_redirect` is optional per page** — add it only where a translated equivalent exists. Pages without a counterpart simply won't show a language toggle.

### Related

- [[en/user/publishing|Publishing notes]] — frontmatter properties reference
- [[en/user/advanced|Custom domains, CLI, SEO]] — serve each language on its own domain
