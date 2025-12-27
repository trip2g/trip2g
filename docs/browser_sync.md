# Browser Sync Module

Модуль для синхронизации локальных markdown файлов с сервером trip2g прямо из браузера.

## Сборка

```bash
cd obsidian-sync
npm install
npm run build:browser
```

Результаты:
- `dist/browser-sync.mjs` — ESM bundle (~30KB)
- `assets/ui/sync/browser-sync.js` — IIFE bundle для MAM (~30KB)

## Подключение

### Standalone (ESM)

```html
<script type="module">
  import { BrowserEnv, configureStorage } from './browser-sync.mjs';
</script>
```

### MAM/$mol

Модуль автоматически доступен как `$.$trip2g_sync`:

```typescript
// В вашем .view.ts
const env = new this.$.$trip2g_sync.BrowserEnv({
  apiUrl: 'https://yoursite.com/graphql',
  apiKey: 'your-api-key',
}, {
  onProgress: (p) => console.log(p),
})

await env.init()
await env.selectDirectory()  // user gesture required
const result = await env.sync()
```

Файлы:
- `assets/ui/sync/sync.ts` — MAM wrapper (require)
- `assets/ui/sync/browser-sync.js` — IIFE bundle
- `assets/ui/sync/browser-sync.bundle.d.ts` — TypeScript типы

## API

### Конфигурация хранилища

```typescript
import { configureStorage } from './browser-sync.mjs';

// Вызвать ДО создания BrowserEnv
configureStorage({
  dbName: 'my-app-sync'  // Имя IndexedDB базы (default: 'trip2g-sync')
});
```

### Создание BrowserEnv

```typescript
import { BrowserEnv, type UICallbacks } from './browser-sync.mjs';

const callbacks: UICallbacks = {
  // Прогресс синхронизации
  onProgress: (progress) => {
    console.log(`${progress.step}: ${progress.current}/${progress.total}`);
    // progress.path - текущий файл (опционально)
  },

  // Конфликты (локальный и серверный файл изменены)
  onConflict: async (conflicts) => {
    // conflicts: Array<{ path, localContent, remoteContent, localHash, remoteHash }>
    // Показать UI выбора
    return conflicts.map(() => 'keep_local'); // или 'keep_remote', 'keep_both', 'skip'
  },

  // Конфликты ассетов
  onAssetConflict: async (conflicts) => {
    // conflicts: Array<{ path, absolutePath, noteId, localHash, remoteHash, remoteUrl }>
    return conflicts.map(() => 'keep_local'); // или 'keep_remote', 'skip'
  },

  // Файлы удалены на сервере
  onServerDeleted: async (paths) => {
    // Спросить пользователя: удалить локально?
    return false; // true = удалить, false = оставить
  },

  // Подтверждение push
  confirmPush: async (paths) => {
    // Показать список файлов для загрузки
    return true; // true = загрузить, false = отменить
  },

  // Логирование
  onLog: (message, level) => {
    console.log(`[${level}] ${message}`);
  }
};

const env = new BrowserEnv({
  apiUrl: 'https://yoursite.com/graphql',
  apiKey: 'your-api-key',
  twoWaySync: false,        // default: false (только push)
  publishField: 'publish',  // опционально: фильтр по frontmatter
}, callbacks);
```

### Методы BrowserEnv

#### Управление директорией

```typescript
// Инициализация (загрузить состояние из IndexedDB)
await env.init();

// Проверить есть ли сохраненная директория с разрешениями
const hasDir = await env.hasStoredDirectory();

// Запросить разрешение для сохраненной директории (требует user gesture)
const granted = await env.requestStoredPermission();

// Выбрать новую директорию (требует user gesture)
const selected = await env.selectDirectory();

// Имя текущей директории
const name = env.getDirectoryName(); // null если не выбрана

// Готов ли env к синхронизации
const ready = env.isReady();

// Очистить директорию и состояние
await env.clearDirectory();
```

#### Синхронизация

```typescript
// Получить план синхронизации (preview)
const plan = await env.getSyncPlan();
console.log('Will push:', plan.pushes.length);
console.log('Will pull:', plan.pulls.length);
console.log('Conflicts:', plan.conflicts.length);

// Выполнить синхронизацию
const result = await env.sync();
console.log('Pushed:', result.pushed);
console.log('Pulled:', result.pulled);
console.log('Errors:', result.errors);
```

## UI Requirements

### 1. Выбор директории

```typescript
// Кнопка "Выбрать папку" - ДОЛЖНА быть в user gesture handler
selectFolderButton.onclick = async () => {
  const selected = await env.selectDirectory();
  if (selected) {
    updateUI(env.getDirectoryName());
  }
};
```

### 2. Восстановление разрешений

При перезагрузке страницы File System Access API требует повторного разрешения:

```typescript
async function initSync() {
  await env.init();

  if (await env.hasStoredDirectory()) {
    // Директория сохранена и разрешение активно
    showSyncButton();
  } else if (await loadDirectoryHandle()) {
    // Директория сохранена, но разрешение потеряно
    showRequestPermissionButton();
  } else {
    // Директория не выбрана
    showSelectFolderButton();
  }
}

// Кнопка запроса разрешения - ДОЛЖНА быть в user gesture handler
requestPermissionButton.onclick = async () => {
  const granted = await env.requestStoredPermission();
  if (granted) {
    showSyncButton();
  }
};
```

### 3. Кнопка синхронизации

```typescript
syncButton.onclick = async () => {
  syncButton.disabled = true;
  try {
    const result = await env.sync();
    showResult(result);
  } catch (e) {
    showError(e.message);
  } finally {
    syncButton.disabled = false;
  }
};
```

### 4. Отображение прогресса

