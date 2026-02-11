# Frontmatter Patches: дизайн-документ

## Цель

Правила на основе путей, которые патчат YAML-метаданные (frontmatter) markdown-файлов через jsonnet-выражения. Вычисляются на этапе `parsePage()` в `internal/mdloader/loader.go`.

Сейчас `DefaultLayout`, `SiteTitleTemplate` и `free` задаются отдельными механизмами: конфиг-значениями в БД, флагами в `loader.Config`, проверками в `rendernotepage`. Frontmatter patches объединяют это в единый декларативный механизм: «для таких-то путей — примени такие-то метаданные». Правила хранятся в БД, управляются через админку, чейнятся по приоритету.

Jsonnet выбран с самого начала, а не простой JSON merge — потому что даже простые случаи (`{ free: true }`) выглядят как обычный JSON, а сложные (условная логика, доступ к текущим метаданным) не требуют отдельного DSL.

## Сценарий использования

1. **Замена `DefaultLayout`** — правило `*`, приоритет 0: `{ layout: "default" }`. Все заметки без явного `layout:` во frontmatter получают layout по умолчанию.
2. **Замена `SiteTitleTemplate`** — правило `*`, приоритет 0: `meta + { title: meta.title + " — My Site" }`. Title каждой заметки дополняется суффиксом.
3. **Bulk `free` для раздела** — правило `blog/*`: `{ free: true }`. Все заметки в `blog/` становятся бесплатными.
4. **Разные layouts по разделам** — правило `blog/*`, приоритет 10: `{ layout: "blog_layout" }`. Перезаписывает default layout для blog-раздела.
5. **Условная логика** — правило `*`: `if std.objectHas(meta, "layout") then {} else { layout: "default" }`. Ставит layout только если он не задан явно.
6. **Premium контент** — правило `premium/**`: `meta + { free: false, reading_complexity: if std.objectHas(meta, "reading_complexity") then meta.reading_complexity else "advanced" }`.

---

## Таблицы

### `note_frontmatter_patches`

```sql
create table note_frontmatter_patches (
  id integer primary key autoincrement,
  include_patterns text not null,              -- JSON array: ["blog/*", "docs/**"]
  exclude_patterns text not null default '[]', -- JSON array: ["*.draft.md"]
  jsonnet text not null,                       -- jsonnet expression (auto-wrapped, см. раздел "Автооборачивание")
  priority integer not null default 0,         -- lower = evaluated first, rules chain
  description text not null default '',        -- human-readable description
  enabled boolean not null default true,
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now'))
);
```

**Заметки:**
- `include_patterns` / `exclude_patterns` — JSON array строк. Заметка матчится если подходит хотя бы под один include паттерн И НЕ подходит ни под один exclude. Матчинг через `doublestar.Match` (уже зависимость проекта, см. `internal/templateviews/query.go`). Поддерживает `*` (один уровень) и `**` (рекурсивно).
- `jsonnet` — тело jsonnet-выражения. Система автоматически оборачивает его (см. раздел "Автооборачивание"). Пользователь пишет только выражение, возвращающее объект.
- `priority` — порядок применения. Меньше = раньше. Правила с одинаковым приоритетом применяются в порядке `id`. Правила чейнятся: каждое следующее видит `meta` после всех предыдущих.
- `description` — для отображения в админке и в статистике применённых патчей.
- `created_by` — обязательный, для аудита.

---

## Архитектура

### Flow: от загрузки патчей до применения

```
Load() — internal/mdloader/loader.go
    |
    +-- Патчи переданы в Options.FrontmatterPatches (уже загружены из БД, скомпилированы)
    |
    v
parsePage(src) — для каждой заметки
    |
    +-- Parse markdown, extract rawMeta
    |
    v
rawMeta = meta.Get(context)                          // line 525
    |
    v
rawMeta = ldr.applyFrontmatterPatches(src.Path, rawMeta)   // >>> NEW <<<
    |   [для каждого патча (sorted by priority):]
    |   +-- glob match: include/exclude patterns через doublestar.Match
    |   +-- если не матчится -> skip
    |   +-- evaluate jsonnet с текущим meta и path
    |   +-- если ошибка -> warning на заметке, skip этот патч
    |   +-- shallow merge результата OVER rawMeta
    |   +-- записать в stats: {patchID, description}
    |
    v
pp.RawMeta = rawMeta                                 // line 551
    |
    v
pp.Free = pp.RawMeta["free"] == true                 // line 570 — теперь free может прийти из патча
    |
    v
pp.ExtractSubgraphs(), pp.ExtractMetaData()          // остальные поля извлекаются из rawMeta как обычно
```

