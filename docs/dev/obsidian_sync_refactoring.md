# Obsidian Sync Plugin Refactoring Plan

## Problem

Текущий `main.ts` смешивает:
- UI-логику (модалки, badges, settings)
- IO-операции (чтение/запись файлов, API-запросы)
- Бизнес-логику (классификация, разрешение конфликтов)

Это делает код сложным для тестирования и поддержки.

## Solution: Env Interface Pattern (как в Go)

Вынести бизнес-логику в отдельный модуль `src/case/sync.ts` с чистым интерфейсом, который можно легко мокать для тестов.

### Архитектура

```
src/
├── case/
│   └── sync.ts          # Env interface + чистая бизнес-логика
│   └── sync.test.ts     # Unit тесты
├── main.ts              # Obsidian plugin (реализует Env, UI)
└── types.ts             # Shared types
```

## Env Interface

```typescript
// src/case/sync.ts

// ============ Types ============

export interface LocalFile {
  path: string;      // relative to sync folder
  mtime: number;
}

export interface ServerHash {
  path: string;
  hash: string;
}

export interface SyncState {
  files: Record<string, string>;        // path → lastSyncedHash
  mtimes?: Record<string, number>;      // path → mtime (cache validation)
  localHashes?: Record<string, string>; // path → computed hash (performance)
}

export type SyncAction =
  | "unchanged"
  | "push"
  | "pull"
  | "conflict"
  | "local_only"
  | "remote_only"
  | "local_deleted"
  | "server_deleted";

export interface FileClassification {
  path: string;
  action: SyncAction;
  localHash: string | null;
  remoteHash: string | null;
  lastSyncedHash: string | null;
}

export interface SyncPlan {
  classifications: FileClassification[];
  pulls: FileClassification[];
  pushes: FileClassification[];
  conflicts: FileClassification[];
  localDeleted: FileClassification[];
  serverDeleted: FileClassification[];
  unchanged: number;
}

// ============ Env Interface ============

export interface ClassifyEnv {
  // Data retrieval
  getLocalFiles(): Promise<LocalFile[]>;
  getServerHashes(): Promise<ServerHash[]>;
  getSyncState(): SyncState;

  // Operations
  computeHash(content: string): Promise<string>;
  readFileContent(path: string): Promise<string>;
}

// Full sync env (for execute phase)
export interface SyncEnv extends ClassifyEnv {
  // File operations
  writeFile(path: string, content: string): Promise<void>;
  writeBinaryFile(path: string, data: ArrayBuffer): Promise<void>;
  readBinaryFile(path: string): Promise<ArrayBuffer>;
  deleteFile(path: string): Promise<void>;
  createFolder(path: string): Promise<void>;

  // Server operations
  pushNotes(updates: NoteUpdate[], skipCommit: boolean): Promise<PushedNote[]>;
  hideNotes(paths: string[]): Promise<void>;
  fetchNoteContents(paths: string[]): Promise<NoteContent[]>;
  uploadAsset(params: UploadAssetParams): Promise<boolean>;
  commitNotes(): Promise<void>;

  // State
  saveSyncState(state: SyncState): Promise<void>;

  // UI callbacks (можно мокать no-op в тестах)
  onProgress(progress: Progress): void;
  onConflict(conflicts: ConflictInfo[]): Promise<ConflictResolution[]>;
  onAssetConflict(conflicts: AssetConflictInfo[]): Promise<AssetConflictResolution[]>;
  onServerDeleted(paths: string[]): Promise<boolean>;
  confirmPush(paths: string[]): Promise<boolean>;
}
```

## Pure Functions

```typescript
// Классификация одного файла (pure function)
export function classifyFile(
  localHash: string | null,
  remoteHash: string | null,
  lastSyncedHash: string | null
): SyncAction {
  if (localHash === remoteHash) return "unchanged";

  if (localHash !== null && remoteHash === null) {
    return lastSyncedHash ? "server_deleted" : "local_only";
  }

  if (localHash === null && remoteHash !== null) {
    return lastSyncedHash ? "local_deleted" : "remote_only";
  }

  if (!lastSyncedHash) return "conflict";
  if (localHash === lastSyncedHash) return "pull";
  if (remoteHash === lastSyncedHash) return "push";

  return "conflict";
}

// Классификация всех файлов (использует env)
export async function classifySync(env: ClassifyEnv): Promise<SyncPlan> {
  // ... implementation
}

// Выполнение плана (использует полный SyncEnv)
export async function executePlan(env: SyncEnv, plan: SyncPlan): Promise<SyncResult> {
  // ... implementation
}
```

## Testing Strategy

### Framework: Vitest

