# Obsidian Sync — live-pull: дизайн (2026-06-21)

## TL;DR

- **Live-pull — это UX, а не производительность.** `notePaths` сейчас НЕ медленный:
  `latest_content_hash` — сохранённая колонка, выборка всех 2000 ≈ **10.5ms / 80KB**
  (бенч 2026-06-21). Страх «медленный notePaths» устарел (был до precomputed-хэша).
- **Live-pull заменяет таймер, а не запрос.** `notePaths` всё равно нужен для cold-start,
  догона после реконнекта и определения push/conflict (события несут только серверную
  сторону). Поэтому **polling не убираем — делаем реже** (reconcile-бэкстоп ~5-10 мин) +
  SSE для мгновенной реакции. Это ровно то, что ты сказал.
- **Выигрыш:** sub-second свежесть бейджа/контента вместо лага до 60s (важно для
  multi-device / агентов). Трафик idle-WAN тоже падает, но на флаки-связи реконнект =
  полный fetch, так что чистого perf-выигрыша может не быть.
- **Блокер №1 (backend, критичный):** `notebus` падает с `send on closed channel` и
  роняет весь монолит, как только реальный клиент подключается/отключается. **Без этого
  фикса live-pull включать нельзя.**

## Что уже есть / чего нет

| | Состояние |
|---|---|
| Backend подписка `noteChanges` (notebus + resolver + schema) | ✅ готова end-to-end |
| Публикация из commitNotes/pushNotes/updateNotes/hideNotes | ✅ есть (нюанс: gating, см. ниже) |
| Плагин-потребитель (SSE-клиент) | ❌ нет (нет операции в `operations.graphql`, нет `getReader`) |
| Курсор для догона пропущенного (`since`/`afterId`) | ❌ нет; `NoteChangesFilter` = только include/exclude |
| Bus — потеря событий | буфер 64, при переполнении **молча дропает** (`notebus.go:99-101`) |

Событие `NoteUpsertEvent{path, eventType(create|update), versionId, title, noteView, changedHtmlSelectors}`,
`NoteHideEvent{path, pathId}`. **Хэша в событии нет** — удалённый хэш берём из
`noteView.content` (есть в payload) или дотягиваем `notePaths(filter:{paths})`.

## Must-fix backend (до включения live-pull)

| # | Баг | Severity | Фикс |
|---|---|---|---|
| R1 | `notebus.go:78` `close(sub.ch)` вне лока, а `Publish` шлёт в канал после `RUnlock` (`:97-98`) → `send on closed channel` → паника валит монолит. `select/default` НЕ спасает (паника при любом буфере), `recover()` нет. Срабатывает на каждом disconnect во время save. | **CRITICAL** | удалить `close(sub.ch)` — резолвер и так выходит по `ctx.Done()`, канал соберёт GC. ~1 строка |
| R5 | Подписка под `checkapikey` без `canreadnote`; `buildNoteChangeItems` не фильтрует видимость → события (title/url/HTML-форма) текут любому держателю ключа, включая **черновики**, которых нет даже в `AllVisibleNotePaths`. Это хуже, чем notePaths (тот фильтрует `hidden_by`). | HIGH (privacy) | в `buildNoteChangeItems` отбрасывать невидимые заметки (тот же гейт, что `AllVisibleNotePaths`). ~5 строк |
| R4 | `main.go:1277` `if skipWebhooks { return nil }` стоит ДО publish (`:1300-1316`) → синки со `skip_webhooks` не эмитят события. Плагин обычным путём (`pushNotes`+`commitNotes`) флаг НЕ ставит → свои эхо получает (это ок). Касается только агентов со skip_webhooks. | MED | поднять publish-блок выше гейта (webhook-доставка остаётся под гейтом) |

## План по фазам

**Фаза 0 (backend, блокер):** R1 (notebus, ~1 строка) + R5 (видимость, ~5 строк). Без R1 не подключать клиента.

