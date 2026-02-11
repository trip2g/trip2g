# updateNote: атомарное обновление заметок через find/replace

## Цель

Новая мутация `updateNote` и расширение формата agent response для атомарного обновления содержимого заметок без полной перезаписи. Основная операция — find/replace: найти строку в текущем содержимом и заменить на новую.

## Проблема

Текущий flow для обновления части заметки:
1. Прочитать содержимое (GraphQL query)
2. Модифицировать на клиенте
3. `pushNotes` с полным новым содержимым
4. `commitNotes`

Проблемы:
- **Race condition**: между чтением и записью кто-то мог изменить заметку
- **Сложность**: требует GraphQL query + два вызова mutation
- **Избыточность**: для простых операций (дописать в конец) — слишком много шагов

## Решение: find/replace

Атомарная операция: сервер читает текущее содержимое, находит строку, заменяет, создаёт новую версию — всё в одной транзакции.

```
find:    "$MARKER$"
replace: "## New Entry\nContent\n\n$MARKER$"
```

Сервер делает `strings.Replace(currentContent, find, replace, 1)` и сохраняет результат.

`find` — произвольная строка, которую выбирает пользователь. Никакого дефолтного маркера — пользователь сам решает что использовать: `$INSERT$`, `<!-- INSERT -->`, `%INBOX%`, `🔽`, или любую другую строку.

### Паттерн: маркер-якорь

Чтобы делать последовательные вставки, `replace` включает исходный маркер. Так маркер остаётся в тексте и работает как точка для следующей вставки:

```
Запрос 1: find="$INBOX$" replace="Entry 1\n\n$INBOX$"
Запрос 2: find="$INBOX$" replace="Entry 2\n\n$INBOX$"
Запрос 3: find="$INBOX$" replace="Entry 3\n\n$INBOX$"
```

Результат:
```markdown
Entry 1

Entry 2

Entry 3

$INBOX$
```

---

## GraphQL мутация

```graphql
input UpdateNoteInput {
  path: String!
  find: String!       # строка для поиска в текущем содержимом
  replace: String!    # строка для замены
}

type UpdateNotePayload {
  notePathId: Int64!
}

union UpdateNoteOrErrorPayload = UpdateNotePayload | ErrorPayload

type Mutation {
  """
  Атомарная операция find/replace в содержимом заметки.
  X-Api-Key или Authorization: Bearer header обязателен.
  Автоматически коммитит (не требует отдельного commitNotes).
  """
  updateNote(input: UpdateNoteInput!): UpdateNoteOrErrorPayload!
}
```

### Автоматический commit

В отличие от `pushNotes` (требует отдельного `commitNotes`), `updateNote` коммитит сразу. Причина: операция атомарная, нет batch-семантики. После InsertNote вызывается `HandleLatestNotesAfterSave` → триггерятся вебхуки, обновляются подграфы и т.д.

### Auth

Работает с обоими типами авторизации:
- `X-Api-Key` header → обычный API key
- `Authorization: Bearer {token}` → shortapitoken JWT (из webhook payload)

Write patterns из shortapitoken проверяются перед записью — как и для pushNotes.

---

## Расширение agent response (webhooks)

Формат `changes[]` в ответе агента (см. [shared_webhooks.md](shared_webhooks.md)) расширяется поддержкой find/replace:

### Текущий формат (полная замена)

```json
{
  "changes": [
    {
      "path": "inbox.md",
      "content": "полное новое содержимое",
      "expected_hash": "abc123..."
    }
  ]
}
```

### Новый формат (find/replace)

```json
{
  "changes": [
    {
      "path": "inbox.md",
      "find": "<!-- INSERT -->",
      "replace": "## New Entry\nContent\n\n<!-- INSERT -->"
    }
  ]
}
```

### Правила определения режима

| Поля в change | Режим | Описание |
|---------------|-------|----------|
| `content` (без `find`) | full_replace | Текущее поведение — полная замена |
| `find` + `replace` (без `content`) | find_replace | Атомарный find/replace |
| `content` + `find` | ошибка | Конфликт — нельзя указать оба |
| Ни `content`, ни `find` | ошибка | Нет операции |

### Совместимость с expected_hash

