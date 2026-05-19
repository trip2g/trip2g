# Превью лейаута

`/_system/renderlayout` рендерит любой Jet-шаблон без загрузки файлов в ядро системы. Ничего не сохраняется — HTML хранится только в небольшом буфере в памяти. Используйте для отладки правок в шаблонах и быстрого превью до того, как что-либо попадёт в продакшн-vault.

## Для кого

- **Разработчики** — редактируют лейауты локально перед деплоем
- **AI-агенты** — тестируют Jet-шаблоны, читают ошибки компиляции/рантайма из ответа и показывают промежуточный результат человеку, с которым работают
- **Все, кто** хочет увидеть, как заметка выглядит с другим лейаутом

## Быстрый старт

```bash
curl -X POST https://yoursite.com/_system/renderlayout \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "layout": {
      "path": "_layouts/article.html",
      "src": "<html><body><h1>{{ note.Title() }}</h1>{{ note.HTMLString() }}</body></html>"
    },
    "note": { "src": "# Привет\n\nЭто тест." }
  }'
```

Ответ:
```json
{
  "previewID":  "a3f9b2c1",
  "previewURL": "/_system/renderlayout?preview_id=a3f9b2c1",
  "warnings":   {"layout": [], "note": [], "files": []}
}
```

Откройте `previewURL` в браузере — увидите результат рендера.

---

## POST `/_system/renderlayout`

**Авторизация:** заголовок `X-API-Key` или сессия администратора.

### Тело запроса

```json
{
  "layout": {
    "path": "обязательно — серверный путь к шаблону",
    "src":  "опционально — Jet HTML; заменяет серверный файл по path"
  },
  "note": {
    "path": "опционально — путь к заметке на сервере (posts/hello.md)",
    "src":  "опционально — markdown (альтернатива path)"
  },
  "overrideFiles": [
    {"path": "путь/к/компоненту.html", "src": "замена содержимого для include-файлов"}
  ]
}
```

### Поля

| Поле | Обязательно | Описание |
|------|-------------|----------|
| `layout.path` | **да** | Серверный путь к лейауту. Определяет стартовый шаблон рендера. |
| `layout.src` | нет | Inline Jet HTML. Заменяет серверный файл по `layout.path` только для этого рендера. |
| `note.path` | нет | Путь существующей заметки на сервере. Используется как переменная `note` в шаблоне. |
| `note.src` | нет | Inline markdown. Рендерится и используется как заметка. |
| `overrideFiles` | нет | Массив объектов `{path, src}`. Переопределяет любой серверный файл лейаута при разрешении `include`/`extends`. |

`note.path` и `note.src` взаимоисключающие. Если не указано ни то ни другое, шаблон получает пустую заглушку-заметку.

### Ответ

```json
{
  "previewID":  "a3f9b2c1",
  "previewURL": "/_system/renderlayout?preview_id=a3f9b2c1",
  "warnings":   {
    "layout": ["compile: line 5: undefined variable 'note.Tags'"],
    "note":   [],
    "files":  []
  }
}
```

- `warnings` — объект с тремя ключами: `layout`, `note`, `files`. Каждый содержит массив ошибок компиляции и рантайма Jet для соответствующей области. Все массивы пусты = чистый рендер.
- Рендер с ошибками всё равно сохраняется в буфер (частичный HTML доступен по ссылке).
- `previewID` действует пока буфер не перезапишется (по умолчанию: 10 последних рендеров).

---

## GET `/_system/renderlayout`

| URL | Авторизация | Возвращает |
|-----|-------------|------------|
| `?preview_id=xxx` | нет | HTML для данного ID |
| (без параметров) | API key / админ | Последний рендер |
| `?live` | API key / админ | Последний HTML + скрипт авто-перезагрузки |
| `?longpolling&since=N` | нет | `{action, version}` через ≤ 30 с |

### Авто-обновление (`?live`)

Откройте `?live` в браузере. Страница содержит polling-скрипт, который вызывает `?longpolling` и перезагружает её автоматически при новом рендере. В связке с локальным file watcher получаете live preview по мере редактирования.

---

## Комбинации body и сценарии использования

### 1 — Тестировать серверный лейаут с существующей заметкой

```json
{
  "layout": {"path": "_layouts/article.html"},
  "note":   {"path": "posts/hello.md"}
}
```

Проверяем, что уже задеплоенный лейаут корректно рендерит конкретную заметку.

### 2 — Тестировать локальные правки лейаута (ещё не загружен)

```json
{
  "layout": {
    "path": "_layouts/article.html",
    "src":  "...изменённый Jet HTML..."
  },
  "note": {"path": "posts/hello.md"}
}
```

Серверный `_layouts/article.html` заменяется вашим локальным `src` только для этого рендера. Все `include`/`extends` по-прежнему разрешаются с сервера.

