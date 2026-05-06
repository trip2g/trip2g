---
title: Агент-администратор
free: true
lang_redirect: "[[en/user/agent_admin]]"
---

trip2g можно полностью контролировать через AI-агента, используя один API-ключ. Тот же ключ, которым плагин синхронизации Obsidian пушит заметки, авторизует MCP и даёт доступ к мутациям GraphQL — без входа через браузер.

### Как это работает

```
Архив vault Obsidian      API-ключ (уже внутри)
       │                          │
       ▼                          ▼
Агент открывает vault  ──▶  MCP-сервер
                                  │
                          Доступ к контенту
                          (поиск, заметки)
                                  │
                    enable_mcp_admin_tools?
                                  │
                                  ▼
                      graphql_introspection
                      graphql_request
                                  │
                          Мутации администратора
                      (вебхуки, патчи и т.д.)
```

Агенту нужен только архив vault. API-ключ уже внутри настроек плагина синхронизации — никаких дополнительных шагов для пользователя.

### Настройка

1. Создайте API-ключ в **Администрирование → API-ключи**
2. Передайте архив vault с уже прописанным ключом в плагине
3. Включите **MCP admin-инструменты** на ключе, если агенту нужны мутации

### Admin-инструменты

Когда `enable_mcp_admin_tools` включён, в MCP появляются два дополнительных инструмента:

| Инструмент | Что делает |
|------------|-----------|
| `graphql_introspection(pattern)` | Находит операции GraphQL по ключевому слову (regexp). Возвращает совпавшие типы и все типы, на которые они ссылаются — как `grep -A -B` по схеме. |
| `graphql_request(query, variables?)` | Выполняет любой запрос или мутацию с правами администратора. |

Агенты обычно используют их парой: сначала introspect, чтобы найти нужную операцию и её поля ввода, затем request — чтобы выполнить её.

### Пример: применить frontmatter-патч

```
Агент: graphql_introspection({ pattern: "frontmatter" })
→ Возвращает типы FrontmatterPatch и мутацию createFrontmatterPatch
  со всеми полями ввода.

Агент: graphql_request({
  query: "mutation Create($input: CreateFrontmatterPatchInput!) {
    adminMutation { createFrontmatterPatch(input: $input) { ... on CreateFrontmatterPatchPayload { patch { id } } } }
  }",
  variables: { input: { ... } }
})
→ Патч применён.
```

### Пример: настроить вебхук

```
Агент: graphql_introspection({ pattern: "webhook" })
→ Возвращает ChangeWebhook и мутацию createWebhook со структурой ввода.

Агент: graphql_request({
  query: "mutation { adminMutation { createWebhook(input: {...}) { ... } } }"
})
→ Вебхук создан.
```

### Безопасность

- Аутентификация по API-ключу даёт права администратора на весь контент (все заметки и подграфы)
- `graphql_request` с включёнными admin-инструментами может выполнить любую мутацию — обращайтесь с ключом как с root-паролем
- Скомпрометированные ключи отзывайте в **Администрирование → API-ключи**
- Admin-инструменты по умолчанию выключены. Их включение — осознанное решение.