`expected_hash` работает в обоих режимах:
- **full_replace**: проверяет хеш перед перезаписью (текущее поведение)
- **find_replace**: проверяет хеш перед find/replace — гарантирует что содержимое не менялось с момента когда агент его видел

Для find/replace `expected_hash` опционален. Если не указан — сервер не проверяет, просто делает замену.

---

## Edge cases

| Ситуация | Поведение |
|----------|-----------|
| `find` не найден в содержимом | Ошибка: `"marker not found: {find}"` |
| Заметка не существует | Ошибка: `"note not found: {path}"` |
| Несколько вхождений `find` | Ошибка: `"multiple occurrences of find string, use more specific marker or full replace"` |
| `find` == `replace` | Нет изменений — не создавать новую версию |
| Пустой `find` | Ошибка валидации |
| Пустой `replace` | ОК — удаляет маркер (замена на пустую строку) |
| `expected_hash` mismatch | Ошибка, откат транзакции (как в full_replace) |

---

## Примеры использования

### Inbox (дописать в конец через маркер)

```json
{
  "path": "inbox.md",
  "find": "$INBOX$",
  "replace": "## 2026-02-10 15:30\nНовое сообщение\n\n$INBOX$"
}
```

### Чеклист — пометить задачу выполненной

```json
{
  "path": "todo/sprint.md",
  "find": "- [ ] Deploy v2.0",
  "replace": "- [x] Deploy v2.0"
}
```

На фронте — сахар `completeTask(path, taskText)` поверх find/replace.

**Требуется доработка markdown→HTML рендера**: чеклист-элементы должны выводить `data-line` с исходной markdown-строкой:

```html
<li data-line="- [ ] Deploy v2.0">
  <input type="checkbox" /> Deploy v2.0
</li>
```

Включается флагом в frontmatter:
```yaml
allow_task_toggle: true
```

Для массового включения — через [frontmatter patch](frontmatter_patches.md): `* → { allow_task_toggle: true }`.

Фронт при клике на чекбокс (только если флаг включён):
```js
const line = el.dataset.line
updateNote({ path, find: line, replace: line.replace('[ ]', '[x]') })
```

### Обновление метаданных в frontmatter

```json
{
  "path": "blog/post.md",
  "find": "status: draft",
  "replace": "status: published"
}
```

### Добавление тега

```json
{
  "path": "blog/post.md",
  "find": "tags: [go, sql]",
  "replace": "tags: [go, sql, performance]"
}
```

### Вставка в начало (prepend через маркер)

```markdown
%TOP%

## Existing content
```

```json
{
  "find": "%TOP%",
  "replace": "%TOP%\n\n## New Entry\nContent"
}
```

---

## Реализация

### Новый пакет

```
internal/case/updatenote/
├── resolve.go      — find/replace логика + InsertNote + auto-commit
└── resolve_test.go
```

### Логика

```go
func Resolve(ctx context.Context, env Env, input UpdateNoteInput) (UpdateNoteOrErrorPayload, error) {
    // 1. Прочитать текущее содержимое.
    note, err := env.LatestNoteByPath(ctx, input.Path)
    if err != nil {
        return model.NewError("note not found: " + input.Path), nil
    }

    // 2. Find/replace.
    idx := strings.Index(note.Content, input.Find)
    if idx == -1 {
        return model.NewError("marker not found: " + input.Find), nil
    }

    // 2.1. Check for multiple occurrences (force precision).
    if strings.Index(note.Content[idx+len(input.Find):], input.Find) != -1 {
        return model.NewError("multiple occurrences of find string, use more specific marker or full replace"), nil
    }

    newContent := note.Content[:idx] + input.Replace + note.Content[idx+len(input.Find):]

    // 3. Проверка что содержимое изменилось.
    if newContent == note.Content {
        return &UpdateNotePayload{NotePathID: note.PathID}, nil
    }

    // 4. InsertNote (создает новую версию).
    pathID, err := env.InsertNote(ctx, model.RawNote{
        Path:    input.Path,
        Content: newContent,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to insert note: %w", err)
    }

    // 5. Auto-commit: триггерит вебхуки, подграфы, и т.д.
    err = env.HandleLatestNotesAfterSave(ctx, []int64{pathID})
    if err != nil {
        return nil, fmt.Errorf("failed to handle after save: %w", err)
    }

    return &UpdateNotePayload{NotePathID: pathID}, nil
}
```