### Где хранятся скомпилированные патчи

```go
type loader struct {
    // ... existing fields ...
    frontmatterPatches []CompiledFrontmatterPatch // из Options, sorted by priority
}

type Options struct {
    // ... existing fields ...
    FrontmatterPatches []CompiledFrontmatterPatch
}
```

Патчи загружаются из БД и компилируются один раз в вызывающем коде (например, при старте сервера или при reload заметок). В `loader` передаются уже готовые к evaluation экземпляры.

---

## Jsonnet: автооборачивание

Пользователь пишет только **тело выражения**. Система автоматически оборачивает его:

```
// Пользователь пишет:
{ free: true }

// Система генерирует:
local meta = std.parseJson(std.extVar("meta"));
local path = std.extVar("path");
{ free: true }
```

### Доступные переменные

| Переменная | Тип | Описание |
|------------|-----|----------|
| `meta` | object | Текущие метаданные заметки (после предыдущих патчей в цепочке) |
| `path` | string | Путь заметки, например `"blog/my-post.md"` |

### Стандартная библиотека jsonnet

Доступны все функции `std.*`:
- `std.objectHas(meta, "field")` — проверка наличия поля
- `std.startsWith(path, "blog/")` — проверка пути
- `std.length()`, `std.split()`, `std.join()` и т.д.

### Результат выражения

Jsonnet-выражение **должно вернуть объект** (`{}`). Этот объект shallow-мержится поверх текущего `rawMeta`.

- `{ free: true }` — добавляет/перезаписывает только `free`
- `meta + { title: meta.title + " — Site" }` — возвращает полный meta с изменённым title
- `{}` — ничего не меняет (no-op)

---

## Jsonnet: примеры

### 1. Сделать все заметки в blog/ бесплатными

```
patterns: ["blog/*"]
jsonnet:  { free: true }
```

### 2. Default layout для всех заметок

```
patterns: ["*"]
priority: 0
jsonnet:  { layout: "default" }
```

### 3. Другой layout для blog

```
patterns: ["blog/*"]
priority: 10
jsonnet:  { layout: "blog_layout" }
```

Приоритет 10 > 0, значит этот патч применяется после default layout. Результат: blog-заметки получают `blog_layout`, остальные — `default`.

### 4. Title template (замена SiteTitleTemplate)

```
patterns: ["*"]
jsonnet:  meta + { title: meta.title + " — My Site" }
```

Использует `meta` (авто-инжектированный), дополняет title суффиксом.

### 5. Условный layout (только если не задан)

```
patterns: ["*"]
jsonnet:  if std.objectHas(meta, "layout") then {} else { layout: "default" }
```

Если `layout` уже есть в frontmatter заметки — возвращает `{}` (no-op). Если нет — ставит default.

### 6. Сложный патч для premium-раздела

```
patterns: ["premium/**"]
jsonnet:
  meta + {
    free: false,
    reading_complexity:
      if std.objectHas(meta, "reading_complexity")
      then meta.reading_complexity
      else "advanced"
  }
```

### 7. Логика на основе пути

```
patterns: ["*"]
jsonnet:
  if std.startsWith(path, "blog/")
  then { free: true }
  else {}
```

Хотя это можно решить паттернами `["blog/*"]`, пример показывает что path-логика доступна в jsonnet.

---

## Мерж-семантика

### Shallow merge

Результат jsonnet-выражения **shallow-мержится** поверх текущего `rawMeta`:

```go
result := evaluateJsonnet(patch, rawMeta, path)
for k, v := range result {
    rawMeta[k] = v
}
```

Это значит:
- `{ free: true }` — добавляет ключ `free`, не трогает остальные
- `{ layout: "blog" }` — перезаписывает `layout`, не трогает остальные
- `meta + { title: "new" }` — возвращает весь meta + перезаписывает title. При shallow merge все ключи из результата перезапишут rawMeta, включая те что не менялись (идемпотентно)

### Чейнинг (цепочка правил)

Правила сортируются по `(priority ASC, id ASC)`. Каждое правило видит `meta` **после** всех предыдущих:

