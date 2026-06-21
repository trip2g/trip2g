# Obsidian Sync — разбор кода (2026-06-21)

Смотрели плагин `obsidian-sync/` (ядро `src/sync/`, три фронтенда: Obsidian, browser, CLI) и серверную часть push/commit/hide + подписку `noteChanges`. Цель — производительность на ~2000 заметок, живые обновления, баги, упрощения.

Состояние на сегодня:

- Серверная подписка `noteChanges` готова end-to-end и даже богаче плана (`versionId`, `title`, `noteView`, `changedHtmlSelectors`). Плагин её **не использует**.
- Delta-sync не реализован, но и **не нужен в том виде, что описан в плане** — глобальная монотонная последовательность уже есть (`note_versions.id`).
- Документы `docs/dev/obsidian_sync.md` и `obsidian_sse_pulls.md` — частично ПЛАН, а не код. Расхождения отмечены ниже.

Все находки уже адверсариально проверены (confidence high, если не указано иное).

---

## TL;DR

- **Главная боль 2k — это не сеть, а мёртвый кэш.** `classifySync` читает `syncState.mtimes`/`localHashes`, но их **никто и никогда не пишет** — поэтому каждый sync и каждый фоновый тик заново читает с диска и SHA-256-хэширует все ~2000 файлов последовательно (`classify.ts:91-109`, write-сайт отсутствует во всём `src/`). Фикс маленький, выигрыш — почти весь локальный CPU/IO на idle.
- **Восстановление ассетов крутится по ВСЕМ неизменённым заметкам на каждом sync** (`execute.ts:120-129`), а `env.fetchNoteAssets` **игнорирует аргумент `paths`** и тянет с сервера весь вотлт через пустой `PushNotes` (`env.ts:255-282`). Idle-sync = O(2000) чтений + полный дамп ассетов.
- **Каждый push/commit делает полный reload всех заметок** на сервере (`pushnotes/resolve.go:83` → `PrepareLatestNotes`); плагин шлёт батчами по 100, поэтому начальный sync 2000 заметок = ~21 полный reload, и все — под одним глобальным мьютексом, который ещё и сериализует git-HTTP.
- **Polling badge тяжёлый**: каждые 60с + на каждый focus тянется полный список `notePaths` (~180 кБ) на syncDir, плюс при старте ещё два full-table запроса (`FetchPublishedUrls`, `FetchAllWarnings`).
- **Живые обновления: backend на 100% готов, клиент на 0%.** Дешёвый путь — добавить SSE-клиент в плагин и точечно тянуть только изменённые пути. Backend менять не нужно.
- **Критический баг сервера**: `notebus.Publish` шлёт в канал после снятия RLock, а `Unsubscribe` его `close()` — гонка `send on closed channel` валит весь монолит (`notebus.go:85-104`).
- **Приватность**: заметка с `publish:false` тихо остаётся опубликованной с устаревшим контентом (`filter.ts:72-78`); подписка `noteChanges` отдаёт черновики любому держателю API-ключа без per-note проверки (`schema.resolvers.go:3231-3242`).
- Три рукописных env-адаптера разъехались: browser шлёт запрос на несуществующее поле `latestContent` (двусторонний pull в браузере не работает), а CLI/browser забыли `skipCommit:true` на загрузке ассетов.

---

## Производительность на ~2000 заметок

Корень не один. Их три слоя, и они складываются: **локальный re-hash → сетевые full-vault payloads → серверный full reload**.

### 1. Локальный слой: мёртвый mtime/hash-кэш

`classify.ts:91-109` читает кэш и пропускает хэширование, если mtime не изменился:

```ts
const cachedMtimes = syncState.mtimes || {};
const cachedLocalHashes = syncState.localHashes || {};
...
if (cachedMtime === file.mtime && cachedHash) {
  localHashes.set(file.path, cachedHash);
} else {
  const content = await env.readFileContent(file.path);
  const hash = await env.computeHash(content);   // ← всегда сюда
}
```

Но запись в `.mtimes`/`.localHashes` **не существует нигде в продакшене** — grep по `src/` находит только эти два чтения, объявления типов (`src/types.ts:34,36`, `src/sync/types.ts:39,40`) и фикстуры тестов. `executePlan`/`saveSyncState` пишут только `syncState.files[...]` (`execute.ts:269`). Значит `cachedMtime`/`cachedHash` всегда `undefined`, `if` всегда ложь, и else-ветка читает+хэширует **каждый файл, каждый sync, последовательно** (`await` внутри `for`).