### Расширение applychanges.go

```go
func applyChange(ctx context.Context, env Env, change AgentChange, depth int) error {
    if change.Find != "" {
        return applyFindReplace(ctx, env, change, depth)
    }
    return applyFullReplace(ctx, env, change, depth)
}

func applyFindReplace(ctx context.Context, env Env, change AgentChange, depth int) error {
    note, err := env.LatestNoteByPath(ctx, change.Path)
    if err != nil {
        return fmt.Errorf("note not found: %s", change.Path)
    }

    if change.ExpectedHash != "" && note.ContentHash != change.ExpectedHash {
        return fmt.Errorf("expected_hash mismatch for %s", change.Path)
    }

    idx := strings.Index(note.Content, change.Find)
    if idx == -1 {
        return fmt.Errorf("marker not found in %s: %q", change.Path, change.Find)
    }

    // Check for multiple occurrences (force precision).
    if strings.Index(note.Content[idx+len(change.Find):], change.Find) != -1 {
        return fmt.Errorf("multiple occurrences of find string in %s, use more specific marker or full replace", change.Path)
    }

    newContent := note.Content[:idx] + change.Replace + note.Content[idx+len(change.Find):]

    _, err = env.InsertNote(ctx, model.RawNote{Path: change.Path, Content: newContent})
    return err
}
```

### Валидация AgentChange (ozzo-validation)

```go
type AgentChange struct {
    Path         string `json:"path"`
    Content      string `json:"content"`
    Find         string `json:"find"`
    Replace      string `json:"replace"`
    ExpectedHash string `json:"expected_hash"`
}

func (c AgentChange) Validate() error {
    return validation.ValidateStruct(&c,
        validation.Field(&c.Path, validation.Required),
        // Должен быть либо content, либо find+replace
        validation.Field(&c.Content, validation.When(c.Find == "", validation.Required)),
        validation.Field(&c.Replace, validation.When(c.Find != "", validation.Required)),
    )
}
```

### SQL-запрос для чтения содержимого по path

```sql
-- name: LatestNoteContentByPath :one
select np.id, np.value as path, np.latest_content_hash,
       nv.content
from note_paths np
join note_versions nv on nv.path_id = np.id and nv.version = np.version_count
where np.value = ? and np.hidden_at is null;
```

---

## План реализации

### Этап 1: Backend

1. SQL-запрос `LatestNoteContentByPath` + `make sqlc`
2. `internal/case/updatenote/` — resolve + тесты
3. GraphQL schema: `updateNote` mutation
4. Resolver в `schema.resolvers.go`
5. Auth: проверка write patterns для updateNote (как для pushNotes)

### Этап 2: Agent response

6. Расширить `AgentChange` struct — добавить `Find`, `Replace` поля
7. Обновить валидацию в `agentresponse.go`
8. Добавить `applyFindReplace` в `applychanges.go`
9. Тесты для find/replace режима в agent response

### Этап 3: Тесты

10. Unit: find/replace с маркером, без маркера, edge cases
11. Unit: agent response с find/replace
12. E2E: webhook agent возвращает find/replace changes

---

## Решённые вопросы

1. **Почему find/replace, а не append/prepend/insertAt?** — find/replace — самый общий механизм. Через него выражаются все остальные операции: append через маркер в конце, prepend через маркер в начале, insert через маркер в произвольном месте. Один механизм вместо трёх.

2. **Почему не regexp?** — `find` — точная строка, не регулярное выражение. Агент должен точно знать что меняет, а не "что-то похожее". Предсказуемость важнее гибкости.

3. **Почему ошибка при нескольких вхождениях?** — Предсказуемость и безопасность. Если `find` строка встречается несколько раз, агент должен использовать более уникальный маркер (`<!-- inbox:2026-02-10 -->`) или явно указать что делать с каждым вхождением. Автоматическая замена первого вхождения слишком неявная и может привести к ошибкам.

4. **Автокоммит?** — Да. `updateNote` — атомарная операция на одну заметку. Нет смысла в отдельном commitNotes. Для batch-операций остаётся pushNotes + commitNotes.

4. **Совместимость с agent response?** — Обратно совместимо. Старый формат (`content` без `find`) работает как раньше. Новый формат (`find` + `replace`) — дополнение.