```
Заметка: blog/post.md, frontmatter: { title: "My Post" }

Rule 1 (priority=0, patterns=["*"]):     { layout: "default" }
  → rawMeta = { title: "My Post", layout: "default" }

Rule 2 (priority=10, patterns=["blog/*"]): { layout: "blog_layout", free: true }
  → rawMeta = { title: "My Post", layout: "blog_layout", free: true }

Rule 3 (priority=20, patterns=["*"]):    meta + { title: meta.title + " — Site" }
  → rawMeta = { title: "My Post — Site", layout: "blog_layout", free: true }
```

Rule 3 видит `meta.title = "My Post"` (оригинальный) и `meta.layout = "blog_layout"` (от Rule 2).

### Удаление ключей

Shallow merge не удаляет ключи. Если нужно «убрать» значение, можно перезаписать его на `null`:

```
{ layout: null }
```

Код, читающий `rawMeta["layout"]`, получит `nil` — как будто поля нет.

---

## Статистика

### Per-note: какие патчи повлияли на заметку

В `NoteView` добавляется поле:

```go
type AppliedFrontmatterPatch struct {
    PatchID     int
    Description string
}

// В NoteView:
AppliedFrontmatterPatches []AppliedFrontmatterPatch
```

Заполняется в `applyFrontmatterPatches()`: каждый раз, когда патч матчится и успешно применяется, добавляется запись.

### Per-patch: какие заметки затронул патч

Обратный индекс строится после загрузки всех заметок:

```go
// В NoteViews или возвращается отдельно:
FrontmatterPatchStats map[int][]string  // patch_id -> []path
```

Строится итерацией по `NoteViews.List` → по `AppliedFrontmatterPatches` каждой заметки.

### Отображение в админке

- **Страница заметки** — блок «Applied frontmatter patches» со списком описаний + ID патчей
- **Список патчей** — колонка «Affected notes» с количеством затронутых заметок
- **Страница патча** — список всех затронутых путей

---

## Производительность

### Pre-compile

Jsonnet-выражения компилируются в AST один раз при загрузке патчей из БД. Используется `google/go-jsonnet`:

```go
import "github.com/google/go-jsonnet"

type CompiledFrontmatterPatch struct {
    ID              int
    IncludePatterns []string
    ExcludePatterns []string
    Priority        int
    Description     string
    // compiledAST хранит pre-parsed jsonnet
    compiledAST     jsonnet.AST  // или строка — зависит от API go-jsonnet
    wrappedSource   string       // полный jsonnet с auto-wrapping
}
```

### VM reuse

`go-jsonnet` `VM` — stateless (ext vars задаются перед каждым evaluation). Один VM переиспользуется для всех вызовов `parsePage()`. Loader однопоточный (итерирует `Sources` последовательно), гонок нет.

```go
func (ldr *loader) applyFrontmatterPatches(path string, rawMeta map[string]interface{}) map[string]interface{} {
    // VM создаётся один раз в Load(), хранится в loader
    for _, patch := range ldr.frontmatterPatches {
        if !matchPatterns(patch, path) {
            continue
        }

        metaJSON, _ := json.Marshal(rawMeta)
        ldr.jsonnetVM.ExtVar("meta", string(metaJSON))
        ldr.jsonnetVM.ExtVar("path", path)

        result, err := ldr.jsonnetVM.EvaluateAnonymousSnippet("patch", patch.wrappedSource)
        if err != nil {
            // warning, skip
            continue
        }

        var merged map[string]interface{}
        json.Unmarshal([]byte(result), &merged)
        for k, v := range merged {
            rawMeta[k] = v
        }
    }
    return rawMeta
}
```

### Оценка времени

| Операция | Время |
|----------|-------|
| Простой snippet (`{ free: true }`) | ~10 us |
| Snippet с `meta +` | ~20 us |
| Snippet с условиями | ~30 us |
| Glob match per patch | ~1 us |

Для сайта с 1000 заметок и 5 патчей: ~5 * 1000 * 15us = ~75ms дополнительно к полной загрузке. Приемлемо.

---

## Обработка ошибок

### Валидация при сохранении

При создании/обновлении патча — evaluate jsonnet против синтетических данных:

```go
testMeta := map[string]interface{}{"title": "test"}
testPath := "test/page.md"
```

Если jsonnet выдает ошибку — мутация возвращает ошибку, патч не сохраняется. Это ловит синтаксические ошибки и ошибки типов в compile time.

