# noteChanges: Live-подписка на изменения заметок

## Цель

Единая GraphQL подписка `noteChanges` для трёх потребителей:

1. **Obsidian plugin** — auto-pull файлов с сервера при изменениях агентами
2. **Админка** — live-обновление страниц (админ видит изменения онлайн)
3. **Публичный сайт** — плашка "Контент обновился, хотите обновить страницу?"

## Текущее состояние

### Как сейчас работает sync
- **Ручной sync**: клик по иконке → classify → pull/push/conflict
- **Polling**: каждые 60 сек `checkForPendingChanges()` обновляет badge
- **Focus check**: при фокусе окна — тоже проверка
- **Push**: пользователь подтверждает отправку
- **Pull**: только при ручном sync (если `twoWaySync: true`)

### SSE инфраструктура (уже есть)
- **Backend**: gqlgen SSE transport (`transport.SSE{}` в `handler.go`)
- **Единственная подписка**: `currentTime(format: String)` — тестовая
- **Caddy**: SSE запросы обходят timeout/compress handlers
- **Клиент (фронтенд)**: `$trip2g_sse_host` в `assets/ui/sse/sse.ts` — авто-реконнект, парсинг событий
- **`$trip2g_graphql_raw_subscription`** в `assets/ui/graphql/index.ts` — реализован и работает

### Система вебхуков (переиспользуем)
- `handlenotewebhooks.Resolve()` — вызывается из `commitNotes` и `hideNotes`
- `webhookutil.MatchesAny(path, patterns)` — glob matching через `doublestar`
- `NoteChange { PathID, Path, Event }` — структура события (create/update/remove)
- Фильтрация: `includePatterns` + `excludePatterns` + event type

## Архитектура решения

```
                                          ┌─────────────────┐
┌─────────────┐  SSE noteChanges(input)   │                 │
│   Obsidian  │ ◄──────────────────────── │                 │
│   Plugin    │  auto-pull files          │                 │
└─────────────┘                           │   trip2g Server │
                                          │                 │
┌─────────────┐  SSE noteChanges(input)   │  commitNotes ─┐ │
│   Админка   │ ◄──────────────────────── │  hideNotes  ──┤ │
│   (mol)     │  live reload данных       │               ▼ │
└─────────────┘                           │           notebus│
                                          │               │ │
┌─────────────┐  SSE noteChanges(input)   │               ▼ │
│  Публичный  │ ◄──────────────────────── │          SSE fan-out
│    сайт     │  плашка "обновить"        │                 │
└─────────────┘                           └─────────────────┘
```

### Поток данных (общий)

```
1. Агент/пользователь → pushNotes + commitNotes
2. commitNotes → IncrementSyncVersion (в той же транзакции что и save)
3. commitNotes → HandleLatestNotesAfterSave → triggerWebhooks
4.                                           → notebus.Publish (NEW)
5. notebus → фильтрация по patterns каждого подписчика → SSE event
6. Потребитель получает {paths, event, hashes, version}
```

**Eventual consistency:** между записью контента и Publish — окно ~миллисекунды (последовательные вызовы в одном goroutine). Версия уже в базе к моменту Publish.

### Поток: Obsidian plugin

```
6. SSE → Plugin получает событие с paths, hashes, event, version
7. Classify по хэшам из события (без доп. запроса)
8. Safe pull → скачать и записать
9. Conflict → Notice + badge, ждать ручного sync
```

Одно событие на один вызов Publish — paths батчатся. Debounce не нужен.

### Поток: Админка

```
6. SSE → mol-компонент получает событие
7. Инвалидировать кэш данных для изменённых paths
8. UI обновляется реактивно (mol wire)
```

### Поток: Публичный сайт

```
6. SSE → JS на странице получает событие
7. Если текущая страница в списке paths → показать плашку
8. Пользователь нажимает "Обновить" → перезагрузка / fetch нового контента
```

## Backend: GraphQL подписка

### Схема

