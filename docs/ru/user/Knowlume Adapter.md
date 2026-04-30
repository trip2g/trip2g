---
free: true
title: Knowlume Adapter
lang_redirect: "[[en/user/knowlume-adapter]]"
---

Knowlume Adapter синхронизирует фрагменты из [Knowlume](https://knowlu.me) в trip2g по расписанию. Каждый фрагмент становится Markdown-заметкой с frontmatter в вашем хранилище.

Подключение занимает 2–3 минуты: укажите URL адаптера в [[cron_webhooks|cron-вебхуке]], задайте расписание — и синхронизация пойдёт автоматически.

### Подключение

1. Перейдите в настройки проекта → Вебхуки по расписанию → Создать.
2. Вставьте URL адаптера в поле webhook:

```
https://knowlumegate.2pub.me/webhook?knowlume_api_key=ВАШ_КЛЮЧ
```

3. Задайте расписание в cron-синтаксисе, например `0 * * * *` — каждый час.
4. Сохраните.

`knowlume_api_key` — API-ключ вашего аккаунта Knowlume. Найти его можно в настройках профиля на knowlu.me.

Готово. После первого срабатывания по расписанию заметки из Knowlume появятся в хранилище.

### Как работает фильтрация (Jsonnet)

В поле **instruction** вебхука пишется Jsonnet-выражение. Адаптер запускает его перед отправкой данных в trip2g. Переменная `input` — уже распарсенный объект ответа с полями `status`, `message`, `changes`. Каждый элемент `changes` — файл с полями `path`, `content` и `meta` (только у фрагментов), где `meta` содержит метаданные Knowlume.

Конструкция `input { changes: [...] }` расширяет объект: сохраняет `status` и `message`, заменяет только `changes`.

**Только фрагменты с воспроизводимостью выше 6:**

```jsonnet
input { changes: [ c for c in input.changes if std.objectHas(c, "meta") && c.meta.knowlume_reproducibility > 6 ] }
```

**Только фрагменты из конкретного проекта:**

```jsonnet
input { changes: [ c for c in input.changes if std.objectHas(c, "meta") && c.meta.knowlume_project_name == "Мой проект" ] }
```

**Только небанальные фрагменты (non_banality > 7) из конкретного источника:**

```jsonnet
input { changes: [ c for c in input.changes if std.objectHas(c, "meta") && c.meta.knowlume_non_banality > 7 && c.meta.knowlume_source_id == "abc123" ] }
```

**Только индексные файлы (без фрагментов):**

```jsonnet
input { changes: [ c for c in input.changes if !std.objectHas(c, "meta") ] }
```

Проверяйте выражения в разделе «Песочница» ниже — без лишних запросов к Knowlume.

### Песочница

Перейдите по URL адаптера в браузере — откроется отладчик:

```
https://knowlumegate.2pub.me/webhook?knowlume_api_key=ВАШ_КЛЮЧ
```

Левая колонка — входной JSON от Knowlume, правая — результат после Jsonnet-трансформации.

Две кнопки:

- **Run** — запросить свежие данные из Knowlume и запустить синхронизацию.
- **Apply** — применить Jsonnet к кешированному ответу без запроса к Knowlume.

Используйте Apply, чтобы быстро проверять Jsonnet-выражения без лишних запросов.

---

### Для AI-агентов

Если агент должен проанализировать или отфильтровать данные адаптера, используйте endpoint:

```
POST https://knowlumegate.2pub.me/webhook/apply?knowlume_api_key=ВАШ_КЛЮЧ
Content-Type: application/json
```

Получить полный JSON без фильтрации:

```json
{}
```

Применить Jsonnet-фильтр:

```json
{
  "jsonnet": "local r = std.parseJson(std.extVar(\"input\")); r { changes: [c for c in r.changes if std.startsWith(c.path, \"knowlume/\")] }"
}
```

Ответ при успехе:

```json
{
  "output": { "status": "ok", "changes": [...] }
}
```

При ошибке в Jsonnet — `{ "error": "..." }` со статусом 400.

Поле `input` опционально. Если не передано, адаптер использует последний кешированный ответ. Передайте произвольный JSON, чтобы протестировать трансформацию на тестовых данных:

```json
{
  "jsonnet": "local r = std.parseJson(std.extVar(\"input\")); r",
  "input": { "status": "ok", "changes": [] }
}
```

---

Смотрите также: [[cron_webhooks|Вебхуки по расписанию]], [[change_webhooks|Вебхуки при изменении заметок]], [[integrations|Интеграции]].
