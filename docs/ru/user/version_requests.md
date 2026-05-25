---
title: Восстановление перезаписанных заметок
free: true
lang_redirect: "[[en/user/version_requests]]"
---

Заметку случайно перезаписали — содержимое пропало, текущая версия неверная. trip2g хранит все версии каждой заметки. Любую из них можно получить двумя GraphQL-запросами через `graphql_request`.

Требуется включённый admin-режим у API-ключа. См. [[ru/user/agent_admin]].

### Шаг 1. Найдите момент перезаписи

Запросите список версий заметки. Версии возвращаются от новых к старым.

```graphql
query {
  admin {
    noteVersionHistory(filter: { path: "папка/моя-заметка.md" }) {
      totalCount
      nodes {
        versionId
        version
        contentLength
        createdAt
      }
    }
  }
}
```

Пример ответа:

```json
{
  "noteVersionHistory": {
    "totalCount": 5,
    "nodes": [
      { "versionId": 204, "version": 5, "contentLength": 312,  "createdAt": "2026-05-25T14:02:00Z" },
      { "versionId": 198, "version": 4, "contentLength": 318,  "createdAt": "2026-05-24T09:15:00Z" },
      { "versionId": 171, "version": 3, "contentLength": 4821, "createdAt": "2026-05-23T11:40:00Z" },
      { "versionId": 155, "version": 2, "contentLength": 4790, "createdAt": "2026-05-22T08:30:00Z" },
      { "versionId": 138, "version": 1, "contentLength": 3102, "createdAt": "2026-05-20T17:10:00Z" }
    ]
  }
}
```

Просматривайте `contentLength` по дате. Падение с 4821 байта (версия 3) до 318 (версия 4) — момент перезаписи. Версия 3 (`versionId: 171`) — последняя нетронутая копия.

### Шаг 2. Получите содержимое

```graphql
query {
  admin {
    noteVersion(versionId: 171) {
      versionId
      path
      version
      content
      createdAt
    }
  }
}
```

Поле `content` содержит исходный markdown. Скопируйте его обратно в Obsidian и нажмите Sync.

### Постраничная загрузка

Если у заметки много версий, используйте `limit` и `offset`:

```graphql
noteVersionHistory(filter: { path: "папка/моя-заметка.md", limit: 20, offset: 40 })
```

Размер страницы по умолчанию — 50 записей. Общее количество версий возвращает поле `totalCount`.
