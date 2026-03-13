# Telegram Publish Through User Accounts

## Overview

Параллельный пайплайн публикации заметок через Telegram user accounts (MTProto) вместо ботов (Bot API). Позволяет использовать Premium-аккаунты для публикации длинных постов.

## Architecture

### Current Bot Pipeline
```
Note → telegram_publish_tags → telegram_publish_chats → tg_bot_chats → Bot API
                                                                ↓
                                            telegram_publish_sent_messages
```

### New Account Pipeline (parallel)
```
Note → telegram_publish_tags → telegram_publish_account_chats → telegram_accounts → MTProto
                                                                        ↓
                                              telegram_publish_sent_account_messages
```

**Important**: Один тег может быть привязан и к bot-чату, и к account-чату. Если пользователь так настроит — это его решение.

## Database Schema

### New Tables

#### telegram_accounts
```sql
create table telegram_accounts (
  id integer primary key autoincrement,
  phone text not null unique,
  session_data text not null,          -- AES-256-GCM encrypted MTProto session (base64)
  display_name text not null default '', -- default: [first_name, last_name, username].join(" ")
  is_premium integer not null default 0 check (is_premium in (0, 1)),
  enabled integer not null default 1 check (enabled in (0, 1)),
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);
```

#### telegram_publish_account_chats
```sql
create table telegram_publish_account_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,   -- telegram's chat_id (not our internal id)
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (account_id, telegram_chat_id, tag_id)
);
```

#### telegram_publish_account_instant_chats
```sql
create table telegram_publish_account_instant_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (account_id, telegram_chat_id, tag_id)
);
```

#### telegram_publish_sent_account_messages
```sql
create table telegram_publish_sent_account_messages (
  note_path_id integer not null references note_paths(id) on delete restrict,
  account_id integer not null references telegram_accounts(id) on delete restrict,
  telegram_chat_id integer not null,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0 check (instant in (0, 1)),
  content_hash text not null default '',
  content text not null default '',
  post_type text not null default 'text'
);

create unique index idx_telegram_publish_sent_account_messages_unique
  on telegram_publish_sent_account_messages(note_path_id, account_id, telegram_chat_id)
  where instant = 0;

create index idx_telegram_publish_sent_account_messages_account_id
  on telegram_publish_sent_account_messages(account_id);

create index idx_telegram_publish_sent_account_messages_note_path_id
  on telegram_publish_sent_account_messages(note_path_id);
```

## Files to Create

### MTProto Client (internal/tgtd)

Весь код связанный с gotd/td находится в `internal/tgtd/`. Пример использования см. `cmd/channelexport/main.go`.

```
internal/tgtd/
├── client.go      -- MTProto client wrapper
├── auth.go        -- Auth manager for 2FA flow
└── session.go     -- Session storage/encryption
```

### Case Files (duplicate bot logic)

| Bot File | Account File |
|----------|--------------|
| `internal/case/sendtelegrampublishpost/` | `internal/case/sendtelegramaccountpublishpost/` |
| `internal/case/updatetelegrampublishpost/` | `internal/case/updatetelegramaccountpublishpost/` |
| `internal/case/backjob/sendtelegrammessage/` | `internal/case/backjob/sendtelegramaccountmessage/` |
| `internal/case/backjob/sendtelegrampost/` | `internal/case/backjob/sendtelegramaccountpost/` |
| `internal/case/backjob/updatetelegrammessage/` | `internal/case/backjob/updatetelegramaccountmessage/` |
| `internal/case/backjob/updatetelegrampost/` | `internal/case/backjob/updatetelegramaccountpost/` |

### Cronjob Extension

Модифицировать `internal/case/cronjob/sendscheduledtelegrampublishposts/resolve.go`:

```go
// Текущая логика:
// 1. Получить список заметок на публикацию
// 2. Для каждой заметки: EnqueueSendTelegramPost()

// Новая логика:
// 1. Получить список заметок на публикацию
// 2. Для каждой заметки:
//    a. EnqueueSendTelegramPost()       -- bot pipeline (existing)
//    b. EnqueueSendTelegramAccountPost() -- account pipeline (new)
// 3. Аналогично для update:
//    a. EnqueueUpdateTelegramPost()
//    b. EnqueueUpdateTelegramAccountPost()
```

