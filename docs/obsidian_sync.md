# Obsidian Sync Plugin

## Overview

Плагин для двухсторонней синхронизации Obsidian vault с сервером trip2g. Отслеживает изменения локально и на сервере, разрешает конфликты.

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

### Ключевая логика (sync.ts)

```typescript
function classifyFile(localHash, remoteHash, lastSyncedHash): SyncAction {
  // Оба хэша совпадают
  if (localHash === remoteHash) return "unchanged";

  // Только локально
  if (localHash !== null && remoteHash === null) return "local_only";

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
   a. PULL - скачать изменения с сервера
   b. CONFLICT - показать UI для разрешения
   c. PUSH - отправить локальные изменения
   d. LOCAL_DELETED - скрыть на сервере (hideNotes)
5. Обновить syncState
```

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

# Получить содержимое файла
query($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    path: value
    latestNoteView { content }
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
- User A синхронизирует → файл скрыт, но есть локально → `local_only` → push (восстановит)