### 3 — Полностью inline рендер (без серверных файлов)

```json
{
  "layout": {
    "path": "_layouts/test.html",
    "src":  "<html><body><h1>{{ note.Title() }}</h1>{{ note.HTMLString() }}</body></html>"
  },
  "note": {"src": "# Заголовок\n\nАбзац."}
}
```

Полностью самодостаточный запрос. Подходит для AI-агентов, тестирующих шаблон с нуля.

### 4 — Переопределить компонент (например, `_blocks.html`)

```json
{
  "layout": {"path": "docs/_layouts/index.html"},
  "note":   {"path": "docs/intro.md"},
  "overrideFiles": [
    {"path": "docs/_layouts/_blocks.html", "src": "...новое содержимое blocks..."}
  ]
}
```

`docs/_layouts/index.html` запускается на сервере как есть. Когда он вызывает `include "docs/_layouts/_blocks.html"`, серверная версия заменяется вашим файлом из `overrideFiles`.

### 5 — Тестировать frontmatter и свойства лейаута

```json
{
  "layout": {"path": "_layouts/article.html"},
  "note":   {
    "path": "some/note.md",
    "src":  "---\nlayout: article\ntags: [go, testing]\n---\n# Тест frontmatter"
  }
}
```

Серверный путь заметки используется для контекста; `src` заменяет содержимое и frontmatter.

### 6 — Поймать Jet-ошибки до деплоя

```json
{
  "layout": {
    "path": "_layouts/article.html",
    "src":  "{{ note.NonExistentMethod() }}"
  },
  "note": {"src": "# Тест"}
}
```

В массиве `warnings.layout` ответа появится Jet-ошибка. AI-агент или разработчик читают её прямо из JSON — без браузера.

---

## CLI-инструмент

В репозитории есть `scripts/renderlayout.py` — обёртка, которая автоматически читает API-ключ из `.obsidian/plugins/trip2g/data.json`:

```bash
# запускать из директории vault (где лежит .obsidian/)
python3 ../scripts/renderlayout.py \
  --layout-file _layouts/article.html \
  --note-src "# Привет"

# → http://localhost:8081/_system/renderlayout?preview_id=abc123

# получить HTML сразу
python3 ../scripts/renderlayout.py \
  --layout-src "{{ note.M().Debug() }}" --layout-path "/_debug.html" \
  --note-path /my-note \
  --fetch
```

Аргументы: `--layout-path`, `--layout-file`, `--layout-src`, `--note-path`, `--note-file`, `--note-src`, `--fetch`.
Предупреждения и ошибки — в stderr; код выхода 1 при сбое.

Инструкция для агентов и советы по отладке шаблонов (включая функцию `debug()`) — в `docs/skills/check_templates.md`.

---

## Сценарий работы с AI-агентом

Агент, работающий над лейаутом совместно с человеком, может показывать промежуточный результат мгновенно — без деплоя и без загрузки в vault:

1. Агент вносит правки в лейаут или заметку локально.
2. Агент делает POST на `/_system/renderlayout`. В vault ничего не записывается.
3. Ответ содержит `previewURL` и `warnings`.
4. Если все массивы в `warnings` пусты — агент пишет пользователю: *«Вот промежуточный результат: [ссылка]»*
5. Если в каком-либо массиве `warnings` есть ошибки — агент читает Jet-ошибку, исправляет шаблон и делает POST снова.
6. Пользователь открывает ссылку, видит рендер и даёт обратную связь — до того как хоть один файл был загружен.

Буфер хранит последние 10 рендеров. Ссылки перестают работать после перезапуска сервера или когда буфер заполняется.

---

## Live превью с file watcher

Установите [watchexec](https://github.com/watchexec/watchexec), затем запустите:

```bash
watchexec -e html -- sh -c '
  curl -s -X POST \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
      \"layout\": {
        \"path\": \"_layouts/article.html\",
        \"src\":  $(cat _layouts/article.html | jq -Rs .)
      },
      \"note\": {\"path\": \"posts/hello.md\"}
    }" \
    https://yoursite.com/_system/renderlayout
'
```

Держите `https://yoursite.com/_system/renderlayout?live` открытым в браузере. Страница перезагружается мгновенно после получения POST — long-poll соединение уже ждёт на сервере.

---

## Демо-vault

Тестовые файлы находятся в `docs/demo/`:

| Файл | Назначение |
|------|-----------|
| `docs/demo/_layouts/article.html` | Базовый лейаут статьи |
| `docs/demo/_layouts/with_component.html` | Лейаут с `include` |
| `docs/demo/_layouts/components/header.html` | Включаемый компонент header |
| `docs/demo/hello.md` | Пример заметки |

Используйте как отправную точку при тестировании собственных лейаутов.
