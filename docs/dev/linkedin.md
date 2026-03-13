# LinkedIn Post Publishing

## Статус

Документ в разработке. Анализ: `.omc/plans/linkedin-integration-analysis.md`.

---

## Архитектура (план)

Два уровня, аналогично Telegram:

- **Layer 1** (`internal/case/`) — подготовка данных, enqueue jobs, без внешних API вызовов
- **Layer 2** (`internal/case/backjob/`) — вызовы LinkedIn REST API, запись в DB после успеха

Queue: отдельная `linkedin_api_jobs` с concurrency=1 (rate limit защита).

---

## LinkedIn Apps vs Accounts

Два уровня в DB и admin UI:

- `linkedin_apps` — credentials LinkedIn Developer App (client_id, client_secret). Каждый пользователь регистрирует свой app на `developer.linkedin.com`.
- `linkedin_accounts` — OAuth токены конкретного LinkedIn профиля, привязаны к app.

Один app может использоваться несколькими аккаунтами.

---

## OAuth Flow

Стандартный 3-legged OAuth 2.0. Скопы для личных аккаунтов:

```
openid profile email w_member_social
```

`w_member_social` — auto-approved, не требует LinkedIn partner review.

### Callback URL

Backend должен иметь публичный callback endpoint (аналог Google/GitHub OAuth callback в `internal/googleauth/`):

```
GET /linkedin/oauth/callback?code=...&state=...
```

В dev окружении нужен HTTPS — использовать Caddy proxy или ngrok.

---

## Токены: важные нюансы

### Lifetimes

| Token | Lifetime | Примечание |
|-------|----------|------------|
| access_token | 60 дней | Нужно обновлять проактивно |
| refresh_token | 365 дней | Счётчик от даты выдачи, НЕ от последнего использования |

### Programmatic Refresh Tokens — ВАЖНО

Refresh tokens для `w_member_social` **доступны только если** на LinkedIn Developer App включён продукт **"Programmatic Refresh Tokens"**.

**Что это значит для admin UI:**
- При создании app в поле README/заметке указывать: "Обязательно включи 'Programmatic Refresh Tokens' в Developer Portal"
- Если refresh_token пустой (= app не включил этот product) — access_token истечёт через 60 дней и нужна ручная реавторизация

**Как проверить что продукт включён:**
1. Зайти на `developer.linkedin.com/apps/{app_id}`
2. Вкладка Products
3. Убедиться что "Programmatic Refresh Tokens" добавлен

**В DB:** `linkedin_accounts.refresh_token` может быть пустым строкой — означает что refresh недоступен.

### Cron job: `refreshlinkedintokens`

Аналог `refreshtelegramaccounts`. Запускается ежедневно:

1. Достать все enabled LinkedIn accounts
2. Если `access_token_expires_at < now + 7 days`:
   - Если `refresh_token` не пустой И `refresh_token_expires_at > now` → рефрешить через `POST /oauth/v2/accessToken` с `grant_type=refresh_token`
   - Если refresh недоступен → пометить account `needs_reauth = true`, залогировать
3. Если `refresh_token_expires_at < now + 30 days` → пометить `needs_reauth = true` (скоро нужна ручная реавторизация)

**Admin UI:** Колонка "Истекает через X дней" с цветовой индикацией + кнопка Re-Auth.

---

## API Endpoints

| Operation | Endpoint | Method |
|-----------|----------|--------|
| Create post | `POST https://api.linkedin.com/rest/posts` | POST |
| Edit post | `POST https://api.linkedin.com/rest/posts/{encoded_urn}` + `X-RestLi-Method: PARTIAL_UPDATE` | POST |
| Delete post | `DELETE https://api.linkedin.com/rest/posts/{encoded_urn}` | DELETE |
| Get user info | `GET https://api.linkedin.com/v2/userinfo` | GET |
| Upload image init | `POST https://api.linkedin.com/rest/images?action=initializeUpload` | POST |
| Upload image binary | `PUT {uploadUrl}` | PUT |

### Обязательные заголовки

```
Authorization: Bearer {access_token}
Content-Type: application/json
X-Restli-Protocol-Version: 2.0.0
LinkedIn-Version: 202601   ← текущий YYYYMM
```

### Post ID

Возвращается в response header `x-restli-id`:
```
urn:li:share:6844785523593134080
```

Post URL: `https://www.linkedin.com/feed/update/{post_urn}/`