**Что НЕ ловит:** ошибки, зависящие от конкретных метаданных. Например, `meta.nonexistent_field` пройдёт валидацию с тестовым meta, но может упасть на реальной заметке. Это ожидаемо — runtime ошибки обрабатываются gracefully.

### Runtime ошибки

Если jsonnet-выражение падает для конкретной заметки:

1. **Логируется warning** на заметке через `pp.AddWarning(model.NoteWarningWarning, "frontmatter patch %d failed: %v", patch.ID, err)`
2. **Патч пропускается** для этой заметки
3. **Загрузка продолжается** — следующие патчи в цепочке и следующие заметки обрабатываются нормально
4. **Сайт не ломается** — заметка отображается с теми метаданными, что были до сбойного патча

### Что может пойти не так

| Ситуация | Поведение |
|----------|-----------|
| Синтаксическая ошибка в jsonnet | Ловится при сохранении, патч не создается |
| `meta.field` на заметке без `field` | Runtime error, warning, патч пропущен |
| Jsonnet возвращает не объект (массив, строку) | Runtime error при unmarshal, warning, патч пропущен |
| Jsonnet зацикливается | Timeout VM (100ms), warning, патч пропущен |
| Невалидный glob pattern | Ловится при сохранении через `doublestar.Match` |

---

## Структура кода

### Новые пакеты

```
internal/frontmatterpatch/
+-- patch.go          — типы (CompiledFrontmatterPatch), Compile(), Evaluate(), ApplyRules()
+-- match.go          — matchPatterns() через doublestar
+-- patch_test.go     — table-driven tests: compile, evaluate, merge, chaining, error handling

internal/case/admin/
+-- createfrontmatterpatch/
|   +-- resolve.go      — создание патча (admin mutation), валидация jsonnet
|   +-- resolve_test.go
+-- updatefrontmatterpatch/
|   +-- resolve.go      — обновление, ре-валидация jsonnet
|   +-- resolve_test.go
+-- deletefrontmatterpatch/
    +-- resolve.go      — удаление (hard delete)
    +-- resolve_test.go
```

### Изменения в существующем коде

| Файл | Изменение |
|------|-----------|
| `internal/mdloader/loader.go` | Поле `frontmatterPatches` в `loader`, `FrontmatterPatches` в `Options`, метод `applyFrontmatterPatches()`, VM init в `Load()` |
| `internal/model/note.go` | Тип `AppliedFrontmatterPatch`, поле `AppliedFrontmatterPatches` в `NoteView` |
| `internal/graph/schema.graphqls` | Мутации, запросы, типы (см. раздел GraphQL) |
| `internal/graph/schema.resolvers.go` | Резолверы мутаций и запросов |
| `db/migrations/NNNN_note_frontmatter_patches.sql` | Миграция |
| `db/queries/queries.read.sql` | Запросы чтения |
| `db/queries/queries.write.sql` | Запросы записи |

---

## GraphQL схема

```graphql
# Admin mutations
type Mutation {
  createFrontmatterPatch(input: CreateFrontmatterPatchInput!): FrontmatterPatch!
  updateFrontmatterPatch(input: UpdateFrontmatterPatchInput!): FrontmatterPatch!
  deleteFrontmatterPatch(id: Int!): Boolean!
}

input CreateFrontmatterPatchInput {
  includePatterns: [String!]!        # glob patterns: ["blog/*", "docs/**"]
  excludePatterns: [String!]         # exclude glob patterns (optional)
  jsonnet: String!                   # jsonnet expression body
  priority: Int! = 0                 # lower = evaluated first
  description: String! = ""
  enabled: Boolean! = true
}

input UpdateFrontmatterPatchInput {
  id: Int!
  includePatterns: [String!]
  excludePatterns: [String!]
  jsonnet: String
  priority: Int
  description: String
  enabled: Boolean
}

# Admin queries
type Query {
  frontmatterPatches: [FrontmatterPatch!]!
  frontmatterPatchStats(patchId: Int!): FrontmatterPatchStats!
}

type FrontmatterPatch {
  id: Int!
  includePatterns: [String!]!
  excludePatterns: [String!]!
  jsonnet: String!
  priority: Int!
  description: String!
  enabled: Boolean!
  createdAt: DateTime!
  updatedAt: DateTime!
  affectedNotesCount: Int!           # количество затронутых заметок (из stats)
}

type FrontmatterPatchStats {
  patchId: Int!
  affectedPaths: [String!]!          # пути заметок, которые затронул этот патч
}

# На NoteView (если доступен в GraphQL)
type NoteView {
  # ... existing fields ...
  appliedFrontmatterPatches: [AppliedFrontmatterPatchInfo!]!
}

type AppliedFrontmatterPatchInfo {
  patchId: Int!
  description: String!
}
```