```graphql
input NoteChangesInput {
  """Glob patterns для фильтрации (doublestar). Пустой = все."""
  includePatterns: [String!]
  """Glob patterns для исключения."""
  excludePatterns: [String!]
}

enum NoteChangeEventType {
  CREATE
  UPDATE
  REMOVE
}

type NoteChangeEvent {
  """Пути изменённых файлов."""
  paths: [String!]!
  """Хэши для каждого path (в том же порядке). Пустой массив для remove."""
  hashes: [String!]!
  """Тип события."""
  event: NoteChangeEventType!
  """Глобальная версия сайта после этого изменения."""
  version: Int64!
}

type Subscription {
  currentTime(format: String = "2006-01-02 15:04:05"): String!
  """
  X-Api-Key header must be set.
  Подписка на изменения заметок. Фильтрация по glob patterns (как в webhooks).
  """
  noteChanges(input: NoteChangesInput): NoteChangeEvent!
}
```

### Pub/Sub механизм

In-process pub/sub на каналах (~50 строк). Библиотеки не нужны:
- `kelindar/event` (552 stars) — 4-10x быстрее каналов, но для миллионов событий/сек. У нас десятки.
- `asaskevich/EventBus` (2.8k stars) — callback-based, а gqlgen ожидает каналы.
- Кастомное решение идеально ложится на gqlgen (resolver возвращает `<-chan T`).

**Структура пакетов:**

```
internal/model/notebus.go       # NoteBusEvent struct (данные)
internal/notebus/notebus.go     # Bus, Subscriber, Subscribe/Unsubscribe/Publish (логика)
```

**Тип события в `internal/model`:**

```go
// internal/model/notebus.go

// NoteBusEvent описывает изменение заметок для real-time подписок.
type NoteBusEvent struct {
    Paths   []string // Изменённые пути.
    Hashes  []string // Хэши (в том же порядке). Пустой для remove.
    Event   string   // "CREATE", "UPDATE", "REMOVE".
    Version int64    // Глобальная версия сайта после изменения.
}
```

**Bus в `internal/notebus`:**

```go
// internal/notebus/notebus.go

type Env interface {
    Logger() logger.Logger
}

type Subscriber struct {
    Ch              <-chan model.NoteBusEvent // read-only для потребителя.
    ch              chan model.NoteBusEvent   // write для Bus.
    includePatterns []string
    excludePatterns []string
}

type Stats struct {
    Subscribers int   // Текущее количество подписчиков.
    Published   int64 // Всего опубликовано событий.
    Dropped     int64 // Потеряно из-за slow consumers.
}

type Bus struct {
    mu    sync.RWMutex
    subs  map[*Subscriber]struct{}
    env   Env
    stats Stats
}

func New(env Env) *Bus { ... }
func (b *Bus) Subscribe(include, exclude []string) *Subscriber { ... }
func (b *Bus) Unsubscribe(s *Subscriber) { ... }
func (b *Bus) Publish(event model.NoteBusEvent) { ... }
func (b *Bus) GetStats() Stats { ... }
```

`Publish()` фильтрует по `webhookutil.MatchesAny()` для каждого подписчика. Buffered channel (size 64). `select` с `default` для slow consumers (drop + логирование через `env.Logger()`).

**Тесты notebus:**
1. Subscribe → Publish → получили событие
2. Subscribe с patterns → Publish с non-matching path → НЕ получили
3. Unsubscribe → Publish → канал закрыт, событие не пришло
4. 3 подписчика, Publish → все получили (fan-out)
5. Буфер заполнен → событие dropped, stats.Dropped инкрементирован

**Интеграция в `app` (`cmd/server/main.go`):**

```go
type app struct {
    // ...existing...
    noteBus *notebus.Bus
}

// Метод для case-пакетов — через Env interface.
func (a *app) PublishNoteEvent(event model.NoteBusEvent) {
    a.noteBus.Publish(event)
}

// Метод для subscription resolver.
func (a *app) SubscribeNoteChanges(include, exclude []string) *notebus.Subscriber {
    return a.noteBus.Subscribe(include, exclude)
}

func (a *app) UnsubscribeNoteChanges(sub *notebus.Subscriber) {
    a.noteBus.Unsubscribe(sub)
}

// Для admin monitoring.
func (a *app) NoteBusStats() notebus.Stats {
    return a.noteBus.GetStats()
}
```

**Env interface в case-пакетах** (commitnotes, hidenotes):

```go
// internal/case/commitnotes/resolve.go
type Env interface {
    // ...existing...
    PublishNoteEvent(event model.NoteBusEvent)
}
```

Тестируемость: в тестах мокаем `PublishNoteEvent` и проверяем что он вызван с правильным событием.

### Resolver

