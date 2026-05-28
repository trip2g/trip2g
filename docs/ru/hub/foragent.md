---
title: "Foragent"
free: true
lang: ru
lang_redirect: "[[en/hub/foragent]]"
---

# Foragent

База знаний со **скилами агентов из открытых источников** ([foragent.ru](https://foragent.ru)),
подключённая к хабу через федерацию. Собраны скилы со всего GitHub — коллекции вроде
**superpowers** и awesome-skills-листов: code review, написание тестов и TDD, деплой
и DevOps, рефакторинг, безопасность, документация. Каждый скил идёт с метаданными,
ссылкой на GitHub и тегами.

Скилы предоставляются **как есть**; часть проверяется командой foragent.ru — сканер
безопасности и policy gate выставляют вердикт (`scanner_status`, `risk_score`) и флаг
`recommended`. Решение об использовании всегда за вами: скилы **не устанавливаются
автоматически**.

Как и Telegram-каналы, это **не trip2g-инстанс**, а отдельная нода, реализующая
протокол MCP-федерации. Сама по себе она недоступна: войти можно только через этот
хаб, у которого есть ключ (прописан в админке trip2g).

## Популярные скилы

Самые заезженные (по ⭐ GitHub):

- **superpowers** (`obra/superpowers`) — brainstorming, написание и выполнение
  планов (writing / executing plans), test-driven development (TDD), systematic
  debugging, code review (requesting / receiving), subagent-driven development,
  git worktrees, verification before completion, dispatching parallel agents,
  using superpowers.
- **openclaw** (`openclaw/openclaw`) — coding agent skill, browser automation и
  скрейпинг.

## Как искать скилы через MCP

Через MCP-эндпоинт хаба (`trip2g.com/_system/mcp`):

1. **Поиск** — `federated_search(kb_id: "foragent", query: "…")`:
   - простой текст: `{"kb_id":"foragent","query":"code review typescript"}`;
   - JSON-DSL в поле `query` для фильтров:
     `{"op":"search","text":"deploy","technology_tags":["docker"],"sort":"stars"}`.
     Поля: `category`, `task_tags`, `technology_tags`, `capability_tags`,
     `workflow_stages`, `role_tags`, `licenses`, `sort` (`stars` | `updated` | `name`),
     `limit` (1–20).
2. **Детали скила** — `federated_note_html(kb_id: "foragent", note_id: "<id из результата>")`:
   полный паспорт (автор, категория, теги, ⭐, лицензия, вердикт сканера,
   рекомендация, ссылка на SKILL.md).
3. **Похожие** — `federated_similar(kb_id: "foragent", note_id: "<id>")`.

В каждом результате: `note_id`, `title`, `url` (GitHub), `category`, теги
(`tasks`, `capabilities`, `technologies`, `roles`), `stars`, статус сканера и флаг
`recommended`. Скилы **не устанавливаются автоматически** — это только каталог для
поиска и оценки.

Источник: [foragent.ru](https://foragent.ru)