Выбран за:
- Нативная поддержка ESM и TypeScript
- Быстрый (использует esbuild)
- Jest-совместимый API
- Встроенный mocking

### Setup

```bash
npm install -D vitest
```

```json
// package.json
{
  "scripts": {
    "test": "vitest",
    "test:run": "vitest run",
    "test:coverage": "vitest run --coverage"
  }
}
```

### Test Cases

#### classifyFile (unit tests)

| Local | Remote | LastSynced | Expected |
|-------|--------|------------|----------|
| A | A | * | unchanged |
| A | null | null | local_only |
| A | null | A | server_deleted |
| null | A | null | remote_only |
| null | A | A | local_deleted |
| new | old | old | push |
| old | new | old | pull |
| local | remote | base | conflict |
| local | remote | null | conflict |

#### classifySync (integration tests)

1. **Unchanged files** - mtime + hash cached, no file reads
2. **Local changes** - detected as push
3. **Server changes** - detected as pull
4. **Hash caching** - uses cached hash when mtime unchanged
5. **Locally deleted** - files in syncState but not local
6. **Multiple files** - correct grouping by action

#### executePlan (integration tests)

1. **Execute pulls** - files downloaded and written
2. **Execute pushes** - files read and uploaded
3. **Handle conflicts** - proper resolution flow
4. **Asset sync** - assets uploaded/downloaded
5. **State update** - syncState updated after operations

### Mock Example

```typescript
const createMockEnv = (
  localFiles: Array<{ path: string; mtime: number; content: string }>,
  serverHashes: Array<{ path: string; hash: string }>,
  syncState: SyncState = { files: {} }
): ClassifyEnv => ({
  getLocalFiles: vi.fn().mockResolvedValue(
    localFiles.map(f => ({ path: f.path, mtime: f.mtime }))
  ),
  getServerHashes: vi.fn().mockResolvedValue(serverHashes),
  getSyncState: vi.fn().mockReturnValue(syncState),
  computeHash: vi.fn().mockImplementation(async (content) => `hash:${content}`),
  readFileContent: vi.fn().mockImplementation(async (path) =>
    localFiles.find(f => f.path === path)?.content ?? ''
  ),
});
```

## Migration Plan

### Phase 1: Extract Classification Logic
1. Create `src/case/sync.ts` with types and `classifyFile`, `classifySync`
2. Create `src/case/sync.test.ts` with unit tests
3. Setup Vitest
4. Verify tests pass

### Phase 2: Integrate with main.ts
1. Create adapter in main.ts that implements ClassifyEnv
2. Replace inline classification logic with `classifySync(env)`
3. Verify plugin still works

### Phase 3: Extract Execution Logic
1. Add `SyncEnv` interface methods
2. Implement `executePlan` function
3. Add tests for execution
4. Migrate main.ts to use `executePlan`

### Phase 4: Cleanup
1. Remove duplicate code from main.ts
2. Move remaining types to appropriate files
3. Update documentation

## Key Principles

1. **Separation of Concerns** - Business logic knows nothing about Obsidian API
2. **Dependency Injection** - All IO through Env interface
3. **Pure Functions** - Where possible (classifyFile)
4. **Testability** - Every function can be tested with mocks
5. **Incremental Migration** - Don't break working code, migrate piece by piece

---

## Архитектура после рефакторинга

```
obsidian-sync/src/
├── sync/                 # Platform-agnostic sync module
│   ├── types.ts          # Interfaces (Env, SyncPlan, SyncAction)
│   ├── classify.ts       # classifyFile, classifySync
│   ├── filter.ts         # filterPlan
│   ├── execute.ts        # executePlan
│   ├── classify.test.ts  # Unit tests
│   ├── filter.test.ts    # Unit tests
│   ├── execute.test.ts   # Unit tests
│   └── cli/
│       ├── env.ts        # Node.js реализация Env (fs, fetch)
│       ├── client.ts     # GraphQL client
│       └── cmd.ts        # CLI runner (парсинг args, запуск)
├── env.ts                # ObsidianSyncEnv (Obsidian реализация Env)
├── main.ts               # Плагин (использует sync/)
└── ...
```

---

## Stories

### Фаза A: Новый код (изолированно, 100% тесты)

#### Story 1: Types и classifyFile

**Goal**: Создать типы и pure function классификации одного файла.

**Acceptance Criteria**:
- [ ] AC1: `src/sync/types.ts` с типами `SyncAction`, `FileClassification`, `SyncPlan`, `SyncState`
- [ ] AC2: `src/sync/classify.ts` с функцией `classifyFile(localHash, remoteHash, lastSyncedHash): SyncAction`
- [ ] AC3: `src/sync/classify.test.ts` с 10 unit tests (100% coverage для classifyFile)
- [ ] AC4: Vitest настроен, `npm run test` проходит