---

## SQL-запросы (sqlc)

```sql
-- queries.read.sql

-- name: ListFrontmatterPatches :many
select * from note_frontmatter_patches order by priority asc, id asc;

-- name: ListEnabledFrontmatterPatches :many
select * from note_frontmatter_patches where enabled = true order by priority asc, id asc;

-- name: FrontmatterPatchByID :one
select * from note_frontmatter_patches where id = ?;


-- queries.write.sql

-- name: InsertFrontmatterPatch :one
insert into note_frontmatter_patches (include_patterns, exclude_patterns, jsonnet, priority, description, enabled, created_by)
values (?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: UpdateFrontmatterPatch :one
update note_frontmatter_patches
set include_patterns = coalesce(?, include_patterns),
    exclude_patterns = coalesce(?, exclude_patterns),
    jsonnet = coalesce(?, jsonnet),
    priority = coalesce(?, priority),
    description = coalesce(?, description),
    enabled = coalesce(?, enabled),
    updated_at = datetime('now')
where id = ?
returning *;

-- name: DeleteFrontmatterPatch :exec
delete from note_frontmatter_patches where id = ?;
```

---

## Точка интеграции

### В `parsePage()` — `internal/mdloader/loader.go`

Вставка после строки 525 (rawMeta получен из парсера) и до строки 551 (rawMeta присваивается в NoteView):

```go
// line 525-526
rawMeta = meta.Get(context)

// >>> NEW: Apply frontmatter patches <<<
var appliedPatches []model.AppliedFrontmatterPatch
rawMeta, appliedPatches = ldr.applyFrontmatterPatches(src.Path, rawMeta)

// ... existing code ...

pp.RawMeta = rawMeta                                // line 551
pp.AppliedFrontmatterPatches = appliedPatches       // >>> NEW <<<
```

### В `Options` — `internal/mdloader/loader.go`

```go
type Options struct {
    Sources []SourceFile
    Log     logger.Logger
    Version string
    Config  Config

    NoteCache func(source SourceFile) *model.NoteView

    // >>> NEW <<<
    FrontmatterPatches []frontmatterpatch.CompiledPatch
}
```

### В `Load()` — инициализация VM

```go
func Load(options Options) (*model.NoteViews, error) {
    ldr := &loader{
        // ... existing ...
        frontmatterPatches: options.FrontmatterPatches,
    }

    // >>> NEW: init jsonnet VM if patches exist <<<
    if len(ldr.frontmatterPatches) > 0 {
        ldr.jsonnetVM = jsonnet.MakeVM()
        ldr.jsonnetVM.MaxStack = 500 // prevent stack overflow
        // 100ms timeout prevents infinite loops in jsonnet
        ldr.jsonnetVM.Timeout = 100 * time.Millisecond
    }

    // ... rest of Load() ...
}
```

---

## Замена конфигов

### `DefaultLayout` → frontmatter patch

**Текущий механизм:** `config.DefaultLayout` (из `SiteConfig`) → `rendernotepage.go:119` → `endpoint.go:105-106`.

**Замена:** Создать патч:
```
patterns: ["*"]
priority: 0
jsonnet:  if std.objectHas(meta, "layout") then {} else { layout: "default" }
```

Layout попадает в `rawMeta["layout"]`, откуда его читает `ExtractMetaData()` → `NoteView.Layout`. Код в `endpoint.go` продолжает работать как раньше, но теперь layout приходит из frontmatter, а не из конфига.

### `SiteTitleTemplate` → frontmatter patch

**Текущий механизм:** `env.SiteTitleTemplate()` → `rendernotepage.go:164` → `formatTitle()`.

**Замена:** Создать патч:
```
patterns: ["*"]
priority: 100   // высокий приоритет, чтобы применяться последним
jsonnet:  meta + { title: meta.title + " — My Site" }
```

Title модифицируется на этапе `parsePage()`, до `ExtractTitle()`. Функция `formatTitle()` в `rendernotepage` больше не нужна.

### Миграция

Не автоматическая. Админ вручную создаёт патчи и убирает конфиг-значения. Документировать в UI: «Если вы используете DefaultLayout или SiteTitleTemplate в конфиге — рекомендуем мигрировать на frontmatter patches».