Это и есть боль №1: и при ручном sync, и при фоновом обновлении badge (`main.ts:317`).

### 2. Сетевой слой: full-vault payloads даже при no-op

- `env.fetchNoteAssets(paths)` игнорирует `paths`: делает `PushNotes({ updates: [] })`, что на сервере возвращает ассеты ВСЕХ заметок (`pushnotes/resolve.go:53-58` → `buildPushedNotes` по `nvs.List` + все layouts), и фильтрует клиентски (`env.ts:255-282`). Вызывается до 3× за sync (`execute.ts:60,72,126`) — самый тяжёлый payload.
- Восстановление ассетов (`execute.ts:120-129`) строит `unchangedPaths` из всех `unchanged && remoteHash !== null` и безусловно зовёт `uploadMissingAssetsForNotes` — то есть на любом синхронизированном вотлте проходит все 2000 заметок.
- Badge-poll тянет полный `FetchServerHashes` (~180 кБ на 2000 строк) каждые 60с и на каждый focus, на syncDir (`main.ts:134-143` → `classify.ts:80-83` → `env.ts:102-110`). На сервере нет `since`/incremental-фильтра (`NotePathsFilter` — только `like`/`search`/`paths`, `schema.graphqls:1408-1424`).
- При старте — ещё два full-table запроса: `FetchPublishedUrls` (`main.ts:146,200-220`) и (по кнопке) `FetchAllWarnings`.

Важно: сам `latestContentHash` **дешёвый** — это сохранённая колонка, не пересчитывается per-request (`generated.go:30044`, `IsResolver:false`; пишется один раз в `insertnote/resolve.go:34,43,79`). Страх «82 кБ × 3 пересчёт хэша» из плана не подтверждается — стоит только payload и частота.

### 3. Серверный слой: полный reload на каждый push

`PushNotes` зовёт `PrepareLatestNotes` независимо от числа изменений (`pushnotes/resolve.go:83`). `PrepareLatestNotes` → `latestNoteLoader.Load` перечитывает всё: `RawNotes` (full content+embedding всех строк, `loader.go:147`), `RawAssets` (`:152`), `loadChunks` (`:324`), `generateSitemap` (`:331`). `NoteCache` экономит парсинг, но DB-чтения, карта ассетов, sitemap и search-hash-цикл — безусловны. Плагин шлёт батчами по 100 (`execute.ts:252-257`), поэтому **2000 заметок = ~20 reload + 1 commit reload**.

### Таблица: проблема → причина → фикс → effort

| Проблема | Причина (file:line) | Минимальный фикс | Effort |
|---|---|---|---|
| Re-hash всех файлов каждый sync | кэш `mtimes`/`localHashes` читается, но не пишется (`classify.ts:91-109`) | в `classifySync` собирать `newMtimes`/`newLocalHashes` и писать в `syncState`, persist в `saveSyncState`; перейти на `app.vault.cachedRead` (`env.ts:132`) | small |
| Серийные чтения при cold cache | `await` чтение+хэш в цикле (`classify.ts:96-109`) | после фикса кэша — bounded `Promise.all` чанками ~32 | small |
| `fetchNoteAssets` тянет весь вотлт | пустой `PushNotes`, клиентский фильтр (`env.ts:255-282`; `cli/env.ts:329-363`; `browser/index.ts:493-555`) | переключить на существующий `FetchNoteAssets({filter:{paths}})` (op уже есть, `operations.graphql:37-47`) | medium |
| Recovery по всем unchanged | gate только `length>0` (`execute.ts:120-129`) | ограничить набор пушнутыми путями или флагом «recovery needed» | small |
| Re-read/hash ассетов без кэша | нет mtime-кэша (`execute.ts:826-873`) | кэшировать хэши ассетов по mtime как заметки | medium |
| Один ассет у N заметок грузится N раз | dedup-ключ `noteId:localPath` (`execute.ts:564`) | хэшировать `localPath` один раз за прогон; сервер контент-адресный | small |
| Полный reload на каждый push-батч | `PrepareLatestNotes` всегда (`pushnotes/resolve.go:83`, `loader.go:147-331`) | инкрементальный reload по changed pathIDs (есть `singleNoteLoaderEnv`); хотя бы skip sitemap на skipCommit | large |
| Глобальный мьютекс сериализует push+git | `LockNoteWrites` вокруг всего reload (`schema.resolvers.go:2628-2631`), тот же лок в git (`gitapi/api.go:279-281`) | держать лок только вокруг записи+swap, или RWMutex; materialize только для receive-pack | medium |
| git materialize на каждый fetch/clone | materialize до dispatch (`gitapi/api.go:279-281`), ~2 процесса/blob (`materialize.go:93-104`) | materialize только для записи; dirty-флаг; batch через `git fast-import` | medium |
| Polling badge: full hashes 60с+focus | нет ETag/delta (`main.ts:134-140`, `schema.resolvers.go:3096`) | revision-counter/ETag или перевести badge на SSE | small |
| Старт: 2 full-table запроса | `FetchPublishedUrls`/`FetchAllWarnings` без фильтра (`main.ts:146,200-220`) | lazy по активному файлу или кэш в localStorage | small |
| CLI/browser: reload на каждый ассет | нет `skipCommit:true` (`cli/env.ts:404-411`, `browser/index.ts:574-581`) | добавить `skipCommit:true` | trivial |
| N+1 `NotePathByID` в after-save | цикл по changed (`main.go:1279-1295`) | один `WHERE id IN (...)` | small |
| O(changes×notes) `GetByPathID` | линейный скан (`note.go:1295-1303`), зовётся per-change | индекс `byPathID` в `NoteViews`, hoist `LatestNoteViews()` из цикла | small |
| ~2000 фантомных update-событий | `insertnote` no-op возвращает ID, push добавляет все (`insertnote/resolve.go:71-73`, `pushnotes/resolve.go:80,97`) | отделять changed pathIDs от всех, публиковать только changed | medium |
| pushNotes отдаёт все заметки на батч | `buildPushedNotes` по всем (`pushnotes/resolve.go:103,191-223`), клиент выбирает `notes` | запрашивать `updated{...}`, не строить `Notes` на skipCommit | small |

