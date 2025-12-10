# Obsidian Sync Plugin

## Overview

Плагин для синхронизации Obsidian vault с сервером trip2g. По умолчанию работает в режиме односторонней синхронизации (push only). Двусторонняя синхронизация включается опционально для импорта TG каналов или серверной автоматизации.

## Настройки плагина

### API URL
URL вашего Trip2g сайта. Пример: `https://yoursite.trip2g.com`

### API Key
API ключ из админ-панели Trip2g.

### Sync folder (Папка синхронизации)
Папка в Obsidian vault для синхронизации. Используйте `/` для синхронизации всего хранилища (все markdown файлы будут синхронизированы).

### Publish fields (Поля публикации)
Фильтр синхронизации по frontmatter полям. Поддерживает список полей через запятую.

**Как работает:**
- Если указаны поля (например, `publish, public`), синхронизируются только файлы, у которых хотя бы одно из этих полей установлено в `true`
- Если поле пустое — синхронизируются все файлы в папке
- Файлы без указанных полей "защищены" — не отправляются на сервер, не перезаписываются при pull, не показываются как конфликты

**Пример frontmatter:**
```yaml
---
publish: true
---
```

**Логика фильтрации:**
| Действие | Поведение |
|----------|-----------|
| `local_only` | push только если есть поле |
| `push` | push только если есть поле |
| `pull` | не скачивать поверх локального файла без поля (защита) |
| `conflict` | не показывать конфликт если локальный файл без поля |
| `local_deleted` | hide на сервере только если файл был "публикуемый" |

Подробнее: https://trip2g.com/docs/onboarding

### Two-way sync (Двусторонняя синхронизация)

Включает загрузку обновлений с сервера. **По умолчанию выключена.**

**Когда включить:**
- Импорт Telegram каналов — сервер создаёт заметки из постов TG канала
- Серверная автоматизация — когда другие системы обновляют файлы на сервере
- Совместная работа — когда несколько пользователей редактируют одни файлы

**Когда выключена (по умолчанию):**
- Только push: локальные изменения отправляются на сервер
- Конфликты разрешаются в пользу локальной версии (автоматически перезаписывают сервер)
- Новые файлы с сервера не скачиваются
- Удаление файлов на сервере игнорируется

**Когда включена:**
- Полная двусторонняя синхронизация
- При конфликте показывается диалог выбора версии
- Новые файлы с сервера скачиваются
- При удалении файла на сервере предлагается удалить локально

## Архитектура

```
┌─────────────────┐     GraphQL API      ┌─────────────────┐
│  Obsidian Vault │ ◄──────────────────► │   trip2g Server │
│                 │                       │                 │
│  - .md files    │     pushNotes         │  - notes table  │
│  - assets       │     hideNotes         │  - assets       │
│                 │     notePaths         │                 │
└─────────────────┘                       └─────────────────┘
        │
        ▼
┌─────────────────┐
│   localStorage  │
│                 │
│  - syncState    │
│  - lastSyncedAt │
│  - file hashes  │
└─────────────────┘
```

## Sync State

Состояние синхронизации хранится в `localStorage` браузера:

```typescript
interface SyncState {
  files: Record<string, string>;  // path → lastSyncedHash
  lastSyncedAt?: number;
}
```

`lastSyncedHash` - SHA-256 хэш содержимого файла на момент последней успешной синхронизации.

## Алгоритм классификации файлов

При синхронизации каждый файл классифицируется на основе трёх хэшей:

| localHash | remoteHash | lastSyncedHash | Action | Описание |
|-----------|------------|----------------|--------|----------|
| A | A | * | `unchanged` | Файлы идентичны |
| A | null | * | `local_only` | Только локально → push |
| null | A | null/empty | `pull` | Новый на сервере → скачать |
| null | A | B | `local_deleted` | Удалён локально → hide на сервере |
| A | B | null | `conflict` | Первая синхронизация, оба существуют |
| A | B | A | `pull` | Локальный не изменён, сервер изменён |
| A | B | B | `push` | Локальный изменён, сервер не изменён |
| A | B | C | `conflict` | Оба изменены |
| A | null | B | `server_deleted` | Удалён на сервере → спросить пользователя |

### Ключевая логика (sync.ts)