**Фаза 1 — badge-only live-pull (80% пользы, 0 риска для корректности):**
- `subscription NoteChanges` в `operations.graphql`.
- Новый `obsidian-sync/src/sync/LivePullConnection.ts` — pure-JS SSE: `fetch` + `getReader()`
  (graphql-request стримить не умеет), POST `/_system/graphql` `Accept: text/event-stream` +
  `X-API-Key`, reconnect с backoff `[3,6,12,30]s`, health-check 60s (abort→reconnect, если нет
  данных, включая keepalive). Парсинг **построчно** `event:`/`data:` как `assets/ui/sse/sse.ts:95-128`
  (НЕ split по `\n\n` — это в плане было неверно).
- На событие — двигать существующий бейдж (как `checkForPendingChanges`). Один cold-start
  `FetchServerHashes` для сидирования. **Заменить 60s `setInterval`** на reconnect-driven reconcile.
- Бейдж — advisory: даже рассинхрон стрима безопасен, реальный `syncDirectory` классифицирует с нуля.

**Фаза 2 — catch-up + порядок (корректность догона):**
- На каждый connect/reconnect — один `catchUpPull`: ставит `isSyncing` (сериализует против
  событий через существующую очередь `pendingLivePull`), гоняет `classifySync`, выполняет
  **только pulls**, потом дренит очередь. Закрывает gap после оффлайна и дропов буфера.
- Низкочастотный (~5-10 мин) полный reconcile как бэкстоп против потери событий (bus лоссовый).
- Ключи коннекций — составные `apiUrl+path` (не только apiUrl — иначе коллизия двух syncDir).

**Фаза 3 — авто-pull контента (рискованная, опционально):**
- Тянуть только когда `classifyFile === "pull"` (т.е. `localHash === lastSyncedHash` — локальное
  не трогали; либо новый файл без локальной копии). `conflict` — **никогда** не писать, только
  бейдж + Notice. `NoteHideEvent` — через существующий `ServerDeletedModal`, никогда не авто-удалять.
- **КРИТИЧНО: НЕ переиспользовать `executePlan`** — у него безусловные asset-пассы по `unchanged`
  (`execute.ts:122-129` `uploadMissingAssetsForNotes` **пушит на сервер** + триггерит `commitNotes`).
  Авто-pull должен звать `executePulls` + `downloadAssetsForNotes` напрямую (~15 строк, структурно
  не может пушить).
- Эхо своих пушей: классификация по хэшу → `unchanged`. Но классифицировать против **свежего
  in-memory** syncState, не перечитывать из IndexedDB (окно между push HTTP и `saveSyncState`).

## Ключевые правила безопасности

1. Авто-pull пишет файл **только** при `classifyFile === "pull"`. `conflict` → бейдж, ждать ручного синка.
2. Авто-pull строить на `executePulls`/`downloadAssetsForNotes`, **не** `executePlan` (иначе латентный пуш).
3. Bus лоссовый и без курсора → периодический полный reconcile **обязателен** как бэкстоп (не презентовать live-pull как «убрали polling совсем»).
4. Reconnect: стрим открываем первым, `catchUpPull` сериализован с применением событий (общая очередь + `isSyncing`).
5. `NoteHideEvent` → модалка подтверждения, не авто-delete.

## Открытые вопросы
- Делать ли настоящий курсор `notePathsDelta(sinceVersion)` (новая колонка/запрос, как в `obsidian_sse_pulls.md`) — даёт gap-free догон без полного fetch. Отложено на потом; `note_versions.id` — готовый монотонный seq, но запроса «изменения с версии N» нет.
- Серверный keepalive (`transport.SSE{KeepAlivePingInterval}`) — подтвердить, что шлётся ≤30s, иначе health-check будет зря реконнектить каждые 60s.

## Файлы
Backend: `internal/notebus/notebus.go:78`, `internal/graph/note_changes.go:8-35`, `cmd/server/main.go:1277,1300-1316`.
Плагин: новый `obsidian-sync/src/sync/LivePullConnection.ts`; `operations.graphql`; `src/main.ts` (lifecycle, autoPull, catchUpPull, settings, замена 60s таймера); `src/sync/types.ts`, `src/types.ts` (типы события + SyncDir patterns); `src/sync/execute.ts` (executePulls/downloadAssetsForNotes напрямую).

_Источник: воркфлоу-аудит 2026-06-21 (verify + design + tradeoff + adversarial critique). Связано: [[obsidian_sync_2026-06-21]] (общий разбор), [[obsidian_sync_bench_2026-06-21]] (бенч)._