Каждый pipeline сам определяет, есть ли у заметки чаты для публикации (по тегам).

## Admin API

### GraphQL Schema

```graphql
# Types
type AdminTelegramAccount @goModel(model: "trip2g/internal/db.TelegramAccount") {
  id: Int64!
  phone: String!
  displayName: String!
  isPremium: Boolean!
  enabled: Boolean!
  createdAt: Time!
}

type AdminTelegramAccountsConnection {
  nodes: [AdminTelegramAccount!]! @goField(forceResolver: true)
}

# Live chat info from Telegram API (not stored in DB)
type AdminTelegramAccountChat {
  telegramChatId: String!
  chatTitle: String!
  chatType: String!
}

type AdminTelegramAccountChatsConnection {
  nodes: [AdminTelegramAccountChat!]! @goField(forceResolver: true)
}

type AdminTelegramAccountAuthState {
  phone: String!
  state: AdminTelegramAccountAuthStateEnum!
  passwordHint: String  # hint for 2FA password if needed
}

enum AdminTelegramAccountAuthStateEnum {
  WAITING_FOR_CODE
  WAITING_FOR_PASSWORD
  AUTHORIZED
  ERROR
}

# Queries (under AdminQuery)
extend type AdminQuery {
  telegramAccounts: AdminTelegramAccountsConnection!
  telegramAccountChats(accountId: Int64!): AdminTelegramAccountChatsConnection!
}

# Mutations (under AdminMutation)
extend type AdminMutation {
  # Step 1: Start auth, sends code to phone
  startTelegramAccountAuth(input: AdminStartTelegramAccountAuthInput!): AdminStartTelegramAccountAuthOrErrorPayload!

  # Step 2: Complete auth with code (and optional 2FA password)
  completeTelegramAccountAuth(input: AdminCompleteTelegramAccountAuthInput!): AdminCompleteTelegramAccountAuthOrErrorPayload!

  # Cancel pending auth
  cancelTelegramAccountAuth(input: AdminCancelTelegramAccountAuthInput!): AdminCancelTelegramAccountAuthOrErrorPayload!

  # Manage existing accounts
  updateTelegramAccount(input: AdminUpdateTelegramAccountInput!): AdminUpdateTelegramAccountOrErrorPayload!
  deleteTelegramAccount(input: AdminDeleteTelegramAccountInput!): AdminDeleteTelegramAccountOrErrorPayload!

  # Set tags for account chat (replaces all existing tags)
  setTelegramAccountChatPublishTags(input: AdminSetTelegramAccountChatPublishTagsInput!): AdminSetTelegramAccountChatPublishTagsOrErrorPayload!
  setTelegramAccountChatPublishInstantTags(input: AdminSetTelegramAccountChatPublishInstantTagsInput!): AdminSetTelegramAccountChatPublishInstantTagsOrErrorPayload!
}

# Inputs
input AdminStartTelegramAccountAuthInput {
  phone: String!
}

input AdminCompleteTelegramAccountAuthInput {
  phone: String!
  code: String!
  password: String  # optional, for 2FA
}

input AdminCancelTelegramAccountAuthInput {
  phone: String!
}

input AdminUpdateTelegramAccountInput {
  id: Int64!
  displayName: String
  enabled: Boolean
}

input AdminDeleteTelegramAccountInput {
  id: Int64!
}

input AdminSetTelegramAccountChatPublishTagsInput {
  accountId: Int64!
  telegramChatId: String!
  tagIds: [Int64!]!  # empty array = remove all tags
}

input AdminSetTelegramAccountChatPublishInstantTagsInput {
  accountId: Int64!
  telegramChatId: String!
  tagIds: [Int64!]!  # empty array = remove all tags
}

# Payloads
type AdminStartTelegramAccountAuthPayload {
  authState: AdminTelegramAccountAuthState!
}
union AdminStartTelegramAccountAuthOrErrorPayload = AdminStartTelegramAccountAuthPayload | ErrorPayload

type AdminCompleteTelegramAccountAuthPayload {
  account: AdminTelegramAccount!
}
union AdminCompleteTelegramAccountAuthOrErrorPayload = AdminCompleteTelegramAccountAuthPayload | ErrorPayload

type AdminCancelTelegramAccountAuthPayload {
  success: Boolean!
}
union AdminCancelTelegramAccountAuthOrErrorPayload = AdminCancelTelegramAccountAuthPayload | ErrorPayload

type AdminUpdateTelegramAccountPayload {
  account: AdminTelegramAccount!
}
union AdminUpdateTelegramAccountOrErrorPayload = AdminUpdateTelegramAccountPayload | ErrorPayload

type AdminDeleteTelegramAccountPayload {
  success: Boolean!
}
union AdminDeleteTelegramAccountOrErrorPayload = AdminDeleteTelegramAccountPayload | ErrorPayload

type AdminSetTelegramAccountChatPublishTagsPayload {
  success: Boolean!
}
union AdminSetTelegramAccountChatPublishTagsOrErrorPayload = AdminSetTelegramAccountChatPublishTagsPayload | ErrorPayload

type AdminSetTelegramAccountChatPublishInstantTagsPayload {
  success: Boolean!
}
union AdminSetTelegramAccountChatPublishInstantTagsOrErrorPayload = AdminSetTelegramAccountChatPublishInstantTagsPayload | ErrorPayload
```