```go
func (r *subscriptionResolver) NoteChanges(ctx context.Context, input *model.NoteChangesInput) (<-chan *model.NoteChangeEvent, error) {
    // 1. Проверить API key
    _, err := checkapikey.Resolve(ctx, r.env(ctx), "note_changes")
    if err != nil {
        return nil, err
    }

    // 2. Парсить patterns
    var include, exclude []string
    if input != nil {
        include = input.IncludePatterns
        exclude = input.ExcludePatterns
    }

    // 3. Подписаться через app
    env := r.env(ctx)
    sub := env.SubscribeNoteChanges(include, exclude)

    // 4. Маппинг канала model.NoteBusEvent → model.NoteChangeEvent (GraphQL type)
    ch := make(chan *model.NoteChangeEvent, 1)
    go func() {
        defer close(ch)
        defer env.UnsubscribeNoteChanges(sub)
        for {
            select {
            case <-ctx.Done():
                return
            case event, ok := <-sub.Ch:
                if !ok {
                    return
                }
                ch <- &model.NoteChangeEvent{
                    Paths:   event.Paths,
                    Hashes:  event.Hashes,
                    Event:   event.Event,
                    Version: event.Version,
                }
            }
        }
    }()

    return ch, nil
}
```

### Точки интеграции (PublishNoteEvent)

**1. `HandleLatestNotesAfterSave` в `cmd/server/main.go`:**

Уже резолвит pathIDs → paths + определяет event type. Добавить `PublishNoteEvent` рядом с webhook dispatch.

**TODO при реализации:** проверить как webhook dispatch получает paths и event types — переиспользовать тот же механизм. Вероятно есть batch-запрос `NotePathsByIDs` или аналог (не использовать `NotePathByID` в цикле — это N запросов).

```go
func (a *app) HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error {
    // ...existing: updatesubgraphs, handletgpublishviews, embeddings...

    // Версия уже инкрементирована до вызова этой функции (в commitNotes).
    version := ... // передаётся как параметр или читается из контекста

    // Собираем изменения (переиспользуем данные из webhook dispatch).
    // Используем batch query, не NotePathByID в цикле.
    notePaths, npErr := a.NotePathsByIDs(ctx, changedPathIDs)
    if npErr != nil { ... }

    var busPaths []string
    var busHashes []string
    for _, np := range notePaths {
        busPaths = append(busPaths, np.Value)
        busHashes = append(busHashes, np.LatestContentHash)
    }

    if len(busPaths) > 0 {
        // Event type берём из webhook dispatch (create/update различаются).
        a.PublishNoteEvent(model.NoteBusEvent{
            Paths:   busPaths,
            Hashes:  busHashes,
            Event:   eventType, // "CREATE" или "UPDATE" — из той же логики что для webhooks
            Version: version,
        })
    }

    // ...existing webhook dispatch...
}
```

**2. `hideNotes` в `internal/case/hidenotes/resolve.go`:**

Добавить `PublishNoteEvent` в Env interface и вызвать рядом с `triggerWebhooks`.

**TODO при реализации:** проверить что `Resolve()` в hidenotes имеет доступ к paths как строкам, а не только pathIDs. Проверить flow — hideNotes использует `HandleLatestNotesAfterSave` или отдельный dispatch?

```go
type Env interface {
    // ...existing...
    PublishNoteEvent(event model.NoteBusEvent)
}

// В Resolve(), после triggerWebhooks:
env.PublishNoteEvent(model.NoteBusEvent{
    Paths:   paths,        // строковые пути
    Hashes:  []string{},   // remove — пустой массив
    Event:   "REMOVE",
    Version: version,      // из IncrementSyncVersion()
})
```

**3. pushNotes без commit** (skipCommit=true):
- Не публикуем — файлы ещё не закоммичены
- Событие уйдёт при commitNotes

### Авторизация

SSE подписка использует тот же `X-Api-Key` что и queries. Проверка через `checkapikey.Resolve()`. API key создаётся в админке и привязан к сайту.

Нюанс: SSE — long-lived соединение. Если API key отозван:
- Текущая подписка продолжит работать до дисконнекта
- Это ОК — аналогично поведению webhook deliveries
- При реконнекте — получит ошибку авторизации

### SSE Heartbeat / Keepalive

SSE соединения могут молча умирать за NAT/прокси. Решение:

**Сервер:** `transport.SSE{KeepAlivePingInterval: 30 * time.Second}` — если gqlgen поддерживает. Иначе — периодическая отправка SSE comment (`:keepalive\n\n`) из middleware.

**Клиент (Obsidian):** таймер 60 секунд. Если за 60 секунд не пришло ни одного SSE event (включая keepalive) — считаем соединение мёртвым → reconnect.

**Клиент (браузер, mol):** браузерный EventSource API автоматически обнаруживает разрыв. Дополнительная логика не нужна.

### Observability

`Bus.GetStats()` возвращает `Stats{Subscribers, Published, Dropped}`. Доступно через:

```graphql
type NoteBusStats {
  subscribers: Int!
  published: Int64!
  dropped: Int64!
}

type Admin {
  # ...existing...
  noteBusStats: NoteBusStats!
}
```

Страница в админке для мониторинга. Позже — Prometheus metrics.

## Delta Sync: убрать FetchServerHashes на 2000 файлов

### Проблема (скриншот Network tab)

```
graphql  82.5 kB   769ms   FetchServerHashes — ВСЕ 2000 хэшей
graphql  82.5 kB  1.03s    FetchServerHashes повторно (badge check)
graphql  82.5 kB  1.07s    FetchServerHashes ещё раз
graphql   223 kB   974ms   PushNotes / FetchNoteContents
graphql   223 kB  1.63s    PushNotes / FetchNoteContents
─────────────────────────────────────────────────────────
Итого:   868 kB   ~6s      При том что изменилось 0-5 файлов
```

82.5 kB x 3 = 247 kB только на хэши, при каждом sync. При 2000 заметках это ~40 байт x 2000 = 80 кБ за запрос. Растёт линейно с количеством файлов.

### Решение: глобальная версия + delta query

Добавить глобальный счётчик версий. Клиент хранит `lastSyncedVersion` и запрашивает только изменения.

**Новая колонка в `note_paths`:**
```sql
alter table note_paths add column last_changed_version integer not null default 0;
```

**Отдельная таблица для глобального счётчика:**
```sql
create table sync_version (
  id integer primary key check (id = 1),  -- singleton row
  version integer not null default 0
);
insert into sync_version (id, version) values (1, 0);
```

**Инкремент при commitNotes / hideNotes:**
```sql
-- 1. Атомарно инкрементировать глобальную версию
update sync_version set version = version + 1 where id = 1 returning version;

-- 2. Записать новую версию в изменённые paths
update note_paths set last_changed_version = ? where id in (?...);
```

Одна транзакция: инкремент + обновление paths. sqlc query для обоих. Выполняется **до** `HandleLatestNotesAfterSave`, чтобы версия была в базе к моменту Publish.

**Новый GraphQL фильтр:**
```graphql
input NotePathsFilter {
  like: String
  paths: [String!]          # NEW: фильтр по конкретным путям
  changedSinceVersion: Int64  # NEW: только изменённые после версии
}

input NotePathsDeltaFilter {
  sinceVersion: Int64!
}

type NotePathsDelta {
  """Текущая глобальная версия сайта."""
  currentVersion: Int64!
  """Изменённые paths (пустой если нет изменений)."""
  notePaths: [NotePath!]!
  """Paths удалённых/скрытых заметок с момента версии."""
  removedPaths: [String!]!
}

type Query {
  # ...existing...
  """Delta sync: только изменения с указанной версии. X-Api-Key required."""
  notePathsDelta(filter: NotePathsDeltaFilter!): NotePathsDelta!
}
```

### Как меняется flow sync

**Сейчас (2000 заметок, 5-10 сек):**
```
1. FetchServerHashes → 2000 хэшей, 82 kB, ~1s
2. Compute 2000 local hashes (cached by mtime, ~5ms)
3. Classify 2000 files
4. Push/Pull changed files
```

**После delta sync (2000 заметок, <1 сек):**
```
1. notePathsDelta(sinceVersion: 142) → 3 changed paths, ~200 bytes, ~50ms
2. Compute hashes только для 3 файлов
3. Classify 3 файла
4. Push/Pull changed files
```

**Первый sync / reset** (sinceVersion = 0):
```
Как сейчас — полный FetchServerHashes. Один раз.
```

### Как это работает с SSE

SSE и delta sync дополняют друг друга:

```
Нормальная работа:
  SSE подписка → получаем события реального времени
  Каждое событие содержит version → обновляем lastSyncedVersion

Реконнект после offline (порядок критичен!):
  1. Подключить SSE (начинаем получать новые события в буфер)
  2. notePathsDelta(sinceVersion: lastSyncedVersion) → пропущенные изменения
  3. Merge пропущенные + буферизованные события → autoPull
  4. Обновить lastSyncedVersion
  Важно: сначала SSE, потом delta — иначе события между delta и SSE потеряны.

Cold start (Obsidian запустился):
  1. notePathsDelta(sinceVersion: lastSyncedVersion) → delta с прошлой сессии
  2. Подключаем SSE

Fallback (lastSyncedVersion === undefined или невалиден):
  Полный FetchServerHashes → как сейчас. Один раз.
```

### Хранение removedPaths

Для delta sync нужно знать какие файлы были удалены/скрыты после `sinceVersion`.

Используем `hidden_by` + `last_changed_version` (без новой таблицы):
- Скрытые paths уже помечены `hidden_by IS NOT NULL`
- При hideNotes → обновлять `last_changed_version`
- Delta query: `WHERE last_changed_version > sinceVersion AND hidden_by IS NOT NULL` → removedPaths

### Хранение lastSyncedVersion в клиенте

```typescript
interface SyncState {
  files: Record<string, string>;      // path → lastSyncedHash
  mtimes?: Record<string, number>;
  localHashes?: Record<string, string>;
  lastSyncedAt?: number;
  lastSyncedVersion?: number;          // NEW: глобальная версия сервера
}
```

При `lastSyncedVersion === undefined` → первый sync, полный FetchServerHashes. После первого sync → delta.

## Frontend: Obsidian Plugin

### Новые настройки

```typescript
interface SyncDir {
    // ...existing...
    /**
     * Glob patterns для live pull подписки.
     * Если указаны — SSE соединение активно, auto-pull включён.
     * Если пустой массив / не указано — live pull выключен.
     * Примеры: ["**"], ["blog/**", "docs/**"]
     */
    livePullIncludePatterns?: string[];
    /**
     * Glob patterns для исключения из live pull.
     * Примеры: ["drafts/**", "private/**"]
     */
    livePullExcludePatterns?: string[];
}
```

Live pull включён когда `livePullIncludePatterns` непустой. `twoWaySync` должен быть включён (иначе нечего тянуть).

В UI настроек:
```
[x] Two-way sync
  Live pull include patterns: [**              ]
  Live pull exclude patterns: [drafts/**       ]
```

Поле текстовое, patterns через запятую. Пустое поле = live pull выключен. Пользователь может ограничить какие папки получать автоматически, а какие — только при ручном sync.

**Каждый syncDir — отдельный сервер.** Одно SSE соединение на syncDir. Мультиплексирование не нужно — syncDir-ы подключены к разным инстансам trip2g.

### SSE соединение

Obsidian плагин не может использовать `$trip2g_sse_host` (это mol-класс). Нужна своя реализация на чистом JS, аналогичная по логике:

```typescript
class LivePullConnection {
    private controller: AbortController | null = null;
    private reconnectTimer: number | null = null;
    private reconnectDelay = 3000;
    private lastEventAt = 0;
    private healthCheckTimer: number | null = null;
    private healthCheckInterval = 60000; // 60 сек — если нет данных, reconnect

    constructor(
        private apiUrl: string,
        private apiKey: string,
        private includePatterns: string[],
        private excludePatterns: string[],
        private onChanges: (event: NoteChangeEvent) => void,
        private onStatusChange: (connected: boolean) => void,
    ) {}

    connect(): void { ... }
    disconnect(): void { ... }
    private async stream(signal: AbortSignal): Promise<void> { ... }
    private startHealthCheck(): void {
        // Каждые 60 сек проверять lastEventAt.
        // Если > 60 сек без данных (включая keepalive) → reconnect.
    }
}
```

**GraphQL subscription query:**
```graphql
subscription NoteChanges($input: NoteChangesInput) {
    noteChanges(input: $input) {
        paths
        hashes
        event
        version
    }
}
```

**SSE transport**: POST `/graphql` с `Accept: text/event-stream` (аналогично `$trip2g_sse_host`).

### Auto-Pull логика