**Test Cases**:
| # | localHash | remoteHash | lastSyncedHash | Expected |
|---|-----------|------------|----------------|----------|
| 1 | A | A | * | unchanged |
| 2 | A | null | null | local_only |
| 3 | A | null | A | server_deleted |
| 4 | null | A | null | remote_only |
| 5 | null | A | A | local_deleted |
| 6 | new | old | old | push |
| 7 | old | new | old | pull |
| 8 | A | B | C | conflict |
| 9 | A | B | null | conflict |
| 10 | null | null | * | unchanged (edge case) |

**Tasks**:
1. `npm install -D vitest`
2. Создать `src/sync/types.ts`
3. Создать `src/sync/classify.ts` с `classifyFile`
4. Создать `src/sync/classify.test.ts`
5. Добавить в package.json: `"test": "vitest run"`

---

#### Story 1.5: filterPlan с FilterOptions

**Goal**: Функция фильтрации плана по бизнес-правилам (twoWaySync, publishField).

**Acceptance Criteria**:
- [ ] AC1: Interface `FilterOptions` в `types.ts`
- [ ] AC2: Функция `filterPlan(plan: SyncPlan, options: FilterOptions): SyncPlan` в `filter.ts`
- [ ] AC3: При `twoWaySync: false` — pulls, remoteOnly, serverDeleted игнорируются, conflicts становятся pushes
- [ ] AC4: При `publishField` — фильтрация по callback `hasPublishField(path)`
- [ ] AC5: Unit tests с 100% coverage

**FilterOptions Interface**:
```typescript
interface FilterOptions {
  twoWaySync: boolean;
  // Callback для проверки publishFields (интеграция с Obsidian metadataCache)
  // Возвращает true если файл имеет хотя бы одно из publish полей = true
  hasPublishFields?: (path: string) => boolean;
}
```

**Логика фильтрации (twoWaySync: false)**:
| Original Action | Filtered Action |
|-----------------|-----------------|
| pull | *игнорируется* |
| remote_only | *игнорируется* |
| server_deleted | *игнорируется* |
| conflict | push |
| push | push |
| local_only | local_only (push) |
| local_deleted | local_deleted |
| unchanged | unchanged |

**Логика фильтрации (publishFields)**:
| Action | hasPublishFields=true | hasPublishFields=false |
|--------|---------------------|----------------------|
| push | push | *игнорируется* |
| local_only | local_only | *игнорируется* |
| pull | pull | *игнорируется* (защита) |
| conflict | conflict | *игнорируется* (защита) |
| local_deleted | local_deleted | *игнорируется* |

**Tasks**:
1. Добавить `FilterOptions` в `types.ts`
2. Создать `src/sync/filter.ts` с `filterPlan`
3. Создать `src/sync/filter.test.ts`
4. Убедиться что все тесты проходят

---

#### Story 2: classifySync с ClassifyEnv

**Goal**: Функция классификации всех файлов с dependency injection.

**Acceptance Criteria**:
- [ ] AC1: Interface `ClassifyEnv` в `types.ts`
- [ ] AC2: Функция `classifySync(env: ClassifyEnv): Promise<SyncPlan>`
- [ ] AC3: Кэширование хешей по mtime
- [ ] AC4: Unit tests с mock env (100% coverage)

**ClassifyEnv Interface**:
```typescript
interface ClassifyEnv {
  getLocalFiles(): Promise<LocalFile[]>;
  getServerHashes(): Promise<ServerHash[]>;
  getSyncState(): SyncState;
  computeHash(content: string): Promise<string>;
  readFileContent(path: string): Promise<string>;
}
```

---

#### Story 3: executePlan с SyncEnv

**Goal**: Функция выполнения sync плана.

**Acceptance Criteria**:
- [ ] AC1: Interface `SyncEnv extends ClassifyEnv`
- [ ] AC2: Функция `executePlan(env: SyncEnv, plan: SyncPlan): Promise<SyncResult>`
- [ ] AC3: Обработка pulls, pushes, conflicts, assets
- [ ] AC4: Unit tests с mock env (100% coverage)

