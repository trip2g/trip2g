# `GET /_system/federation/admin` — read-эндпоинт топологии федерации

**Audience:** разработчики trip2g. Эндпоинт для внешнего admin-клиента (панель `trip2g_simplepanel`), которому нужна полная картина федерации инстанса одним вызовом.

**Status:** design, 2026-06-16.
**Парный спек (потребитель):** `trip2g_simplepanel/docs/superpowers/specs/2026-06-16-federation-graph-design.md`.

**Reference:** `internal/case/mcp/`, `internal/case/admin/listfederationsecrets/`, `internal/model/mcp_federation_note.go`, `internal/graph/helpers.go` (`checkAdmin`).

---

## 1. Зачем

Панель строит граф федерации пула: кто на кого проксит, где связь односторонняя, где маршрут без авторизации. Сейчас собрать это с инстанса нельзя за один вызов:

- `federationSecrets` (GraphQL) отдаёт секреты (inbound/outbound, `kbUrl`, `revokedAt`, `subgraphCount`), но **не** отдаёт сами scope-сабграфы и **не** отдаёт KB-ноты;
- **KB-ноты** (`mcp_federation_kb_url` во frontmatter) наружу не отдаются вообще — а это «куда инстанс реально проксит», особенно для публичных пиров (у них только KB-нота, без секрета);
- инстанс не отдаёт собственную идентичность (`kb_id` / `mcp_url`).

Эндпоинт закрывает всё это одним admin-only GET-ом.

## 2. Контракт

```
GET /_system/federation/admin
Cookie: <session-cookie>              // admin-сессия, см. ниже
Accept: application/json
```

**Как клиент получает admin-сессию.** Эндпоинт не вводит новой авторизации — обычный `checkAdmin` по `UserToken()`, который читает session-cookie (или `t2g_` personal token). Внешний клиент (панель) получает cookie тем же путём, что и браузер: POST `/_system/hat` с HAT-JWT (`{e, ae:true}`, подпись `JWT_SECRET` инстанса) → инстанс ставит `Set-Cookie` → клиент переиспользует её для этого GET и для admin-GraphQL-мутаций. Никакого Bearer-admin-JWT trip2g не принимает (`UserToken()` пропускает не-`t2g_` Bearer в anonymous).

### URL namespace

Пути под `/_system/federation/` — сиблинги, права видны прямо из URL (публичный код физически не может вернуть admin-данные):

| Путь | Права | Статус |
|---|---|---|
| `/_system/federation/admin` | admin-only (`checkAdmin`) | этот спек |
| `/_system/federation/public` | аноним/любой | **зарезервировано на будущее** (см. §7) |
| `/_system/federation/` (база) | — | свободна, позже под индекс/`capabilities` |

**Авторизация:** admin-only. Переиспользовать существующий паттерн `checkAdmin(ctx)` (`internal/graph/helpers.go:43`) — извлечь `UserToken` из `appreq.Request`, проверить `token.IsAdmin()`. Не-admin → `403`. Анонимный → `401`.

**Ответ `200 application/json`:**

```json
{
  "self": {
    "name": "alice",
    "kb_id": "alice.team.io",
    "mcp_url": "https://alice.team.io/_system/mcp",
    "subgraphs": [
      {"id": 1, "name": "internal"},
      {"id": 2, "name": "public"}
    ]
  },
  "outbound": [
    {
      "id": 12,
      "kid": "bob-2026",
      "kb_url": "https://bob.team.io/_system/mcp",
      "revoked_at": null,
      "subgraphs": [{"id": 1, "name": "internal"}]
    }
  ],
  "inbound": [
    {
      "id": 7,
      "kid": "carol-2026",
      "revoked_at": null,
      "subgraphs": [{"id": 1, "name": "internal"}]
    }
  ],
  "kb_notes": [
    {
      "kb_id": "bob",
      "kb_url": "https://bob.team.io/_system/mcp",
      "description": "Use when: Bob's work-status updates.",
      "path": "federation/bob.md"
    }
  ]
}
```

### 2.1 Поля

**`self`** — идентичность инстанса:
- `name` — человекочитаемое имя (из конфигурации/домена).
- `kb_id` — дефолтный kb_id инстанса (хост из `mcp_url`), чтобы пир мог его адресовать.
- `mcp_url` — собственный `/_system/mcp` endpoint (по базовому URL инстанса).
- `subgraphs[]` — все сабграфы инстанса (`id`, `name`). Источник: `ListAllSubgraphs` (та же выборка, что у GraphQL `allSubgraphs`). Нужно панели, чтобы построить scope-пикер без отдельного вызова.

**`outbound[]`** — секреты, которыми инстанс ходит к пирам (есть `kb_url`):
- `id`, `kid`, `kb_url`, `revoked_at`.
- `subgraphs[]` — scope (какие сабграфы этого инстанса видит пир под этим kid). **Новое** относительно `federationSecrets`, который отдаёт только `subgraphCount`. Источник: `federation_secret_subgraphs` JOIN `subgraphs`.