```typescript
async autoPull(event: NoteChangeEvent): Promise<void> {
    if (this.isSyncing) {
        // Sync уже идёт — отложить, проверим после
        this.pendingAfterSync = event;
        return;
    }

    const { paths, hashes, event: eventType, version } = event;

    if (eventType === "REMOVE") {
        // Проверить, есть ли эти файлы локально
        // Если есть — показать ServerDeletedModal
        this.updateLastSyncedVersion(version);
        return;
    }

    // CREATE или UPDATE: classify по хэшам из события (без доп. запроса)
    const safePulls: string[] = [];
    const conflicts: string[] = [];

    for (let i = 0; i < paths.length; i++) {
        const path = paths[i];
        const remoteHash = hashes[i];
        const lastSynced = syncState.files[path];
        const localHash = await computeLocalHash(path);

        if (eventType === "CREATE" && !localHash) {
            // Новый файл, локально нет — всегда safe pull
            safePulls.push(path);
        } else {
            const action = classifyFile(localHash, remoteHash, lastSynced);
            if (action === "pull") {
                safePulls.push(path);
            } else if (action === "conflict") {
                conflicts.push(path);
            }
            // push, unchanged — игнорируем в auto-pull
        }
    }

    // Выполнить safe pulls
    if (safePulls.length > 0) {
        await pullFiles(safePulls);
        // Download assets for pulled notes
        await downloadAssetsForNotes(env, safePulls);
        new Notice(`↓ ${safePulls.length} files updated from server`);
    }

    // Конфликты — уведомить
    if (conflicts.length > 0) {
        new Notice(`⚠ ${conflicts.length} conflicts detected. Click sync to resolve.`);
        updateBadge();
    }

    this.updateLastSyncedVersion(version);
}
```

### Ассеты при auto-pull

Текущий flow при обычном pull:
```
1. Pull note content (FetchNoteContents)
2. Write .md file
3. FetchNoteAssets(paths) → список ассетов [{absolutePath, url, hash}]
4. Для каждого ассета: exists locally? → нет → downloadAsset(url)
```

Переиспользуем `downloadAssetsForNotes(env, pulledPaths)` из `execute.ts` как есть. Он уже:
- Вызывает `fetchNoteAssets(paths)` только для pulled files
- Дедуплицирует по absolutePath
- Проверяет `fileExists()` перед скачиванием
- Создаёт директории при необходимости

**Оптимизация на будущее:** добавить `assetReplaces` в `FetchNoteContents`, чтобы получать контент + ассеты одним запросом (уже возможно в текущей схеме — `NotePath` имеет и `content`, и `assetReplaces`).

### Жизненный цикл SSE соединения

```
onload():
  for each syncDir where livePull && twoWaySync:
    create LivePullConnection
    connect()

onunload():
  disconnect all connections

Settings changed:
  if livePull toggled on → connect()
  if livePull toggled off → disconnect()
  if apiUrl/apiKey changed → reconnect()

Connection lost:
  auto-reconnect с delay 3s → 6s → 12s → 30s (exponential backoff, max 30s)

No data for 60s (health check):
  reconnect (серверный keepalive должен приходить каждые 30s)

Window focus:
  if connection dead → reconnect immediately
```

### Include/Exclude patterns в плагине

Patterns передаются напрямую из настроек в SSE подписку:

```typescript
function buildSubscriptionInput(syncDir: SyncDir): NoteChangesInput | null {
    const include = syncDir.livePullIncludePatterns;
    if (!include || include.length === 0) return null; // live pull выключен

    return {
        includePatterns: include,
        excludePatterns: syncDir.livePullExcludePatterns ?? [],
    };
}
```

**Важно:** `syncDir.path` — это клиентская логика (папка в Obsidian vault). На сервере paths уже относительные. Patterns в подписке матчат серверные paths как есть. Плагин не трогает файлы вне `syncDir.path` — это ортогональная защита на уровне клиента.

Пользователь контролирует:
- `["**"]` — получать всё
- `["blog/**", "docs/**"]` — только определённые папки
- exclude `["drafts/**"]` — не получать черновики
- Пустое поле — live pull выключен, только ручной sync

### Защита от собственных изменений

Когда Obsidian plugin делает push → commitNotes → событие приходит обратно по SSE.

**Решение: не фильтровать, положиться на classify + хэши в событии.**

При получении собственного события:
1. SSE приходит с `{paths: ["note.md"], hashes: ["abc123"]}`
2. Плагин сравнивает: `syncState.files["note.md"] === "abc123"` → да, только что запушили
3. `classifyFile(localHash, remoteHash="abc123", lastSynced="abc123")` → `unchanged`
4. Никаких запросов, никаких действий

