---
title: "CLI для синхронизации"
slug: cli
free: true
---

Командная утилита для синхронизации markdown-файлов с сервером trip2g. Работает без Obsidian — из терминала или CI/CD.

CLI и Obsidian-плагин используют общее ядро синхронизации.

### Когда нужен CLI

- **CI/CD пайплайны.** Автоматическая публикация документации при push в репозиторий.
- **Несколько репозиториев.** Каждая команда синхронизирует свою папку с разными метаданными.
- **Миграции.** Массовое обновление frontmatter-полей.
- **Скрипты.** Интеграция с другими инструментами.

Подробнее о сценариях использования — в статье про документацию из репозитория.

### Установка

```bash
# Скачать готовый файл
curl -L -o trip2g-sync.mjs \
  https://github.com/trip2g/obsidian-sync/releases/download/0.3.5/trip2g-sync.mjs

# Или собрать из исходников
cd obsidian-sync
npm install
npm run build:cli
# CLI собирается в dist/trip2g-sync.mjs
```

### Использование

```bash
# Базовая синхронизация
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY

# С указанием сервера
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --api-url https://yoursite.com/graphql

# Двусторонняя синхронизация (получить изменения с сервера)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --two-way

# Пробный запуск (показать что произойдёт, без изменений)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --dry-run
```

### Переменные окружения

Вместо аргументов командной строки можно использовать переменные:

```bash
export API_KEY=your_api_key
export ENDPOINT=https://yoursite.com/graphql

node trip2g-sync.mjs --folder ./vault
```

### Параметры

| Параметр | Сокр. | Описание |
|----------|-------|----------|
| `--folder <path>` | `-f` | Папка для синхронизации (обязательно) |
| `--api-url <url>` | `-u` | GraphQL endpoint (по умолчанию: `$ENDPOINT` или `http://localhost:8081/graphql`) |
| `--api-key <key>` | `-k` | API-ключ (по умолчанию: `$API_KEY`) |
| `--two-way` | `-2` | Двусторонняя синхронизация |
| `--conflict-resolution <mode>` | `-c` | Разрешение конфликтов: `local`, `remote`, `skip`, `fail` (по умолчанию: `local`) |
| `--meta <key=value>` | `-m` | Добавить/перезаписать поле frontmatter (можно повторять) |
| `--verbose` | `-v` | Подробный вывод |
| `--dry-run` | `-n` | Показать план без выполнения |
| `--help` | `-h` | Справка |

### Флаг --meta и subgraph

Флаг `--meta` добавляет поля в frontmatter всех синхронизируемых файлов. Главный сценарий — объединение нескольких репозиториев в один портал документации с разграничением доступа.

**Пример: три репозитория → один сайт документации**

```bash
# Репозиторий backend-команды
node trip2g-sync.mjs --folder ./docs --meta subgraph=backend-team

# Репозиторий frontend-команды
node trip2g-sync.mjs --folder ./docs --meta subgraph=frontend-team

# Репозиторий DevOps
node trip2g-sync.mjs --folder ./docs --meta subgraph=devops --meta team=infra
```

Все три репозитория публикуют в один endpoint. Поле `subgraph` маркирует документы — потом можно настроить доступ: backend-команда видит только `subgraph=backend-team`, DevOps видит всё.

**Как работает:**

1. **Файл без frontmatter** — создаётся новый блок:
   ```markdown
   ---
   subgraph: backend-team
   ---
   # Контент
   ```

2. **Файл с frontmatter** — поле добавляется или перезаписывается:
   ```markdown
   ---
   title: API Reference
   subgraph: backend-team    ← добавлено
   ---
   ```

3. **Локальные файлы не меняются** — meta-инъекция происходит только при отправке на сервер.

### Разрешение конфликтов

| Режим | Поведение |
|-------|-----------|
| `local` | Локальная версия побеждает, перезаписывает сервер (по умолчанию) |
| `remote` | Серверная версия побеждает, перезаписывает локальную |
| `skip` | Пропустить конфликтующие файлы |
| `fail` | Остановиться с ошибкой при первом конфликте (для CI) |

### Пример для GitHub Actions

```yaml
on:
  push:
    branches: [main]
    paths: ['docs/**']

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          curl -L -o trip2g-sync.mjs \
            https://github.com/trip2g/obsidian-sync/releases/download/0.3.5/trip2g-sync.mjs
          node trip2g-sync.mjs \
            --folder ./docs \
            --api-url https://docs.yourcompany.com/graphql \
            --api-key ${{ secrets.DOCS_API_KEY }} \
            --meta subgraph=project-name \
            --conflict-resolution fail
```

Подробнее о настройке CI/CD — в статье про документацию из репозитория.

### Состояние синхронизации

CLI создаёт файл `.sync-state.json` в синхронизируемой папке. Там хранятся хеши файлов и timestamps для отслеживания изменений.

Добавьте его в `.gitignore`, если не хотите шарить состояние между машинами.

### Исходный код

[GitHub: trip2g/obsidian-sync](https://github.com/trip2g/obsidian-sync/tree/master/src/sync/cli)