---

## Живые обновления (live pull)

### Что уже готово (backend, не трогать)

Подписка `noteChanges` реализована end-to-end и **богаче плана**:

- `internal/notebus/notebus.go` — Bus с glob-фильтрацией на подписчика и drop-on-full-buffer, 7 unit-тестов.
- Union `NoteChangeItem = NoteUpsertEvent | NoteHideEvent` (`schema.graphqls:1954-1981`); событие несёт `versionId`, `title`, `noteView`, `changedHtmlSelectors`.
- Резолвер `schema.resolvers.go:3231-3272` (gate по API-ключу), `buildNoteChangeItems` (`note_changes.go:8-35`).
- Publish из `HandleLatestNotesAfterSave` (`main.go:1286-1302`) и `hidenotes/resolve.go:73-83`; `updateNotes` покрыт транзитивно.

### Чего не хватает в плагине

Плагина-потребителя нет вообще: grep по `EventSource|noteChanges|getReader|text/event-stream|lastSyncedVersion|autoPull` в `obsidian-sync/src` — пусто. Сгенерированный `graphql.ts:2995-2997` знает только тестовый `currentTime`, в `operations.graphql` нет ни одной подписки. `SyncState` — только хэши (`sync/types.ts:35-41`). Смена контента — только через 60с-poll или ручной sync.

### Минимальный путь включить (реюз 100% backend)

1. Добавить `subscription NoteChanges` в `operations.graphql`, перегенерировать SDK.
2. Маленький pure-JS SSE-клиент (`obsidian-sync/src/sync/livepull.ts`): POST на graphql с `Accept: text/event-stream` + `X-Api-Key`, `getReader()`-цикл, reconnect с backoff (graphql-request стримить не умеет).
3. На каждый `NoteUpsertEvent` брать только `path` и тянуть его через **существующий** `FetchNoteContents({filter:{paths}})` + `FetchNoteAssets({filter:{paths}})`, прогонять существующий `classifyFile`, auto-pull только безопасных (`localHash === lastSyncedHash`), конфликты — в badge.
4. Когда SSE подключён — отключить 60с-poll (оставить как fallback при разрыве). Это убирает большинство `FetchServerHashes`, потому что badge/auto-pull реагируют на события, а не диффят 2000 хэшей.

**Важно для надёжности live-pull (две серверные правки):**

- При `skip_webhooks=true` (нормальный режим sync) `HandleLatestNotesAfterSave` возвращается ДО `PublishNoteChanges` (`main.go:1274-1276` vs `1311-1312`) — подписчики не узнают о пушах своего же издателя. Поднять конструкцию `busBatch`+publish выше гейта `skipWebhooks`.
- `updateNotes` hide-only батч не публикует remove (`updatenotes/resolve.go:97-130`) — добавить `PublishNoteChanges` по аналогии с `hidenotes`.