```typescript
function classifyFile(localHash, remoteHash, lastSyncedHash): SyncAction {
  // Оба хэша совпадают
  if (localHash === remoteHash) return "unchanged";

  // Только локально
  if (localHash !== null && remoteHash === null) {
    // Был синхронизирован раньше → удалён на сервере
    if (lastSyncedHash) return "server_deleted";
    // Никогда не видели → новый локальный файл
    return "local_only";
  }

  // Только на сервере
  if (localHash === null && remoteHash !== null) {
    // Был синхронизирован раньше → удалён локально
    if (lastSyncedHash) return "local_deleted";
    // Никогда не видели → новый файл с сервера
    return "pull";
  }

  // Первая синхронизация для этого файла
  if (!lastSyncedHash) return "conflict";

  // Локальный не изменён, сервер изменён
  if (localHash === lastSyncedHash) return "pull";

  // Локальный изменён, сервер не изменён
  if (remoteHash === lastSyncedHash) return "push";

  // Оба изменены
  return "conflict";
}
```

## Порядок синхронизации

```
1. Получить список файлов и хэши с сервера (fetchServerHashes)
2. Вычислить хэши локальных файлов
3. Классифицировать все файлы
4. Выполнить действия:
   a. PULL - скачать изменения с сервера + ассеты
   b. CONFLICT - показать UI для разрешения
   c. PUSH - отправить локальные изменения + ассеты
   d. LOCAL_DELETED - скрыть на сервере (hideNotes)
5. Обновить syncState
```

## Badge индикатор

На иконке синхронизации показывается цветной badge когда есть ожидающие изменения:

| Цвет | Состояние |
|------|-----------|
| 🔵 Синий | Есть изменения на сервере (pull) |
| 🟢 Зелёный | Есть локальные изменения (push) |
| 🟠 Оранжевый | Есть и pull, и push |
| 🔴 Красный | Есть конфликты |

Проверка происходит:
- Каждые 60 секунд
- При фокусе на окно Obsidian
- Через 3 секунды после загрузки плагина

Tooltip иконки показывает количество: `Trip2g Sync (↓3 ↑2)`

## Подтверждение загрузки (Push Confirmation)

Перед загрузкой файлов на сервер показывается модальное окно подтверждения со списком файлов.

| Опция | Действие |
|-------|----------|
| Upload | Загрузить файлы на сервер |
| Cancel | Отменить загрузку |
| Don't ask again | Сохранить настройку и больше не показывать подтверждение |

Настройку "Не спрашивать подтверждение" можно изменить в настройках плагина.

## Обработка удалённых на сервере файлов

Когда файл удалён/скрыт на сервере, но существует локально, показывается модальное окно:

| Опция | Действие |
|-------|----------|
| Delete locally | Удалить локальные файлы |
| Keep locally | Оставить файлы локально (не будут загружены повторно) |

## Разрешение конфликтов

При конфликте открывается ConflictView с опциями:

| Опция | Действие |
|-------|----------|
| Keep Local | Push локальную версию на сервер |
| Keep Remote | Заменить локальный файл серверной версией |
| Keep Both | Создать копию `filename (server).md` |
| Skip | Пропустить, конфликт останется до следующей синхронизации |

## Миграция (первая синхронизация)

При первой синхронизации (пустой `syncState`) с конфликтами показывается MigrationModal:

- **Trust Server**: Принять все серверные версии как базу, скачать новые файлы
- **Review Each**: Показать ConflictView для каждого конфликта

## API Endpoints

### GraphQL Queries

```graphql
# Получить хэши всех файлов
query {
  notePaths {
    path: value
    hash: latestContentHash
  }
}

# Получить содержимое файла с ассетами
query($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    path: value
    latestNoteView {
      content
      assetReplaces {
        id    # путь в заметке (image.png)
        url   # URL для скачивания
        hash  # SHA256 хэш
      }
    }
  }
}
```

### GraphQL Mutations

```graphql
# Отправить изменения
mutation PushNotes($input: PushNotesInput!) {
  pushNotes(input: $input) {
    ... on PushNotesPayload {
      notes { id path assets { path sha256Hash } }
    }
  }
}

# Скрыть файлы (при локальном удалении)
mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input) {
    ... on HideNotesPayload { success }
  }
}

# Загрузить asset
mutation($input: UploadNoteAssetInput!) {
  uploadNoteAsset(input: $input) {
    ... on UploadNoteAssetPayload { uploadSkipped }
  }
}
```

## Синхронизация ассетов

### Путь ассетов (absolutePath)

**Важно**: `absolutePath` хранится на сервере **относительно папки синхронизации** (без prefix), чтобы разные пользователи могли синхронизировать в свои папки.

**Пример:**
- User A: sync folder = `notes/`, файл в vault `notes/assets/img.png`
  - При PUSH: отправляет `absolutePath = "assets/img.png"` (без `notes/`)