## Frontend: Админка (mol)

Админка использует существующую инфраструктуру `$trip2g_graphql_subscription()`.

### Интеграция

```typescript
// Подписка на все изменения (админ видит всё)
const sub = $trip2g_graphql_subscription(`
    subscription NoteChanges($input: NoteChangesInput) {
        noteChanges(input: $input) { paths event hashes version }
    }
`, { input: { includePatterns: ["**"] } })
```

### Поведение

При получении события — инвалидировать кэш `reset_query_marker`:

```typescript
@ $mol_mem
note_changes() {
    const data = this.subscription().data()
    if (!data) return null

    // Инвалидировать кэш — все query перезапросят данные
    // reset_query_marker уже есть в graphql/index.ts
    return data.noteChanges
}
```

Админ видит:
- Список изменений в реальном времени (какие страницы изменились)
- Автообновление если открыта страница редактирования изменённой заметки
- Notice: "Страница blog/post.md обновлена агентом"

## Frontend: Публичный сайт

### Плашка "Контент обновился"

Для авторизованных пользователей (или всех — по настройке):

```typescript
// Подписка на текущую страницу
@ $mol_mem
page_update_subscription() {
    const path = this.current_page_path()
    if (!path) return null

    return $trip2g_graphql_raw_subscription(`
        subscription NoteChanges($input: NoteChangesInput) {
            noteChanges(input: $input) { paths event hashes version }
        }
    `, { input: { includePatterns: [path] } })
}

@ $mol_mem
has_update() {
    const data = this.page_update_subscription()?.data()
    if (!data) return false

    // Проверить что текущая страница в списке изменённых
    return data.noteChanges.paths.includes(this.current_page_path())
}
```

### UI плашки

```
┌──────────────────────────────────────────────┐
│ ℹ Эта страница была обновлена. [Обновить]    │
└──────────────────────────────────────────────┘
```

- Показывается вверху страницы (sticky banner)
- "Обновить" → `location.reload()` или fetch нового контента без перезагрузки
- Плашка исчезает при закрытии или обновлении
- Не показывается для `event: REMOVE` (страница удалена — другая логика)

### Авторизация для публичного сайта

Текущий resolver проверяет API key. Для публичного сайта нужно:
- Либо отдельная подписка `publicNoteChanges` без авторизации (только публичные заметки)
- Либо разрешить `noteChanges` для авторизованных пользователей (по сессии, не API key)
- Либо отложить на будущее — сейчас только для админки и Obsidian (оба с API key)

**Рекомендация:** начать без публичного сайта. Добавить позже когда ясна модель авторизации.

## План реализации

### Фаза 1: Backend — notebus + подписка + delta sync

1. **`internal/model/notebus.go`** — `NoteBusEvent` struct
2. **`internal/notebus/notebus.go`** — `Bus` с Subscribe/Unsubscribe/Publish, Stats, логирование drops
3. **Тесты notebus** — subscribe, pattern filtering, fan-out, unsubscribe, buffer overflow
4. **Встроить в `app`** — `noteBus` field, `PublishNoteEvent()`, `SubscribeNoteChanges()`, `UnsubscribeNoteChanges()`, `NoteBusStats()`
5. **Миграция** — `sync_version` таблица (singleton), `note_paths.last_changed_version` колонка
6. **sqlc queries** — `IncrementSyncVersion`, `NotePathsChangedSince`, `NotePathsRemovedSince`
7. **Инкремент версии** — в `commitNotes` и `hideNotes` (до HandleLatestNotesAfterSave, одна транзакция)
8. **Схема GraphQL** — `NoteChangeEventType` enum, `noteChanges` subscription, `NoteChangesInput`, `NoteChangeEvent` (с hashes + version), `notePathsDelta` query, `NoteBusStats` в admin
9. **`make gqlgen`** — сгенерировать код
10. **Resolver** — `NoteChanges()` с API key, подписка через `env.SubscribeNoteChanges()`
11. **Resolver** — `notePathsDelta()` с API key
12. **Publish из `HandleLatestNotesAfterSave`** — рядом с webhook dispatch, реальный event type (CREATE/UPDATE)
13. **Publish из `hideNotes`** — добавить `PublishNoteEvent` в Env interface
14. **SSE keepalive** — `transport.SSE{KeepAlivePingInterval: 30 * time.Second}` или аналог
15. **Тест**: `curl` SSE подписка + pushNotes через CLI → события приходят