### Delta-sync для cold-start/reconnect

Plan предлагает новую таблицу `sync_version` + колонку `last_changed_version` — **это лишнее**. Глобальная монотонная последовательность уже есть: `note_versions.id integer primary key autoincrement` (migration `20250515071315`), и она уже отдаётся как `NoteUpsertEvent.versionId` (`note_changes.go:20`). Достаточно `notePathsDelta(sinceVersion: Int64!)` (или `changedSinceVersion` в `NotePathsFilter`), который выбирает `note_paths` с `max(note_versions.id) > sinceVersion` + скрытые, и возвращает `currentVersion = max(id)`. Без новой таблицы, колонки и миграции.

---

## Баги

| Severity | Область | file:line | Описание | Фикс | Effort |
|---|---|---|---|---|---|
| critical | notebus | `notebus.go:85-104` (+`:73-79`) | `Publish` снимает RLock до отправки в канал, `Unsubscribe` делает `close(sub.ch)`; гонка `send on closed channel` валит весь монолит. Окно — обычный случай: клиент дисконнектится во время save. | не закрывать канал (subscriber выходит по `ctx.Done`), или держать RLock на всём fan-out (sends non-blocking) | small |
| high | classify/filter | `filter.ts:72-78` | `publish:false` у ранее опубликованной заметки → action `push`, но `!publishable` → тихо дропается, `hideNotes` не зовётся. Заметка остаётся живой с устаревшим контентом навсегда. | при `!publishable && remoteHash !== null` пушить как `local_deleted` (паттерн уже есть в `filter.ts:46-53`) | small |
| high | endpoints | `main.go:1274-1276` | При `skip_webhooks` (нормальный sync) bus не публикуется — live-pull для своих пушей не работает. | поднять publish выше гейта `skipWebhooks` | trivial |
| medium | browser | `browser/index.ts:479` | Запрос `content: latestContent` — поля `latestContent` на сервере нет (`schema.graphqls:1349-1357`). Любой pull/conflict в браузере кидает GraphQL Error. | заменить на `content`; лучше — переиспользовать SDK | trivial |
| medium | notebus | `main.go:1305`, `hidenotes/resolve.go:82` | События публикуются ДО commit транзакции (`handler.go:151-167`). Откат → фантомные create/update/remove у подписчиков. | публиковать после commit (стэшить batch на tx-env, emit в `ReleaseTxEnvInRequest`) | medium |
| medium | endpoints | `updatenotes/resolve.go:97-130` | hide через `updateNotes` не публикует remove (а через `hideNotes` — публикует). Подписчики не видят удаления. | добавить `PublishNoteChanges` в Env и emit remove | small |
| medium | main | `main.ts:134-143,272-275` | Нет re-entrancy guard: 60с-тик и focus могут запустить два полных classify ~2000 файлов параллельно, гонка на badge. | флаг `isChecking` + finally; debounce focus | trivial |
| low | execute | `execute.ts:365-369` | `keep_local` пишет `syncState.files[...]` без проверки ответа сервера (в отличие от `executePushes:262-272`). Сервер молча отверг — конфликт «исчез», push потерян. | сверять с `PushedNote` перед записью | trivial |
| low | execute | `execute.ts:131-143` | Push помечает заметки synced до commit; падение commit роняет исключение мимо `saveSyncState`, URL теряются, состояние неоднозначно. | обернуть push+commit+save так, чтобы persist был только после commit; ошибку commit — в `result.errors` | small |
| low | execute | `execute.ts:379-381` | `keep_both` для файла без расширения: `lastIndexOf('.')===-1` → `ext` = всё имя, `baseName` пустой → ` (server)Makefile`. | guard `dot > 0` | trivial |
| low | notebus | `calculatechangeselectors/resolve.go:19-35` | `changedHtmlSelectors` считается лениво из общей `prevHTML[pathID]`; второй save между Publish и резолвом → дифф против неверного baseline. | считать селекторы eager в момент Publish, или ключевать `prevHTML` по versionId | medium |
| low | browser | `browser/index.ts:261-270` | `walkDirectory` кэширует только `md/html`; бинарные ассеты не видны `fileExistsSync`, `.canvas/.base/.excalidraw` молча не синкаются. | кэшировать все файлы; общий `SYNC_NOTE_EXTENSIONS` | small |
| low | browser | `browser/index.ts:312-324` | `writeFile`/`writeBinaryFile` не обновляют `existsCache` (а `deleteFile` обновляет). Свеже-скачанный файл считается отсутствующим в том же sync. | `existsCache.set(path, true)` после записи | trivial |