```typescript
const callbacks: UICallbacks = {
  onProgress: (progress) => {
    const percent = Math.round((progress.current / progress.total) * 100);
    progressBar.style.width = `${percent}%`;
    progressLabel.textContent = `${progress.step}: ${progress.path || ''}`;
  }
};
```

### 5. UI для конфликтов

```typescript
const callbacks: UICallbacks = {
  onConflict: async (conflicts) => {
    // Показать модальное окно с diff для каждого конфликта
    const resolutions = [];
    for (const conflict of conflicts) {
      const resolution = await showConflictModal({
        path: conflict.path,
        local: conflict.localContent,
        remote: conflict.remoteContent,
      });
      resolutions.push(resolution);
    }
    return resolutions;
  }
};
```

## Типы

```typescript
// Прогресс
interface Progress {
  step: 'classify' | 'pull' | 'push' | 'upload_asset' | 'download_asset' | 'conflict' | 'commit';
  current: number;
  total: number;
  path?: string;
}

// План синхронизации
interface SyncPlan {
  pushes: FileClassification[];     // Локальные изменения → сервер
  pulls: FileClassification[];      // Серверные изменения → локально
  conflicts: FileClassification[];  // Оба изменены
  localOnly: FileClassification[];  // Только локально (новые)
  remoteOnly: FileClassification[]; // Только на сервере (новые)
  localDeleted: FileClassification[]; // Удалены локально
  serverDeleted: FileClassification[]; // Удалены на сервере
  unchanged: number;
}

// Результат синхронизации
interface SyncResult {
  pulled: number;
  pushed: number;
  conflictsResolved: number;
  assetsUploaded: number;
  assetsDownloaded: number;
  errors: string[];
}

// Разрешение конфликта
type ConflictResolution = 'keep_local' | 'keep_remote' | 'keep_both' | 'skip';

// Разрешение конфликта ассета
type AssetConflictResolution = 'keep_local' | 'keep_remote' | 'skip';
```

## Пример минимальной интеграции

```html
<!DOCTYPE html>
<html>
<head>
  <title>Trip2g Sync</title>
</head>
<body>
  <div id="app">
    <div id="no-folder">
      <button id="select-folder">Выбрать папку</button>
    </div>
    <div id="need-permission" style="display:none">
      <p>Папка: <span id="folder-name"></span></p>
      <button id="request-permission">Разрешить доступ</button>
    </div>
    <div id="ready" style="display:none">
      <p>Папка: <span id="current-folder"></span></p>
      <button id="sync">Синхронизировать</button>
      <button id="change-folder">Сменить папку</button>
      <div id="progress"></div>
      <div id="result"></div>
    </div>
  </div>

  <script type="module">
    import { BrowserEnv, configureStorage, loadDirectoryHandle } from './browser-sync.mjs';

    configureStorage({ dbName: 'my-sync-app' });

    const env = new BrowserEnv({
      apiUrl: 'https://yoursite.com/graphql',
      apiKey: 'YOUR_API_KEY',
      twoWaySync: false,
    }, {
      onProgress: (p) => {
        document.getElementById('progress').textContent =
          `${p.step}: ${p.current}/${p.total} ${p.path || ''}`;
      },
      onLog: (msg, level) => console.log(`[${level}] ${msg}`),
    });

    async function init() {
      await env.init();

      if (await env.hasStoredDirectory()) {
        showReady();
      } else if (await loadDirectoryHandle()) {
        showNeedPermission();
      } else {
        showNoFolder();
      }
    }

    function showNoFolder() {
      document.getElementById('no-folder').style.display = 'block';
      document.getElementById('need-permission').style.display = 'none';
      document.getElementById('ready').style.display = 'none';
    }

    function showNeedPermission() {
      document.getElementById('no-folder').style.display = 'none';
      document.getElementById('need-permission').style.display = 'block';
      document.getElementById('ready').style.display = 'none';
      document.getElementById('folder-name').textContent = env.getDirectoryName() || 'Unknown';
    }

    function showReady() {
      document.getElementById('no-folder').style.display = 'none';
      document.getElementById('need-permission').style.display = 'none';
      document.getElementById('ready').style.display = 'block';
      document.getElementById('current-folder').textContent = env.getDirectoryName();
    }

    document.getElementById('select-folder').onclick = async () => {
      if (await env.selectDirectory()) {
        showReady();
      }
    };

    document.getElementById('request-permission').onclick = async () => {
      if (await env.requestStoredPermission()) {
        showReady();
      }
    };

    document.getElementById('change-folder').onclick = async () => {
      await env.clearDirectory();
      showNoFolder();
    };

    document.getElementById('sync').onclick = async () => {
      const btn = document.getElementById('sync');
      btn.disabled = true;
      try {
        const result = await env.sync();
        document.getElementById('result').textContent =
          `Done! Pushed: ${result.pushed}, Pulled: ${result.pulled}`;
      } catch (e) {
        document.getElementById('result').textContent = `Error: ${e.message}`;
      } finally {
        btn.disabled = false;
      }
    };

    init();
  </script>
</body>
</html>
```

## Browser Support

- Chrome/Edge 86+
- Firefox (через polyfill в browser-fs-access)
- Safari 15.2+ (частичная поддержка)

File System Access API не поддерживается в мобильных браузерах.

## Безопасность

1. **API Key** — не храните в клиентском коде. Используйте:
   - Прокси на бэкенде
   - Short-lived токены
   - OAuth flow

2. **Publish Field** — защита от случайной публикации приватных заметок:
   - На уровне `filterPlan` — фильтрация файлов без поля
   - Defense in depth в `pushNotes` — проверка перед отправкой

3. **IndexedDB** — хранит только handle и sync state, не содержимое файлов
