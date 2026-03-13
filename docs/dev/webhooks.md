# Webhooks Architecture: Why so complex?

Эта система вебхуков спроектирована не просто для *уведомлений* ("что-то случилось"), а для **двусторонней интеграции с AI-агентами**.

Обычные вебхуки (fire-and-forget) хороши для Slack-уведомлений, но плохи для автоматического редактирования контента. Мы строим систему, где агенты могут **читать, править и коммитить** изменения безопасно и эффективно.

## Ключевые отличия от "обычных" вебхуков

| Фича | Зачем нужна (The "Why") |
|------|-------------------------|
| **Синхронный ответ** | **Скорость.** Агент может вернуть исправления (фиксы опечаток, форматирование) прямо в ответе на вебхук. Экономит 3 лишних HTTP-запроса (Auth -> Get -> Commit) и секунды времени. |
| **Защита от рекурсии (`depth`)** | **Безопасность.** Если агент исправит заметку, это вызовет новый вебхук. Без защиты (max_depth) два агента могут зациклить сервер насмерть. Система сама останавливает цепочку. |
| **Short API Tokens (JWT)** | **Security.** Вместо того чтобы давать агенту "ключ от всех дверей" (Master Key), мы даем временный токен (на 1 час) с правами только на *текущие* заметки. Если токен украдут, ущерб минимален. |
| **Подпись HMAC** | **Доверие.** Гарантирует, что данные пришли именно от нас, а не от хакера. Стандарт индустрии (Stripe, GitHub). |

## Сценарии

1.  **AI-Линтер (Sync):**
    *   Пользователь сохраняет пост.
    *   Вебхук летит к Линтеру.
    *   Линтер находит ошибку и *сразу* в ответе возвращает исправленный текст.
    *   Сервер применяет правку.
    *   *Пользователь доволен: "Оно само исправилось!"*

2.  **Генератор дайджестов (Cron + Async):**
    *   Раз в сутки Cron Webhook будит Агента-Журналиста.
    *   Агент получает токен с доступом `read_patterns: ["blog/*"]`.
    *   Агент не спеша читает посты, пишет саммари и пушит новый пост `blog/digest.md`.

## Ещё сценарии

3.  **Индексатор (Change + Chain):**
    *   Линтер (max_depth=1) правит пост, пушит с depth=1.
    *   Индексатор (max_depth=2) видит правку линтера, обновляет поисковый индекс.
    *   Линтер НЕ видит правку индексатора (depth=2 >= max_depth=1). Цепочка остановлена.

4.  **Модератор (Change + Event Filter):**
    *   Webhook с `on_create: true`, `on_update: false`, `on_remove: false`.
    *   Срабатывает только на новые заметки. Проверяет содержимое, помечает или скрывает нежелательное.

5.  **Бэкап (Cron + Async):**
    *   Раз в неделю Cron Webhook будит Бэкап-агента.
    *   Агент получает токен с `read_patterns: ["*"]`, `write_patterns: []` (read-only).
    *   Читает все заметки через API, экспортирует в S3/Git.

## Ключевые решения

| Решение | Почему |
|---------|--------|
| `next_run_at` обновляется атомарно с delivery | Предотвращает дубликаты при крэше между enqueue и update |
| TTL shortapitoken = `max(timeout_seconds, appconfig TTL)` | Токен не истечёт раньше таймаута |
| `ON DELETE CASCADE` убрать из delivery таблиц | Soft delete (`disabled_at`) делает cascade бесполезным. Cleanup cron чистит deliveries |
| `response_schema` — серверная константа | Не хранится в БД, сервер включает в payload при отправке |
| Delivery cleanup: 30 дней | `webhook_delivery_logs` — 7 дней, `deliveries` — 30 дней, `job_statuses` — 30 дней |
| HTTP клиент: `fasthttp.Client` | Проект уже на fasthttp. `DoTimeout` + `AcquireRequest/ReleaseRequest` |
| HTTPS не форсируется, но рекомендуется | HTTP допустим для localhost/dev. Документировать риск для production |
| Batch rollback при `expected_hash` mismatch | Атомарность: все изменения агента применяются или не применяются. Агент получает `previous_error` при retry |

## Тестирование

| Уровень | Инструмент | Что покрывает |
|---------|-----------|---------------|
| **Unit** | Go + `fasthttputil.InMemoryListener` | HMAC, shortapitoken, agent response parsing, write access, delivery job, depth |
| **E2E** | Playwright `request` API (`e2e/webhooks.spec.js`) | Full flow: pushNotes → webhook fires → verify payload/HMAC → agent response → changes applied |

E2E тесты используют debug endpoints (`/debug/test_webhook`, `/debug/wait_all_jobs`) и `pushNotes` через GraphQL (не obsidian-sync CLI). HMAC верификация через Node.js `crypto.createHmac`.

Подробности: [shared_webhooks.md](shared_webhooks.md) → раздел "Тестирование".

### Почему свои debug endpoints, а не готовые решения

Рассматривали [request-baskets](https://github.com/darklynx/request-baskets) и [webhook-tester](https://github.com/tarampampam/webhook-tester), но отказались:
- **Нет ARM-образов** — request-baskets только linux/amd64, а dev-среда на ARM.
- **Мало звезд на github** — оба в плачевном состоянии, без активной поддержки.
- **Избыточная зависимость** — свои endpoints это десяток строк кода, полный контроль над поведением (статус-коды, задержки, кастомные ответы), никаких проблем с совместимостью.

## Документация

Подробные технические спецификации:

*   [**Change Webhooks**](change_webhooks.md) — триггеры на изменение заметок (create/update/remove).
*   [**Cron Webhooks**](cron_webhooks.md) — запуск агентов по расписанию.
*   [**Shared Infrastructure**](shared_webhooks.md) — общие механизмы: Auth (Short Tokens), HMAC подпись, формат ответов, Retry, Debugging, Тестирование.
*   [**Job Statuses**](job_statuses.md) — единый UI для отслеживания всех фоновых задач.