- User B: sync folder = `/` (root)
  - При PULL: получает `absolutePath = "assets/img.png"`, сохраняет как `assets/img.png`
- User C: sync folder = `my-vault/`
  - При PULL: получает `absolutePath = "assets/img.png"`, сохраняет как `my-vault/assets/img.png`

### При PUSH (локальные изменения → сервер)

После отправки заметки сервер возвращает список ассетов с их хэшами. Плагин:
1. Находит локальный файл через `metadataCache.getFirstLinkpathDest()`
2. Вычисляет SHA256 хэш
3. Если хэш отличается от серверного → загружает файл с **относительным** `absolutePath` (без sync folder prefix)

### При PULL (изменения с сервера → локально)

```
Для каждого ассета из assetReplaces:
├─ Получаем absolutePath с сервера (относительный)
├─ Добавляем sync folder prefix → получаем полный путь в vault
├─ Ищем локальный файл
├─ Если найден:
│   ├─ Вычисляем SHA256 хэш
│   ├─ Hash совпадает → пропускаем ✓
│   └─ Hash отличается → скачиваем с asset.url
└─ Если не найден:
    ├─ Скачиваем с asset.url
    └─ Сохраняем по полному пути (sync folder + absolutePath)
```

### Типы ассетов

```typescript
// При push - сервер сообщает что нужно загрузить
interface NoteAsset {
  path: string;      // путь в заметке (относительный к заметке)
  sha256Hash: string;
}

// При pull - сервер даёт URL для скачивания
interface RemoteAsset {
  id: string;           // путь в заметке (image.png)
  url: string;          // полный URL для скачивания
  hash: string;         // SHA256 хэш для проверки
  absolutePath: string; // путь относительно sync folder (БЕЗ prefix)
}
```

## Структура файлов плагина

```
obsidian-sync/
├── src/
│   ├── main.ts          # Plugin entry, sync orchestration
│   ├── sync.ts          # Classification logic, hashing
│   ├── api.ts           # GraphQL client
│   ├── types.ts         # TypeScript interfaces
│   ├── i18n.ts          # Localization (EN/RU)
│   ├── diff.ts          # LCS diff algorithm
│   └── ui/
│       ├── ConflictView.ts   # Side-by-side diff UI
│       └── ConflictModal.ts  # Migration modal
├── styles.css
└── manifest.json
```

## HTTP запросы и CORS

### GraphQL API
GraphQL запросы используют обычный `fetch` через библиотеку `graphql-request`. Это позволяет видеть запросы в DevTools Network tab для отладки.

### Загрузка ассетов
Для загрузки ассетов используется `requestUrl` из Obsidian API вместо `fetch`. Причина — CORS: ассеты могут храниться на CDN/S3, и браузер блокирует запросы без CORS-заголовков.

```typescript
import { requestUrl } from "obsidian";

// Обычный fetch — заблокируется CORS
const response = await fetch(assetUrl); // ❌ CORS error

// requestUrl — обходит CORS (работает через Node.js в Electron)
const response = await requestUrl({ url: assetUrl }); // ✓
const data = response.arrayBuffer;
```

**Особенности `requestUrl`:**
- Не виден в DevTools Network tab
- Возвращает `{ json, text, arrayBuffer, status, headers }` (не `Response` объект)
- Работает только в Obsidian (Electron), не в браузере

## Edge Cases

### localStorage очищен
- Все `lastSyncedHash` потеряны
- Серверные файлы будут классифицированы как `pull` (новые)
- Может привести к восстановлению удалённых файлов

### Несколько устройств
- Каждое устройство имеет свой `syncState`
- Новые файлы с других устройств будут скачаны (`pull`)
- Удаления синхронизируются через `hideNotes`

### Два пользователя
- User A создаёт файл → push
- User B синхронизирует → pull (новый файл)
- User B удаляет → hideNotes
- User A синхронизирует → файл скрыт, но есть локально → `server_deleted` → спросит пользователя

## Релиз новой версии

```bash
cd obsidian-sync

# 1. Обновить версию в manifest.json
# 2. Собрать плагин
npm run build

# 3. Создать релиз (без префикса v — BRAT не понимает v-префикс)
gh release create 0.1.7 main.js manifest.json styles.css \
  --title "0.1.7" \
  --notes "Release notes here"
```

**Важно:** Имя тега должно быть без префикса `v` (например `0.1.7`, не `v0.1.7`), иначе BRAT не сможет найти релиз.