### Auth Flow Implementation

#### In-memory Auth Manager (single instance by design)

```go
// internal/tgtd/auth.go

type PendingAuth struct {
    Phone        string
    Client       *telegram.Client  // gotd client
    State        AuthState
    PasswordHint string
    ExpiresAt    time.Time
}

type AuthManager struct {
    mu      sync.Mutex
    pending map[string]*PendingAuth  // phone -> pending auth
    apiID   int
    apiHash string
}

func NewAuthManager(apiID int, apiHash string) *AuthManager {
    m := &AuthManager{
        pending: make(map[string]*PendingAuth),
        apiID:   apiID,
        apiHash: apiHash,
    }
    go m.cleanupLoop()
    return m
}

func (m *AuthManager) StartAuth(ctx context.Context, phone string) (*PendingAuth, error) {
    // 1. Create new gotd client with in-memory session
    // 2. Run client.Run() in goroutine
    // 3. Send code request via client.Auth().SendCode()
    // 4. Store in pending map with 10min expiry
    // 5. Return state (WAITING_FOR_CODE)
}

func (m *AuthManager) CompleteAuth(ctx context.Context, phone, code, password string) ([]byte, *tg.User, error) {
    // 1. Get pending auth
    // 2. Submit code via client.Auth().SignIn()
    // 3. If 2FA required (ErrPasswordAuthNeeded), submit password
    // 4. Export session data
    // 5. Get user info for display_name: [FirstName, LastName, Username].join(" ")
    // 6. Remove from pending map
    // 7. Return session bytes and user info
}

func (m *AuthManager) CancelAuth(phone string) error {
    // 1. Get pending auth
    // 2. Close client
    // 3. Remove from pending map
}

func (m *AuthManager) cleanupLoop() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        m.mu.Lock()
        now := time.Now()
        for phone, auth := range m.pending {
            if now.After(auth.ExpiresAt) {
                auth.Client.Stop()
                delete(m.pending, phone)
            }
        }
        m.mu.Unlock()
    }
}
```

## Frontend API

### Queries

#### Список аккаунтов

```graphql
query {
  admin {
    allTelegramAccounts {
      nodes {
        id
        phone
        displayName
        isPremium
        enabled
        createdAt
      }
    }
  }
}
```

#### Список чатов аккаунта

```graphql
query GetAccountChats($accountId: Int64!) {
  admin {
    telegramAccountChats(accountId: $accountId) {
      nodes {
        telegramChatId
        chatTitle
        chatType
        publishTags { id, name }
        publishInstantTags { id, name }
      }
    }
  }
}
```

### Mutations

#### 1. Начать авторизацию

Отправляет код на телефон.

