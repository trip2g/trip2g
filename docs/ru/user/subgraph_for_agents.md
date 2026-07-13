---
title: "Подграфы для агентов"
free: true
lang_redirect: "[[en/user/subgraph_for_agents]]"
---

Это отправная точка, а не полное руководство — страница будет расти по мере появления новых сценариев для агентов. Пока здесь минимум: как агенту управлять подграфами через GraphQL вместо админки.

### Предпосылка: API-ключ с включённым admin GraphQL

По умолчанию API-ключ даёт только read-only MCP-инструменты. Чтобы агент мог выполнять admin-мутации (создавать подграфы, выдавать доступ — всё, что лежит под `admin { }`), ключу нужен флаг `enable_mcp_admin_tools = true`.

Если вы разворачиваетесь из демо-хранилища, добавьте `?enable_admin_graphql` к ссылке скачивания onboarding-vault — выданный ключ уже получит этот флаг. Иначе включите его вручную в **Админка → API Keys**. Общая настройка авторизации — в [[ru/user/graphql]], названия MCP-инструментов (`graphql_introspection`, `graphql_request`) — в [[ru/user/mcp]].

### Создать подграф

Отдельной мутации «создать подграф» нет — подграф появляется, как только синхронизируется заметка с этой меткой. Агент создаёт его через `updateNotes`:

```graphql
mutation {
  updateNotes(input: {
    changes: [{
      upsert: {
        path: "subgraphs/team-status.md"
        content: "---\nsubgraph: team-status\n---\nВнутренние статус-заметки, доступ только команде."
      }
    }]
  }) {
    __typename
    ... on UpdateNotesSuccessPayload { paths }
    ... on ErrorPayload { message }
  }
}
```

Подграф `team-status` теперь существует. Чтобы получить его `id` для следующего шага, запросите `admin { allSubgraphs { nodes { id name } } }`.

### Выдать доступ пользователю

```graphql
mutation {
  admin {
    createUserSubgraphAccess(input: {
      userId: 6
      subgraphIds: [3]
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

Если пользователя ещё нет, сначала создайте его: `admin { createUser(input: { email: "..." }) }` — полный разбор в [[ru/user/user_management]].

### Смотрите также

- [[ru/user/subgraphs]] — что такое подграфы и как ими управлять из админки
- [[ru/user/user_management]] — полный цикл мутаций для пользователей и подграфов
- [[ru/user/graphql]] — GraphQL-эндпоинт, способы авторизации
- [[ru/user/federation]] — ограничение поиска агента-пира отдельными подграфами