**SyncEnv Interface**:
```typescript
interface SyncEnv extends ClassifyEnv {
  // File operations
  writeFile(path: string, content: string): Promise<void>;
  writeBinaryFile(path: string, data: ArrayBuffer): Promise<void>;
  readBinaryFile(path: string): Promise<ArrayBuffer>;
  deleteFile(path: string): Promise<void>;
  createFolder(path: string): Promise<void>;
  fileExists(path: string): Promise<boolean>;

  // Server operations
  pushNotes(updates: NoteUpdate[], skipCommit: boolean): Promise<PushedNote[]>;
  hideNotes(paths: string[]): Promise<void>;
  fetchNoteContents(paths: string[]): Promise<NoteContent[]>;
  fetchNoteAssets(paths: string[]): Promise<NoteAssetInfo[]>;
  uploadAsset(params: UploadAssetParams): Promise<boolean>;
  downloadAsset(url: string): Promise<ArrayBuffer | null>;
  commitNotes(): Promise<void>;

  // Asset operations
  computeBinaryHash(data: ArrayBuffer): Promise<string>;
  resolveAssetPath(assetPath: string, notePath: string): Promise<string | null>;

  // State
  saveSyncState(state: SyncState): Promise<void>;

  // UI callbacks
  onProgress(progress: Progress): void;
  onConflict(conflicts: ConflictInfo[]): Promise<ConflictResolution[]>;
  onAssetConflict(conflicts: AssetConflictInfo[]): Promise<AssetConflictResolution[]>;
  onServerDeleted(paths: string[]): Promise<boolean>;
  confirmPush(paths: string[]): Promise<boolean>;
}
```

---

### Фаза B: Node.js runtime + E2E

#### Story 4: Node.js Env и CLI

**Goal**: Реализация Env для Node.js и CLI для E2E тестирования.

**Acceptance Criteria**:
- [ ] AC1: `src/sync/cli/env.ts` реализует `SyncEnv` через `fs` и `fetch`
- [ ] AC2: `src/sync/cli/cmd.ts` — CLI runner
- [ ] AC3: Можно запустить sync на тестовом vault без Obsidian
- [ ] AC4: E2E тест с реальным API проходит

**Использование**:
```bash
npx ts-node src/sync/cli/cmd.ts --folder ./test-vault --api-url https://... --api-key ...
```

---

### Фаза C: Интеграция в Obsidian

#### Story 5: Интеграция в main.ts

**Goal**: Переключить плагин на новый код.

**Acceptance Criteria**:
- [ ] AC1: `main.ts` импортирует из `sync/`
- [ ] AC2: `syncDirectory` использует `classifySync` и `executePlan`
- [ ] AC3: Manual test в Obsidian — всё работает
- [ ] AC4: `syncOld.ts` удалён

---

#### Story 6: Cleanup

**Goal**: Финальная чистка.

**Acceptance Criteria**:
- [ ] AC1: Удалён дублирующий код из `main.ts`
- [ ] AC2: `docs/obsidian_sync.md` обновлён
- [ ] AC3: Code coverage > 90% для `src/sync/`

---

## Definition of Done

Для Фазы A (Stories 1-3):
1. ✅ 100% test coverage для нового кода
2. ✅ `npm run test` проходит
3. ✅ Код изолирован, `main.ts` не изменён

Для Фазы B (Story 4):
1. ✅ CLI работает с реальным API
2. ✅ E2E тест проходит

Для Фазы C (Stories 5-6):
1. ✅ `npm run build` успешен
2. ✅ Manual test в Obsidian
3. ✅ `syncOld.ts` удалён

---

## Sprint Status

| Story | Phase | Status | Notes |
|-------|-------|--------|-------|
| Story 1: Types + classifyFile | A | DONE | classifyFile + classifySync + 26 tests, 100% mutation |
| Story 1.5: filterPlan | A | DONE | twoWaySync, publishFields + 20 tests, 100% mutation |
| Story 2: executePlan | A | DONE | executePlan + 31 tests, 100% mutation (126 mutants) |
| Story 3: Node.js Env + CLI | B | DONE | NodeEnv + CLI runner (npm run sync) |
| Story 4: Интеграция | C | DONE | ObsidianSyncEnv + main.ts refactor |
| Story 5: Cleanup | C | DONE | syncOld.ts удалён, 101 tests pass |

## Mutation Testing Summary (Stryker)

All core sync logic achieves 100% mutation score:

```
-------------|--------|---------|----------|-----------|------------|----------|----------|
File         | score  | covered | # killed | # timeout | # survived | # no cov | # errors |
-------------|--------|---------|----------|-----------|------------|----------|----------|
classify.ts  | 100.00 |  100.00 |       96 |         0 |          0 |        0 |        0 |
execute.ts   | 100.00 |  100.00 |      126 |         1 |          0 |        0 |        0 |
filter.ts    | 100.00 |  100.00 |       64 |         0 |          0 |        0 |        0 |
-------------|--------|---------|----------|-----------|------------|----------|----------|
Total        | 100.00 |  100.00 |      286 |         1 |          0 |        0 |        0 |
```
