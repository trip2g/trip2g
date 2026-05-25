---
title: Управление пользователями через GraphQL
lang_redirect: "[[en/user/user_management]]"
---

Нужно добавить подписчика, открыть коллеге доступ к подграфу или назначить администратора — без браузера. Все три операции доступны как GraphQL-мутации и рассчитаны на вызов из агента или скрипта.

Требуется административный доступ к API. Настройка — в [[ru/user/graphql]].

### Типичный сценарий

1. Создать заметку с `subgraph: name` во фронтматтере → синхронизировать → подграф появится в системе
2. `createUser(email)` → получить `userId`
3. `allSubgraphs` → получить `subgraphIds`
4. `createUserSubgraphAccess(userId, subgraphIds)` → открыть доступ к контенту
5. При необходимости `createAdmin(userId)` → полные права администратора

### Шаг 1. Создать пользователя

```graphql
mutation {
  admin {
    createUser(input: { email: "user@example.com" }) {
      __typename
      ... on CreateUserPayload {
        user { id email }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

В ответе — `id` нового пользователя. Он нужен во всех последующих мутациях.

### Шаг 2. Получить ID подграфов

Подграфы создаются при синхронизации заметки, в которой объявлен подграф во фронтматтере:

```yaml
---
subgraph: premium
---
```

Или несколько сразу:

```yaml
---
subgraphs: [premium, beta]
---
```

После синхронизации из Obsidian подграф появляется в системе. Получить список подграфов с их ID:

```graphql
{ admin { allSubgraphs { nodes { id name } } } }
```

### Шаг 3. Открыть доступ к подграфу

```graphql
mutation {
  admin {
    createUserSubgraphAccess(input: {
      userId: 6
      subgraphIds: [1]
      expiresAt: null
    }) {
      __typename
      ... on CreateUserSubgraphAccessPayload {
        accesses { id userId subgraphId expiresAt }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

`subgraphIds` — массив: в одном вызове можно открыть доступ к нескольким подграфам. Чтобы доступ был постоянным, передайте `null` в `expiresAt` или не указывайте его вовсе. Для временного доступа — метка времени в формате ISO 8601.

### Шаг 4. Назначить администратора (необязательно)

```graphql
mutation {
  admin {
    createAdmin(input: { userId: 6 }) {
      __typename
      ... on CreateAdminPayload {
        admin { id user { id email } }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

Статус администратора даёт полный доступ к панели управления и всем GraphQL-мутациям. Используйте его для участников команды, которые управляют сайтом, — но не для обычных подписчиков.

### Вспомогательные запросы

Если `id` пользователя или подграфа неизвестен, найдите их заранее.

```graphql
# Найти ID пользователя по email
{ admin { allUsers { nodes { id email } } } }

# Список подграфов с их ID
{ admin { allSubgraphs { nodes { id name } } } }

# Проверить текущие доступы пользователя
{ admin { allUserSubgraphAccesses { nodes { id userId subgraphId expiresAt } } } }
```