---

## Упрощения / «малой кровью»

| Тема | file:line | Суть | Фикс | Effort |
|---|---|---|---|---|
| Три env-адаптера дублируют всё | `env.ts:49-536`; `cli/env.ts:44-664`; `browser/index.ts:61-956` | `pushNotes`/`hideNotes`/`fetch*`/`uploadAsset`/`hasPublishFieldInContent` скопированы 3-4 раза и уже разъехались (баг `latestContent`, разный retry, разный `skipCommit`). | общий `RemoteApi` поверх SDK (работает в Node и браузере) + единый `hasPublishFieldInContent` в `sync/utils.ts` | medium |
| Дублирующийся publish-field парсер | `env.ts:506-535`, `cli/env.ts:634-663`, `browser/index.ts:921-955` | три идентичных regex-YAML парсера | вынести одну экспортируемую функцию | small |
| Два publish-field пути в плагине | `main.ts:373-385` (metadataCache) vs `env.ts:506-535` (regex) | разная семантика truthy (`on`/число проходят фильтр, но падают на defense-check → краш push) | свести к одной функции | small |
| Разные списки расширений заметок | `cli/env.ts:127`, `env.ts:480`, `browser/index.ts:261-262` | CLI: +`.html.json`; Obsidian: без него; browser: только `md/html` | один `isSyncableNote()` | small |
| `skipCommit` забыт в CLI/browser | `cli/env.ts:404-411`, `browser/index.ts:574-581` | каждый ассет триггерит полный `PrepareLatestNotes` | добавить `skipCommit:true` | trivial |
| Разный retry загрузки ассетов | `env.ts:284-302` (backoff), `cli/env.ts:365-384` (tight-loop), `browser/index.ts:557-624` (нет retry) | browser теряет ассет на любом сетевом сбое | общий `withRetry` с backoff | small |
| Per-asset `console.log` в hot path | `execute.ts:484,505,513,521,560,573,577` | спам в консоль/stdout на каждый ассет каждого sync | gate за debug-флагом | trivial |
| Сырые `window.setInterval`/`addEventListener` | `main.ts:134-146,149-155` | два `setTimeout` не очищаются на unload | `registerInterval`/`registerDomEvent` | trivial |
| `NoteViews.Copy()` — shallow | `note.go:1017-1020` | копирует только header, карты общие — латентная гонка, а не perf | переименовать в `Snapshot` или сделать настоящую копию | small |
| Стейл-документ плана | `obsidian_sse_pulls.md:188-198,469-492` | event-struct, резолвер и delta-схема не совпадают с кодом | добавить короткий блок «Status (as of code)» | trivial |

---

## Безопасность

**1. Утечка черновиков через подписку (high).** `NoteChanges` делает один `checkapikey.Resolve(ctx, ..., "note_changes")` и затем `SubscribeNoteChanges` по клиентским globs (`schema.resolvers.go:3231-3242`). Нет `canreadnote` и не применяются read-паттерны токена. Webhook-токен с `WebhookReadPatterns` (`checkapikey/resolve.go:126`) может прислать `includePatterns:["**"]` и стримить `path/title/HTML/permalink` всех заметок, включая неопубликованные (`note_changes.go:14-23`). Схема обещает per-note auth (`schema.graphqls:1994`), кода нет.
Фикс: для shortapitoken — AND его `WebhookReadPatterns` в include-фильтр перед `SubscribeNoteChanges`; для обычных ключей — gate per-change через `canreadnote` в `buildNoteChangeItems`. Effort: medium.

**2. Приватность publish:false (high, см. Баги).** `filter.ts:72-78` — заметка остаётся опубликованной.

**3. Browser publish-field — no-op (low).** `hasPublishFieldSync` всегда `return true` (`browser/index.ts:914-919`); фильтр не исключает неопубликованные, а defense-check в `pushNotes` **кидает** и валит весь батч (`browser/index.ts:370-378`). Утечки нет (push не проходит), но UX сломан и расходится с CLI/Obsidian.