**`inbound[]`** — секреты, которыми пиры ходят к этому инстансу (`kb_url` отсутствует):
- `id`, `kid`, `revoked_at`, `subgraphs[]` (scope, выданный пиру).

> Разделение inbound/outbound — по наличию `kb_url` (как в текущей модели `ListFederationSecretsRow`: `KbUrl *string`). Outbound имеют `kb_url`, inbound — нет.

**`kb_notes[]`** — маршруты из vault (ноты с `mcp_federation_kb_url`):
- `kb_id` — `mcp_federation_kb_id` или дефолт (хост `kb_url`).
- `kb_url` — `mcp_federation_kb_url`.
- `description` — тело ноты (или первые N символов) — для подсказки «Use when».
- `path` — путь ноты в vault (для адресации/редактирования).
- Источник: `NoteViews.MCPFederationNotes` (`internal/model/mcp_federation_note.go`, заполняется `ExtractMCPFederationNotes` в `internal/mdloader/loader.go`).

### 2.2 Пустой scope = полный доступ

`subgraphs: []` у секрета означает отсутствие scope-ограничения — пир видит всё доступное (broad grant). Панель помечает такие связи отдельно. Эндпоинт просто отдаёт пустой массив, без спец-флага.

### 2.3 Видимость KB-нот

`kb_notes[]` отдаём **все** ноты с `mcp_federation_kb_url` без фильтра по subgraph-доступу — это admin-only эндпоинт, админ видит весь vault. (В отличие от `accessibleKBNotes()` в `internal/case/mcp/federation_acl.go`, который фильтрует по доступу анонимного/подписчика для рантайма федерации.)

## 3. Реализация

- Новый case `internal/case/admin/federationtopology/` (или handler `internal/case/system/federation/`), по существующему паттерну admin-кейсов.
- Роут: зарегистрировать `GET /_system/federation/admin` рядом с `/_system/hat`, `/_system/mcp` (там же, где регистрируются `/_system/*`). Группа `/_system/federation/` — под будущие сиблинги.
- Сборка ответа:
  1. `self.subgraphs` ← `ListAllSubgraphs`.
  2. `outbound`/`inbound` ← расширить выборку секретов, чтобы тянуть не `subgraphCount`, а сами строки scope (`federation_secret_subgraphs` JOIN `subgraphs` → `[]{id,name}` на секрет). Можно отдельным запросом «scope по всем kid» + сшивка в памяти, чтобы не плодить N+1.
  3. `kb_notes` ← `NoteViews.MCPFederationNotes` из загруженного vault.
  4. `self.name`/`kb_id`/`mcp_url` ← из конфигурации инстанса (базовый URL) + дефолт kb_id = host.
- Сериализация: easyjson по конвенции пакета (как `internal/federation/types.go`).

## 4. Безопасность

- **admin-only**, `checkAdmin`. Без admin → 401/403.
- **Секреты в hex НЕ отдаём.** Только `kid`, `kb_url`, `revoked_at`, scope. `secretHex` показывается один раз при создании (существующий `createInbound` flow) и сюда не попадает.
- Эндпоинт read-only; все мутации остаются в GraphQL (`createInbound/OutboundFederationSecret`, `addFederationSecretSubgraph`, `removeFederationSecretSubgraph`, `revokeFederationSecret`, `updateNotes`).

## 5. Тестирование

- Unit на сборку ответа: фикстуры секретов (inbound/outbound, revoked, с/без scope) + KB-нот + сабграфов → ожидаемый JSON.
- Auth: не-admin → 403, аноним → 401, admin → 200.
- N+1: scope подтягивается батчем, проверить число запросов.
- Совместимость: поля только добавляются; existing `federationSecrets` GraphQL не трогаем.
- `gofmt -w .` + `go test ./...`.

## 6. Вне scope

- Запись через этот эндпоинт (только GET).
- Health/latency пиров (панель меряет сама при краулинге).
- Пагинация (объёмы федерации малы; при необходимости добавить позже).

## 7. На будущее: `/_system/federation/public` (зарезервировано)

Отдельный публичный сиблинг, **не реализуется в этом спеке** — путь резервируется, чтобы не ломать namespace позже.

- **Права:** аноним/любой (без `checkAdmin`).
- **Отдаёт только то, что хаб и так раскрывает в рантайме федерации:** публичную маршрутизацию — `free: true` KB-ноты / публичные пиры (как их видит анонимный MCP-вызов через `accessibleKBNotes()`).
- **Никогда не отдаёт:** сабграфы, scope секретов, `kid`, inbound-секреты, hidden-сабграфы, непубличные KB-ноты.
- Назначение — публичный discovery сети (другой хаб/директория узнаёт, куда этот хаб федерится публично), без раскрытия приватной топологии.

Раздельные пути (а не один эндпоинт с переключением по роли) гарантируют, что публичный код физически не имеет доступа к admin-выборкам.