### Фаза 2: Obsidian SSE live pull + delta sync

1. **`LivePullConnection`** — SSE клиент с reconnect + health check (60s timeout)
2. **Настройки** — `livePullIncludePatterns`, `livePullExcludePatterns`
3. **`autoPull()`** — classify по хэшам из события, safe pull, conflict → badge, CREATE → always pull
4. **Delta sync** — `lastSyncedVersion` в SyncState, `notePathsDelta` вместо `FetchServerHashes`
5. **Reconnect** — SSE first, then `notePathsDelta(sinceVersion)` для пропущенных событий
6. **Жизненный цикл** — connect/disconnect в onload/onunload/settings

### Фаза 3: Админка (mol) — live updates

1. **Подписка** — `$trip2g_graphql_subscription()` с `["**"]`
2. **Инвалидация кэша** — при событии обновить данные
3. **Notice** — показать что изменилось
4. **Monitoring** — страница NoteBusStats в админке

### Фаза 4: Публичный сайт (отложить)

1. Определить модель авторизации
2. Подписка на конкретную страницу
3. Плашка "Контент обновился"

## Edge Cases

| Кейс | Поведение |
|------|-----------|
| SSE отключился, пропущены события | Reconnect: SSE first → `notePathsDelta(sinceVersion)` → merge. Fallback на полный sync если version невалиден |
| Пользователь редактирует файл, пришёл update | `classifyFile` → conflict → badge + Notice |
| Batch commit (10 файлов) | Один Publish с `paths: [10 items]` — одно событие |
| `twoWaySync` выключен | SSE не подключается, livePull недоступен |
| API key отозван | SSE получит ошибку при реконнекте, покажет Notice |
| Plugin unload во время auto-pull | AbortController отменит все fetch-и |
| Несколько syncDir | Отдельные серверы — отдельные SSE подписки |
| `publishField` фильтр | Auto-pull проверяет publishField перед записью файла |
| Файл удалён локально, пришёл update | `classifyFile` → pull (новый файл) → скачать |
| Obsidian закрыт, накопились изменения | При открытии — `notePathsDelta(lastSyncedVersion)` ловит всё |
| Нет данных 60 сек (dead connection) | Health check → reconnect |
| Slow consumer (буфер заполнен) | Событие dropped, залогировано, stats.Dropped++ |
| Получили своё же событие после push | classify по хэшам → `unchanged` → no-op |

## Безопасность

- **Нет авто-push**: пользователь всегда контролирует когда отправлять
- **Конфликты не перезаписываются**: только safe pulls (localHash === lastSyncedHash)
- **publishField защита**: авто-pull не скачивает файлы без publish field
- **API key scope**: подписка использует тот же ключ что и sync
- **Defense in depth**: даже если SSE отправит невалидное событие, classify проверит хэши

## TODO / Будущие улучшения

### Оптимистичная блокировка в pushNotes

**Проблема:** Два человека одновременно редактируют файл → оба делают push → последний перезаписывает изменения первого.

**Решение:** Добавить `expectedHash` в `pushNotes` mutation:

```graphql
input PushNotesNoteContentInput {
  path: String!
  content: String!
  expectedHash: String  # NEW: хэш который клиент ожидает увидеть на сервере
}
```

**Поведение:**
1. Клиент при push передаёт `expectedHash` = тот хэш, который был при последнем pull/sync
2. Сервер проверяет: `currentHash === expectedHash`
3. Если не совпадает → возвращает ошибку `CONFLICT` с актуальным хэшем
4. Клиент показывает конфликт, предлагает сначала pull

**Преимущества:**
- Защита от lost updates (write-write conflicts)
- Атомарная проверка + запись в одной транзакции
- Совместим с текущим classify flow

**Реализация:**
- Миграция: нет, используем существующий `latest_content_hash`
- Backend: проверка в `pushNotes` перед записью
- Frontend: Obsidian plugin передаёт `syncState.files[path]` как `expectedHash`
- Админка: можно передавать `null` (skip проверки) для админских правок

**Приоритет:** Средний. Сценарий редкий (два человека редактируют один файл одновременно), но последствия серьёзные (потеря данных).