В будущем (этап 3): миграционный скрипт, который проверяет наличие конфиг-значений и предлагает создать патчи автоматически.

---

## План реализации

### Этап 1: Ядро

1. Миграция: таблица `note_frontmatter_patches`
2. SQL-запросы (sqlc) + `make sqlc`
3. `internal/frontmatterpatch/` — типы, Compile(), Evaluate(), ApplyRules(), matchPatterns()
4. Тесты: compile, evaluate, merge, chaining, error handling, glob matching
5. Интеграция в `internal/mdloader/loader.go` — `Options`, `applyFrontmatterPatches()`, jsonnet VM init
6. `model.AppliedFrontmatterPatch` тип + поле в `NoteView`
7. Admin mutations: create/update/delete frontmatter patch (с валидацией jsonnet)
8. Admin queries: `frontmatterPatches` (список), `frontmatterPatchStats` (per-patch)
9. `appliedFrontmatterPatches` на NoteView в GraphQL

### Этап 2: Admin UI

10. Фронтенд: CRUD патчей в админке (список, форма, jsonnet editor)
11. Фронтенд: превью — «какие заметки затронет этот патч» (dry run)
12. Фронтенд: на странице заметки — блок «Applied patches»

### Этап 3: Миграция конфигов (опционально)

13. Deprecation warning в UI для DefaultLayout / SiteTitleTemplate конфигов
14. Кнопка «Migrate to frontmatter patch» — создаёт патч из конфиг-значения
15. Удаление legacy-кода `formatTitle()` и fallback на `DefaultLayout` после полной миграции

---

## Решённые вопросы

1. **Почему jsonnet, а не JSON merge?** Jsonnet — надмножество JSON. Простые случаи (`{ free: true }`) выглядят как JSON. Сложные случаи (условия, доступ к meta) не требуют отдельного DSL. Одна зависимость (`google/go-jsonnet`), хорошо протестирована.

2. **Автооборачивание vs полный jsonnet?** Автооборачивание. Пользователь пишет только тело выражения. Система добавляет `local meta = ...; local path = ...;`. Проще, меньше ошибок, меньше boilerplate.

3. **Shallow merge vs deep merge?** Shallow merge. Deep merge неочевиден для вложенных структур и может привести к неожиданным результатам. Если нужно изменить вложенный объект — пользователь явно возвращает `meta + { nested: meta.nested + { key: "value" } }`.

4. **Что делать при runtime ошибке jsonnet?** Warning на заметке, skip патча, продолжить. Не ломать загрузку сайта из-за одного сбойного патча на одной заметке.

5. **Валидация при сохранении?** Да. Evaluate против синтетических `{ "title": "test" }` / `"test/page.md"`. Ловит синтаксические ошибки. Не ловит runtime-зависимые ошибки — это ожидаемо.

6. **Порядок при одинаковом priority?** По `id` (autoincrement). Детерминированный, предсказуемый.

7. **Pre-compile или evaluate каждый раз?** Pre-compile AST при загрузке патчей. VM переиспользуется. ~10us per evaluation.

8. **Одна VM или pool?** Одна VM. Loader однопоточный, parsePage() вызывается последовательно. Гонок нет.

9. **Hard delete или soft delete патчей?** Hard delete. Патчи не имеют deliveries или связанных записей. Если понадобится аудит — добавим soft delete позже.

10. **Как отслеживать какие патчи применились?** Per-note: массив `AppliedFrontmatterPatches` в `NoteView`. Per-patch: обратный индекс `FrontmatterPatchStats`, строится итерацией по NoteViews после загрузки.

---

## Открытые вопросы / Future

1. **Deep merge mode** — опциональный флаг `deep_merge: true` на патче. Пока не нужен, shallow merge покрывает все текущие use cases.
2. **Dry run в API** — мутация `testFrontmatterPatch(jsonnet, path, meta)` которая возвращает результат без сохранения. Полезно для отладки.
3. **Import в jsonnet** — возможность импортировать общие функции из файла. Пока не нужен, выражения простые.
4. **Версионирование патчей** — история изменений. Пока достаточно `updated_at`.
5. **Bulk operations** — включить/выключить все патчи, изменить приоритеты drag-and-drop.
6. **Автоматическая миграция конфигов** — скрипт, который создаёт патчи из `DefaultLayout` и `SiteTitleTemplate` и убирает конфиг-значения.