```graphql
mutation {
  admin {
    startTelegramAccountAuth(input: {
      phone: "+79991234567"
      apiId: 12345678
      apiHash: "abcdef0123456789abcdef0123456789"
    }) {
      ... on AdminStartTelegramAccountAuthPayload {
        authState { phone, state, passwordHint }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

#### 2. Завершить авторизацию

Без 2FA:
```graphql
mutation {
  admin {
    completeTelegramAccountAuth(input: {
      phone: "+79991234567"
      code: "12345"
    }) {
      ... on AdminCompleteTelegramAccountAuthPayload {
        account { id, phone, displayName, isPremium }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

С 2FA (если вернулся error "2FA password required"):
```graphql
mutation {
  admin {
    completeTelegramAccountAuth(input: {
      phone: "+79991234567"
      code: "12345"
      password: "mypassword"
    }) {
      ... on AdminCompleteTelegramAccountAuthPayload {
        account { id, phone, displayName, isPremium }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

#### 3. Отменить авторизацию

Вызывать при закрытии модалки авторизации.

```graphql
mutation {
  admin {
    cancelTelegramAccountAuth(input: { phone: "+79991234567" }) {
      ... on AdminCancelTelegramAccountAuthPayload { success }
      ... on ErrorPayload { message }
    }
  }
}
```

#### 4. Обновить аккаунт

```graphql
mutation {
  admin {
    updateTelegramAccount(input: {
      id: 1
      displayName: "New Name"
      enabled: true
    }) {
      ... on AdminUpdateTelegramAccountPayload {
        account { id, displayName, enabled }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

#### 5. Удалить аккаунт

```graphql
mutation {
  admin {
    deleteTelegramAccount(input: { id: 1 }) {
      ... on AdminDeleteTelegramAccountPayload { success }
      ... on ErrorPayload { message }
    }
  }
}
```

#### 6. Установить publish теги для чата

```graphql
mutation {
  admin {
    setTelegramAccountChatPublishTags(input: {
      accountId: 1
      telegramChatId: "-1001234567890"
      tagIds: [1, 2, 3]
    }) {
      ... on AdminSetTelegramAccountChatPublishTagsPayload { success }
      ... on ErrorPayload { message }
    }
  }
}
```

Передать `tagIds: []` чтобы удалить все теги.

#### 7. Установить instant теги для чата

```graphql
mutation {
  admin {
    setTelegramAccountChatPublishInstantTags(input: {
      accountId: 1
      telegramChatId: "-1001234567890"
      tagIds: [4]
    }) {
      ... on AdminSetTelegramAccountChatPublishInstantTagsPayload { success }
      ... on ErrorPayload { message }
    }
  }
}
```

### Обработка ошибок

| Error Message | Действие |
|---------------|----------|
| `"2FA password required"` | Показать форму ввода пароля, повторить completeTelegramAccountAuth с password |
| `"No pending authentication for phone"` | Сессия истекла (10 мин), начать заново |
| `"Invalid password"` | Неверный пароль 2FA |
| `"Invalid code"` | Неверный код |
| `"sign up required"` | Аккаунт не зарегистрирован в Telegram |

## Implementation Plan

### PR 1: Database + Admin API ✅

**Scope**: Всё для управления аккаунтами через админку.

#### Phase 1: Database & MTProto Client
1. [x] Create migration for `telegram_accounts`
2. [x] Create migration for `telegram_publish_account_chats`
3. [x] Create migration for `telegram_publish_account_instant_chats`
4. [x] Create migration for `telegram_publish_sent_account_messages`
5. [x] Run `make sqlc` to generate DB methods
6. [x] Create `internal/tgtd/client.go` - MTProto client wrapper
7. [x] Create `internal/tgtd/auth.go` - auth manager
8. [x] Create `internal/tgtd/session.go` - session encryption/storage

#### Phase 2: Admin API - Account Management
9. [x] Add GraphQL schema types and mutations
10. [x] Run `make gqlgen`
11. [x] Implement `startTelegramAccountAuth` mutation
12. [x] Implement `completeTelegramAccountAuth` mutation
13. [x] Implement `cancelTelegramAccountAuth` mutation
14. [x] Implement `telegramAccounts` query
15. [x] Implement `telegramAccount.dialogs` field (replaced `telegramAccountChats` query)
16. [x] Implement `updateTelegramAccount` mutation
17. [x] Implement `deleteTelegramAccount` mutation

#### Phase 3: Admin API - Chat-Tag Linking
18. [x] Implement `setTelegramAccountChatPublishTags` mutation
19. [x] Implement `setTelegramAccountChatPublishInstantTags` mutation

#### Testing PR 1
20. [x] Manual test: start auth → receive code on phone
21. [x] Manual test: complete auth with code (and 2FA if enabled)
22. [x] Manual test: list account's chats
23. [x] Manual test: set publish tags for a chat

---

### PR 2: Publishing Pipeline ✅

**Scope**: Публикация заметок через аккаунты.

**Depends on**: PR 1 merged

#### Phase 4: Publishing Cases
1. [x] Create `internal/case/sendtelegramaccountpublishpost/`
2. [x] Create `internal/case/updatetelegramaccountpublishpost/`
3. [x] Create `internal/case/backjob/sendtelegramaccountmessage/`
4. [x] Create `internal/case/backjob/sendtelegramaccountpost/`
5. [x] Create `internal/case/backjob/updatetelegramaccountmessage/`
6. [x] Create `internal/case/backjob/updatetelegramaccountpost/`

#### Phase 5: Cronjob Integration
7. [x] Add SQL queries: `ListSheduledTelegarmAccountPublishNoteIDs`
8. [x] Add `EnqueueSendTelegramAccountPost` to job queue
9. [x] Add `EnqueueUpdateTelegramAccountPost` to job queue
10. [x] Modify `sendscheduledtelegrampublishposts` cronjob:
    - After enqueueing bot posts, also enqueue account posts
    - Same for update posts

#### Additional Changes
11. [x] Extend `resetTelegramPublishNote` to delete account messages
12. [x] Extend `sendTelegramPublishNoteNow` to send via account
13. [x] Add `DeleteMessage` to `tgtd.Client`
14. [x] Fix HTML formatting for MTProto (use `html.String()`)

#### Testing PR 2
15. [x] Manual test: create note with `telegram_publish_tags`
16. [x] Manual test: verify note is published via account
17. [x] Manual test: update note, verify edit works
18. [x] Manual test: verify message appears in Telegram channel

## Technical Notes

### Job Queue

Используем общую очередь для bot и account сообщений (на данном этапе). Можно разделить позже если понадобится изоляция rate limits.

### Session Storage

Session data is stored in DB as text (base64) and encrypted with AES-256-GCM. Encryption key is set via `-data-encryption-key` flag (must be exactly 32 bytes). In production mode the app will panic if the default key is used.

### Display Name

При создании аккаунта `display_name` формируется автоматически из данных Telegram:
```go
parts := []string{}
if user.FirstName != "" {
    parts = append(parts, user.FirstName)
}
if user.LastName != "" {
    parts = append(parts, user.LastName)
}
if user.Username != "" {
    parts = append(parts, "@"+user.Username)
}
displayName := strings.Join(parts, " ")
```

### Rate Limits

MTProto flood wait обрабатывается аналогично Bot API - sleep и retry. См. `internal/telegram/ratelimit.go` для паттерна.

## Reference

- gotd/td documentation: https://github.com/gotd/td
- Existing usage: `cmd/channelexport/main.go`
- Bot publish flow: `internal/case/sendtelegrampublishpost/`

---

## PR 2: Implementation Notes

### What Was Implemented

#### Phase 4: Publishing Cases ✅

1. **`internal/case/sendtelegramaccountpublishpost/`** - отправка поста через аккаунт
   - Получает чаты по тегам заметки из `telegram_publish_account_chats`
   - Для каждого чата ставит в очередь `sendtelegramaccountpost` job

2. **`internal/case/updatetelegramaccountpublishpost/`** - обновление существующих постов
   - Находит ранее отправленные сообщения в `telegram_publish_sent_account_messages`
   - Для каждого ставит в очередь `updatetelegramaccountpost` job

3. **`internal/case/backjob/sendtelegramaccountmessage/`** - низкоуровневая отправка
   - Использует `tgtd.Client.SendMessage()` для отправки через MTProto
   - Сохраняет результат в `telegram_publish_sent_account_messages`

4. **`internal/case/backjob/sendtelegramaccountpost/`** - job wrapper для отправки
   - Рендерит markdown в HTML
   - Вызывает `sendtelegramaccountmessage`

5. **`internal/case/backjob/updatetelegramaccountmessage/`** - низкоуровневое редактирование
   - Использует `tgtd.Client.EditMessage()` для редактирования через MTProto

6. **`internal/case/backjob/updatetelegramaccountpost/`** - job wrapper для обновления
   - Рендерит markdown в HTML
   - Вызывает `updatetelegramaccountmessage`

#### Phase 5: Cronjob Integration ✅

7. **Отдельные SQL запросы для bot и account пайплайнов:**
   - `ListSheduledTelegarmPublishNoteIDs` - только заметки с bot-чатами
   - `ListSheduledTelegarmAccountPublishNoteIDs` - только заметки с account-чатами

8. **Рефакторинг cronjob `sendscheduledtelegrampublishposts`:**
   ```go
   func Resolve(ctx context.Context, env Env) (any, error) {
       res := Result{}

       botPosts, err := enqueueBotJobs(ctx, env)
       // ...

       accountPosts, err := enqueueAccountJobs(ctx, env)
       // ...

       return res, nil
   }
   ```

#### Additional Changes ✅

9. **`resetTelegramPublishNote` мутация** - расширена для удаления account-сообщений:
   - Добавлен `tgtd.Client.DeleteMessage()` для удаления через MTProto
   - Удаляет записи из `telegram_publish_sent_account_messages`
   - Удаляет сообщения из Telegram через account API

10. **`sendTelegramPublishNoteNow` мутация** - расширена для отправки через account:
    - Вызывает `SendTelegramPublishPost()` для bot
    - Вызывает `SendTelegramAccountPublishPost()` для account

11. **`handletgpublishviews`** - instant preview при изменении заметки:
    - Вызывает `EnqueueSendTelegramPost()` для bot
    - Вызывает `EnqueueSendTelegramAccountPost()` для account

### Nuances & Lessons Learned

#### 1. HTML Formatting in MTProto

**Проблема:** Посты отправлялись как plain text, HTML-теги отображались буквально.

**Причина:** Bot API использует `parse_mode: "HTML"`, а MTProto работает иначе - нужно парсить HTML и конвертировать в Telegram entities.

**Решение:** Использовать `gotd/td/telegram/message/html` пакет:
```go
import (
    "github.com/gotd/td/telegram/message"
    "github.com/gotd/td/telegram/message/html"
)

sender := message.NewSender(api)
updates, err := sender.To(peer).StyledText(ctx, html.String(nil, params.Message))
```

Это автоматически парсит HTML и создаёт правильные entities для форматирования.

#### 2. Separate SQL Queries for Bot and Account

**Проблема:** Исходный `ListSheduledTelegarmPublishNoteIDs` выбирал только заметки с bot-чатами.

**Решение:** Создать отдельный `ListSheduledTelegarmAccountPublishNoteIDs`:
```sql
-- name: ListSheduledTelegarmAccountPublishNoteIDs :many
select distinct n.note_path_id
  from telegram_publish_notes n
  join note_paths p on n.note_path_id = p.id
  join telegram_publish_note_tags nt on n.note_path_id = nt.note_path_id
  join telegram_publish_account_chats ac on nt.tag_id = ac.tag_id
  join telegram_accounts a on ac.account_id = a.id
  where p.hidden_by is null
   and publish_at <= datetime('now')
   and published_at is null
   and last_error is null
   and a.enabled = 1;
```

#### 3. DeleteMessage via MTProto

**Нюанс:** Для удаления сообщений через MTProto нужно использовать разные методы в зависимости от типа чата:
- Для каналов: `api.ChannelsDeleteMessages()`
- Для остальных: `api.MessagesDeleteMessages()`

```go
switch p := peer.(type) {
case *tg.InputPeerChannel:
    _, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
        Channel: &tg.InputChannel{
            ChannelID:  p.ChannelID,
            AccessHash: p.AccessHash,
        },
        ID: []int{int(params.MessageID)},
    })
default:
    _, err := api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
        ID: []int{int(params.MessageID)},
    })
}
```

#### 4. Editing Messages by Post Type

**Разные типы постов редактируются по-разному:**

| Post Type | What Can Be Changed | Method |
|-----------|---------------------|--------|
| `text` | Text content | `EditMessage` |
| `photo` | Caption AND photo | `EditMessageWithPhoto` |
| `media_group` | Caption only (NOT media) | `EditMessageCaption` |

**Важно:** Тип поста нельзя менять после публикации. Если text-пост обновился и теперь имеет фото, изменения медиа игнорируются.

```go
switch currentPostType {
case db.TelegramPublishSentMessagePostTypeText:
    // Edit text
    client.EditMessage(...)
case db.TelegramPublishSentMessagePostTypePhoto:
    // Can replace photo
    client.EditMessageWithPhoto(...)
case db.TelegramPublishSentMessagePostTypeMediaGroup:
    // Can only edit caption of first message
    client.EditMessageCaption(...)
}
```

#### 5. No Need for Account Caching

**Первоначальный подход:** Кэшировать account при удалении нескольких сообщений.

**Реальность:** SQLite отлично справляется с N+1 запросами. Кэширование добавляет сложность без заметного выигрыша.

#### 6. bool to int64 Conversion for `instant` Field

**Проблема:** `params.Instant` имеет тип `bool`, а в базе поле `instant` имеет тип `integer`.

**Решение:** Явное преобразование перед вставкой:
```go
var instantInt int64
if params.Instant {
    instantInt = 1
}

insertParams := db.InsertTelegramPublishSentAccountMessageParams{
    // ...
    Instant: instantInt,
}
```

#### 7. Test Mocks for Extended Interfaces

**Проблема:** При расширении `Env` интерфейсов (добавление account-методов) тесты падали с `method is nil`.

**Решение:** Создать helper функцию для добавления дефолтных моков:
```go
addAccountMocks := func(env *EnvMock) *EnvMock {
    env.ListTelegramPublishSentAccountMessagesByNotePathIDFunc = func(...) ([]..., error) {
        return nil, nil
    }
    env.DeleteTelegramPublishSentAccountMessagesByNotePathIDFunc = func(...) error {
        return nil
    }
    // ...
    return env
}

// Usage:
return addAccountMocks(&EnvMock{
    // existing mocks...
})
```

### Files Created/Modified

#### New Files
- `internal/case/sendtelegramaccountpublishpost/resolve.go`
- `internal/case/updatetelegramaccountpublishpost/resolve.go`
- `internal/case/backjob/sendtelegramaccountmessage/resolve.go`
- `internal/case/backjob/sendtelegramaccountpost/resolve.go`
- `internal/case/backjob/updatetelegramaccountmessage/resolve.go`
- `internal/case/backjob/updatetelegramaccountpost/resolve.go`
- `internal/case/backjob/updateallaccounttelegrampublishposts/resolve.go` - обновление всех постов для одного аккаунта
- `internal/case/backjob/updateallaccounttelegrampublishposts/job.go`

#### Modified Files
- `internal/tgtd/client.go` - добавлены `SendMessage`, `EditMessage`, `EditMessageCaption`, `DeleteMessage` с HTML support
- `internal/case/cronjob/sendscheduledtelegrampublishposts/resolve.go` - разделение на `enqueueBotJobs()` и `enqueueAccountJobs()`
- `internal/case/cronjob/updatetelegrampublishposts/resolve.go` - добавлена поддержка обновления account-постов (параллельно с bot-постами)
- `internal/case/admin/resettelegrampublishnote/resolve.go` - удаление account-сообщений
- `internal/case/admin/sendtelegrampublishnotenow/resolve.go` - отправка через account
- `internal/case/handletgpublishviews/resolve.go` - instant preview через account
- `cmd/server/telegram.go` - добавлен `DeleteTelegramAccountMessage()`
- `cmd/server/case_methods.go` - добавлены методы для account publishing
- `cmd/server/jobs.go` - регистрация новых job handlers
- `cmd/server/main.go` - регистрация `UpdateAllAccountTelegramPublishPostsJob`
- `queries.read.sql` - добавлены `ListSheduledTelegarmAccountPublishNoteIDs`, `ListDistinctAccountIDsFromSentAccountMessages`, `ListTelegramPublishSentAccountMessagesByAccountID`