---

## Content Formatting

### Ограничения

- `commentary` max: **3000 символов** (включая announcement footer)
- "See more" threshold: ~140 chars (mobile), ~210 chars (desktop) — первые 140 символов критичны как hook
- Нет HTML, нет Markdown рендеринга

### Зарезервированные символы

Следующие символы в `commentary` **должны быть экранированы** с `\`:

```
| { } @ [ ] ( ) < > # \ * _ ~
```

Пример: Markdown-буллеты `*` → `\*` в payload.

### Перенос строк

- `\n` — перенос строки
- `\n\n` — пустая строка между абзацами (основной паттерн для читабельности)

### Unicode Bold (опционально)

Настоящий bold на LinkedIn — через Unicode Mathematical Sans Bold (U+1D5D4+):
- `**text**` можно конвертировать в Unicode bold как опциональная фича
- НЕ применять к хэштегам (LinkedIn indexer не читает Unicode bold)
- Accessibility предупреждение: screen readers плохо читают Unicode bold

### Хэштеги

- Plain `#tag` работает в API (LinkedIn нормализует при сохранении)
- Оптимально: 3–5 хэштегов в конце поста
- Не экранировать `#` если он часть `#hashtag` (только если standalone `#`)

---

## Announcement Footer Template

Каждый LinkedIn account имеет свой `announcement_template` (поле в `linkedin_accounts`).

Используем Jet template engine + `nvs` из `internal/templateviews/nvs.go`:

```
{{ .content }}

{% if .future_notes %}
Скоро:
{% for _, fn := range .future_notes %}
→ {{ fn.Title }} ({{ fn.LinkedInPublishAt | formatDate }})
{% endfor %}
{% endif %}

{{ .profile_url }}
```

Доступные переменные:
- `{{ .note }}` — текущая заметка (NoteView: title, permalink, meta, теги и т.д.)
- `{{ .nvs }}` — все заметки (NVS), пользователь может подгружать соседние через `nvs.Map`
- `{{ .content }}` — основной текст (уже сконвертированный, с resolved ссылками)
- `{{ .profile_url }}` — LinkedIn profile URL аккаунта
- `{{ .future_notes }}` — список NoteView заметок, на которые ссылается текущая заметка и которые ещё **не опубликованы** в LinkedIn (scheduled в будущем)

### Механизм анонса (ключевой)

`{{ .future_notes }}` — это ссылки из текущей заметки на будущие посты. Пример:

Заметка А ссылается на заметку Б (запланирована через 3 дня):
```
Скоро:
→ Как я ускорил деплой в 5 раз (20 февраля)
```

Когда заметка Б публикуется:
1. `InsertLinkedInPublishSentPost` записывает post_urn для Б
2. `UpdateLinkedPosts` cascade (аналог Telegram) — находит все inLinks (заметки ссылающиеся на Б)
3. Для каждой запускает `updatelinkedinpublishpost`
4. Конвертер перегенерирует текст: теперь Б есть в `sentMap` → ссылка резолвится в реальный LinkedIn URL
5. В `{{ .future_notes }}` заметка Б больше не появляется (она уже опубликована)
6. `PARTIAL_UPDATE` на пост А в LinkedIn → анонс исчезает, появляется реальная ссылка

Этот механизм **уже реализован в Telegram** (`UpdateLinkedPosts` в `sendtelegramaccountmessage`). Для LinkedIn — полная аналогия.

**Размерное ограничение:** footer обрезает `Content` если суммарный текст > 3000 символов (с многоточием `…`).

---

## Link Resolution

Аналог Telegram link resolution, но проще:

- `sentMap[notePathID] → post_urn`
- Link URL: `https://www.linkedin.com/feed/update/{post_urn}/`
- Нет per-channel variation (в отличие от Telegram где разные URL для разных каналов)
- Нет `-100` prefix нормализации

Ненайденные ссылки (заметка запланирована но не опубликована) → footer с анонсом и датой публикации.

---

## Go Implementation

Нет официального LinkedIn Go SDK. Используем raw `net/http` + `golang.org/x/oauth2`.

Структура:
```
internal/linkedin/
  client.go      -- HTTP client с обязательными headers
  auth.go        -- OAuth exchange, token refresh
  posts.go       -- Create/update/delete posts
  images.go      -- 3-step image upload
  types.go       -- Request/response structs
```

Аналог: `internal/tgtd/` для MTProto клиента.