**4. Drop без сигнала клиенту (low).** Буфер bus = 64 (`main.go:1315`), исходящий SSE-канал = 1 (`schema.resolvers.go:3244`). При всплеске bus молча дропает батчи (`notebus.go:97-102`), клиент не узнаёт о пропуске. Фикс: emit «resync needed» при `dropped>0`.

---

## Рекомендуемый порядок действий

### Быстрые победы (small/trivial, высокий impact)

1. **Оживить mtime/hash-кэш** — `classify.ts:91-109` + `saveSyncState`. Убирает почти весь локальный re-hash на idle. Чинит сразу все три фронтенда. (perf #1)
2. **`skipCommit:true` в CLI/browser** — `cli/env.ts:404-411`, `browser/index.ts:574-581`. Один полный reload вместо N. (trivial)
3. **Publish выше гейта `skipWebhooks`** — `main.go:1274-1276`. Без этого live-pull не заработает вообще. (trivial)
4. **Re-entrancy guard в `checkForPendingChanges`** — `main.ts:272-275`. Убирает удвоение нагрузки на slow path. (trivial)
5. **Починить browser `latestContent` → `content`** — `browser/index.ts:479`. (trivial)
6. **Гонка notebus** — `notebus.go:85-104`: не закрывать канал / держать RLock на fan-out. Критично перед включением live-pull. (small)
7. **`publish:false` → hide** — `filter.ts:72-78`. Приватность. (small)
8. **Lazy старт-запросы** — `main.ts:146,200-220`: не тянуть `FetchPublishedUrls`/`FetchAllWarnings` по всему вотлту. (small)

### Крупнее (medium/large, фундамент)

9. **Live-pull клиент** — новый `livepull.ts` + `operations.graphql` + codegen + wiring в `main.ts`. Реюз 100% backend. Снимает большинство `FetchServerHashes` и даёт реальные live-обновления. (medium, наивысший impact для цели №2)
10. **`fetchNoteAssets` на `FetchNoteAssets({filter:{paths}})`** — `env.ts:255-282` и оба клона. Убирает самый тяжёлый payload. (medium)
11. **Восстановление ассетов только по changed** — `execute.ts:120-129`. (small, но завязано на #1)
12. **Delta-sync на `note_versions.id`** — `notePathsDelta(sinceVersion)`, без новой таблицы/миграции. Для cold-start/reconnect/manual. (medium)
13. **Инкрементальный reload на push** — `pushnotes/resolve.go:83`, `loader.go`. Самый дорогой, но снимает «21 reload на sync». (large)
14. **Лок только вокруг записи / RWMutex + materialize только на receive-pack** — `schema.resolvers.go:2628-2631`, `gitapi/api.go:279-281`. (medium)
15. **Per-change auth подписки** — `schema.resolvers.go:3231-3242`. Перед публичным использованием live-pull. (medium)
16. **Indexed `GetByPathID` + batch `NotePathByID`** — `note.go:1295-1303`, `main.go:1279-1295`. (small каждое)

### DRY-рефактор (когда дойдут руки)

17. Общий `RemoteApi` + единые `hasPublishFieldInContent`/`isSyncableNote`/`withRetry` в `sync/utils.ts` — закрывает целый класс drift-багов (#latestContent, retry, расширения, publish-field). (medium)

---

## Что НЕ нашли проблем

- **Чистый классификатор** `classify.ts:12-69` — все комбинации (local, remote, lastSynced) маппятся в верный action, взаимоисключение держится, нет data-loss/resurrection. Хорошо покрыт тестами.
- **`exclude.ts`** — glob-матчинг корректен; единственное отклонение от gitignore (`**/x` не матчит корень, `exclude.ts:40-42`) — low, opt-in, не data-loss.
- **`resolve.ts`** — чистый, корректный, хорошо протестирован.
- **Порядок шагов `execute.ts`** (pull → assets → server-deleted → conflicts → push → hide → sync-assets → commit → save) — здравый, happy-path верный; нормальный push-путь защищает `syncState` ответом сервера (`execute.ts:262-272`).
- **Серверный `latestContentHash`** — сохранён, не пересчитывается per-request (`generated.go:30044`); `FetchServerHashes` = один indexed scan (`queries.read.sql.go:900-904`). Никакого дорогого хэширования на poll нет.
- **Подписка `noteChanges`** — реализована end-to-end, богаче плана, 7 unit-тестов. Backend готов, переделывать не нужно.
- **`remote_only`** (`filter.ts:103-110`) — намеренно игнорирует publishField (это всегда download, не push), приватные данные не утекают; поведение совпадает с тестом.
