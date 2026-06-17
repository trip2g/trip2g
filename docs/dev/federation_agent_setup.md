# Federation Agent Setup — техническое сырьё

**Назначение:** сырой технический референс для агента, который **сам** настраивает обмен знаниями между инстансами trip2g (приватность по умолчанию, общий пул `shared`, хабы, oversight). Из этого документа отдельным пайплайном собирается user-facing инструкция для агентов; здесь — все факты, точные сигнатуры и edge-cases без «причёсывания».

**Статус:** v0, 2026-06-17. Сверено с кодом trip2g:
- доступ к нотам: `internal/case/canreadnote/resolve_with_subgraphs.go`
- федерация: `internal/case/mcp/federation_handlers.go`, `internal/case/mcp/federation_helpers.go`, `internal/federation/`
- frontmatter-патчи: `internal/frontmatterpatch/evaluate.go`
- MCP admin-тулы: `internal/case/mcp/resolve.go`

**Пользовательские доки (trip2g.com/docs), на которые ссылаемся:**
- [federation](https://trip2g.com/docs/en/user/federation) — модель федерации end-to-end
- [advanced](https://trip2g.com/docs/en/user/advanced) — сабграфы и контроль доступа
- [frontmatter-patches](https://trip2g.com/docs/en/user/frontmatter_patches) — патчи фронтматтера
- [hub create](https://trip2g.com/docs/en/hub/_create) — как добавить базу в хаб (KB-нота)
- [mcp](https://trip2g.com/docs/en/user/mcp) — локальный MCP-сервер и тулы
- [selfhosted](https://trip2g.com/docs/en/user/selfhosted) — env-переменные

---

## 1. Контрольная плоскость: чем агент шлёт admin-операции

Все операции ниже — это **admin GraphQL** на конкретном инстансе. Есть два пути его вызвать; остальной документ от пути не зависит.

### 1.1 Через simplepanel (рекомендуется в пуле)

Панель — доверенный control-plane: знает master-secret, аутентифицируется админом в любой инстанс пула по cookie (HAT→session-cookie). Агент работает по `instance_id`, не зная секретов инстансов.

MCP-тулы панели (авторизация — API-ключ панели):

| Тул | Аргументы | Что делает |
|---|---|---|
| `list_instances` | — | список инстансов пула: `{instance_id, name, domain}` |
| `instance_federation` | `instance_id` | топология инстанса (см. §7): self, сабграфы, секреты, KB-ноты |
| `instance_graphql_request` | `instance_id`, `query`, `variables?` | выполнить admin GraphQL на инстансе; панель проксирует под cookie-админом |

Механика проксирования: панель подписывает HAT-JWT (`{e: adminEmail, ae:true}`) ключом инстанса, POST `/_system/hat` → session-cookie → этой cookie шлёт на `/_system/graphql`. Авторизация на инстансе — обычный `checkAdmin` по session-cookie (admin-bypass). Реализация переиспользует клиент `internal/federation/client.go` (`GetTopology`, `GraphQL`).

### 1.2 Напрямую через trip2g MCP (`graphql_request`)

Если кто-то работает с инстансом отдельно (вне пула панели), trip2g сам экспонирует admin-тулы по MCP. Требуется API-ключ инстанса с флагом `enable_mcp_admin_tools=true` (мутация `setApiKeyMcpAdminTools`). Тогда на `/_system/mcp` доступны (см. `internal/case/mcp/resolve.go`):

- `graphql_introspection(pattern)` — найти операции/типы по regexp перед вызовом;
- `graphql_request(query, variables?)` — выполнить query/mutation как admin.

Дальше в документе «вызвать admin GraphQL X» = либо `instance_graphql_request(instance_id, X)` (панель), либо `graphql_request(X)` (нативно).

> **Deployment-зависимо.** Путь 1.1 предполагает развёрнутую simplepanel с доступом к master-secret пула. Путь 1.2 предполагает, что на инстансе включены MCP admin-тулы у ключа. Если ни того, ни другого — настройка делается руками в админке.

---

## 2. Модель приватности

Единица — **агент со своей базой знаний** (инстанс trip2g). Уровни:

- **`private`** — дефолт. Нота без сабграфа должна попадать в `private` (через патч §5.1) и быть видна только самому агенту/админу.
- **`shared`** — общий пул для коллег. Нота с `subgraph: shared` осознанно расшарена. Это же — дефолтный scope федеративных связей.
- **oversight** — избранные ридеры (руководство/СБ) читают **всё, включая `private`**, через связи со scope, включающим `private`.

Топология по умолчанию (низкая церемония):
- агенты связаны через **хаб(ы)** по `shared` (а не N² прямой меш);
- один-несколько **oversight**-ридеров со scope = все сабграфы.

Хабов может быть несколько (например, по отделам); хабы могут федерироваться между собой (с учётом лимита глубины, §8).

---

## 3. Семантика доступа — ОБЯЗАТЕЛЬНО прочитать

Когда федеративный пир (с `kid`) читает ноту, trip2g проверяет (`canreadnote/resolve_with_subgraphs.go`, `allowed` = сабграфы scope этого `kid`):

```
1. нота free:true            → читается ВСЕГДА (публично)
2. len(allowed) == 0         → НИЧЕГО (кроме free). Пустой scope = НЕТ доступа
3. нота без сабграфов        → читается ЛЮБЫМ пиром с непустым scope   ← ВАЖНО
4. сабграф ноты ∈ allowed    → читается
5. иначе                     → не читается
```

**Два следствия, определяющие весь сетап:**

- **(п.3) Нота без сабграфа видна любому пиру с любым непустым scope.** Поэтому «приватно по умолчанию» держится ТОЛЬКО при установленном патче default-private (§5.1). Без него выдача кому-либо `shared` заодно открывает все нетегнутые ноты. **Патч default-private — несущая конструкция, а не опция.**
- **(п.2) Пустой scope = нет доступа, НЕ «полный доступ».** «Читать всё» задаётся перечислением сабграфов в scope (вкл. `private`), вайлдкарда нет.

`free:true` — отдельная ось: делает ноту публичной независимо от сабграфов. Для приватного внутреннего хаба `free` на знаниях не используем (только, при желании, на самих KB-нотах хаба — см. §5.5).

---

## 4. Точные GraphQL-сигнатуры (референс)

Все — admin (cookie-bypass через §1). Источник: `internal/graph/schema.graphqls`.

**Федеративные секреты:**
```graphql
# inbound: «кто-то может читать меня». kid ОБЯЗАТЕЛЕН; secretHex опционален (пустой → сервер сгенерит).
createInboundFederationSecret(input: { kid: String!, description: String, secretHex: String })
  -> { id, kid, secretHex }     # secretHex показывается ОДИН раз

# outbound: «я могу читать пира».
createOutboundFederationSecret(input: { kid: String!, secretHex: String!, kbURL: String!, description: String })
  -> { id }

# scope inbound-секрета: какие сабграфы видит пир под этим kid (allow-list).
addFederationSecretSubgraph(input: { kid: String!, subgraphID: Int64! }) -> { success }
removeFederationSecretSubgraph(input: { kid: String!, subgraphID: Int64! }) -> { success }

revokeFederationSecret(id: Int64!) -> { revokedId }   # выбери поле через __typename, если revokedId не нужен
```

**Сабграфы (чтение):**
```graphql
allSubgraphs -> AdminSubgraphsConnection { nodes { id, name, color, hidden, requireSignin } }
```
Сабграф **создаётся сам**, когда на инстанс прилетает нота с `subgraph: <name>` (отдельной мутации create нет). См. §5.2.

**Заметки (запись) — создаёт/обновляет ноты, в т.ч. с произвольным фронтматтером:**
```graphql
updateNotes(input: { changes: [ { upsert: { path: String!, content: String! } } ] }) -> { ... }
# admin по cookie проходит (checkapikey admin-bypass). X-Api-Key не нужен при cookie-админе.
```

**Frontmatter-патчи:**
```graphql
allFrontmatterPatches -> { nodes { id, includePatterns, excludePatterns, jsonnet, priority, description, enabled } }
createFrontmatterPatch(input: {
  includePatterns: [String!]!, excludePatterns: [String!], jsonnet: String!,
  priority: Int!, description: String!, enabled: Boolean!
}) -> { frontmatterPatch { id } }
updateFrontmatterPatch(input: {...}) ; deleteFrontmatterPatch(input: { id }) 
```

**Федерация (чтение своих секретов):**
```graphql
federationSecrets -> [ { id, kid, kbUrl, description, createdAt, revokedAt, subgraphCount } ]
```
(`kbUrl != null` → outbound; `kbUrl == null` → inbound.)

**Сводно (если есть эндпоинт панели):** `GET /_system/federation/admin` отдаёт self + сабграфы + inbound/outbound со scope + KB-ноты одним вызовом (тул `instance_federation`).

---

## 5. Рецепты операций

Каждый рецепт = что вызвать + проверка. Все вызовы — через §1.

### 5.1 Default-private (несущий шаг; делать ПЕРВЫМ на каждом инстансе)

Ставит патч: нота без сабграфа получает `subgraph: private`. Заметки на диске не меняются (патч виртуальный, на load).

```graphql
createFrontmatterPatch(input: {
  includePatterns: ["**/*.md"],
  excludePatterns: [],
  priority: -100,                 # применяется первым
  enabled: true,
  description: "agent:default-private",
  jsonnet: "if std.objectHas(meta, \"subgraph\") || std.objectHas(meta, \"subgraphs\") then {} else { subgraph: \"private\" }"
})
```

Механика (`frontmatterpatch/evaluate.go`): `meta` приходит в jsonnet как объект (ExtVar JSON), результат **shallow-merge поверх** текущего meta. Возврат `{}` ничего не меняет → ноты с явным сабграфом не трогаются; без сабграфа — получают `private`.

**Идемпотентность:** перед созданием проверь `allFrontmatterPatches` на `description == "agent:default-private"`. Если есть — не дублируй.

**Проверка:** `allFrontmatterPatches` содержит патч; любая нетегнутая нота теперь отдаёт `subgraph: private` (видно через рендер/доступ).

> **Caveat (deployment-зависимо).** Если инстанс отдаёт ещё и публичный сайт — у публичных нот должен быть `free: true` (он перекрывает приватность), либо добавь их пути в `excludePatterns`. Для чисто внутреннего инстанса — неактуально.

### 5.2 Создать сабграф `shared` (+ регламент шаринга в его ноте)

Сабграф материализуется записью ноты с `subgraph: shared`. Заодно тело ноты = человекочитаемый регламент «что сюда класть».

```graphql
updateNotes(input: { changes: [ { upsert: {
  path: "subgraphs/shared.md",
  content: "---\nsubgraph: shared\n---\nРегламент: сюда — знания, которые должны находить агенты коллег. Всё остальное приватно по умолчанию."
} } ] })
```

**Проверка:** `allSubgraphs` содержит `shared`.

Сабграф для oversight (`private` уже существует после первой приватной ноты; отдельная нота-регламент не обязательна). При желании — `subgraphs/private.md`.

### 5.3 Расшарить ноту

Автор/агент добавляет ноте `subgraph: shared` (через `updateNotes` upsert с обновлённым фронтматтером). Всё, что не помечено — остаётся `private` (§5.1).

### 5.4 Приватная федеративная связь A→B (scope `shared`)

Двусторонний обмен ключами (как в [federation](https://trip2g.com/docs/en/user/federation), но автоматически):

1. **На B** (цель) — inbound-секрет, сгенерь уникальный `kid` (напр. `<A-name>-<unixtime>`):
   ```graphql
   createInboundFederationSecret(input: { kid: "A2026", description: "from A" }) -> { kid, secretHex }
   ```
2. **На B** — scope секрета на `shared` (нужен `subgraphID` из `allSubgraphs`):
   ```graphql
   addFederationSecretSubgraph(input: { kid: "A2026", subgraphID: <shared.id на B> })
   ```
3. **На A** (источник) — outbound-секрет на B:
   ```graphql
   createOutboundFederationSecret(input: { kid: "A2026", secretHex: "<из шага 1>", kbURL: "https://B/_system/mcp" })
   ```
4. **На A** — KB-нота-маршрут (иначе агент A не узнает о пути; см. [hub create](https://trip2g.com/docs/en/hub/_create)):
   ```graphql
   updateNotes(input: { changes: [ { upsert: {
     path: "hub/B.md",
     content: "---\nmcp_federation_kb_url: https://B/_system/mcp\nmcp_federation_kb_id: B\nsubgraph: shared\n---\nКогда обращаться к базе B."
   } } ] })
   ```

**Видимость KB-ноты (важно).** KB-нота срабатывает (триггерит `federated_search`) только если её **видит** тот, кто ищет на A. Поэтому сабграф самой KB-ноты должен совпадать с доступом ищущего агента: на общем хабе — `subgraph: shared` (или `free: true` для публичного хаба). НЕ путать со scope секрета (он про то, что A видит у B): scope секрета и сабграф KB-ноты — независимы.

**Проверка:** на A `federationSecrets` содержит outbound на `https://B/_system/mcp`; на B — inbound с тем же `kid` и непустым scope; `federated_search(kb_id="B")` с A возвращает результаты.

### 5.5 Хаб (вместо N² прямого меша)

Хаб H = инстанс, в чьём vault лежат KB-ноты на всех членов (папка `hub/`). Член M:
- даёт H inbound-секрет scope `shared` (H читает `shared` у M) — §5.4 шаги 1–2 на M;
- H имеет outbound-секрет + KB-ноту `hub/M.md` на M — §5.4 шаги 3–4 на H.

Тогда агент любого члена, чей запрос доходит до H (член → H), фанаутится по всем `shared` членов. Симметрично: член тоже федерируется к H (KB-нота `hub/H.md` + секреты), чтобы искать через хаб.

Несколько хабов: H1↔H2 связываются той же §5.4. Помни про **лимит глубины** (§8) — цепочка хаб→хаб→член считается.

### 5.6 Oversight (читать всё, включая `private`)

Для избранного ридера O: при выдаче ему inbound-секрета на каждом инстансе добавь в scope **все** сабграфы, включая `private`:
```graphql
addFederationSecretSubgraph(input: { kid: "OVERSIGHT", subgraphID: <private.id> })
addFederationSecretSubgraph(input: { kid: "OVERSIGHT", subgraphID: <shared.id> })
# ... + остальные сабграфы инстанса
```
Помечай такой секрет в `description` как `oversight`, чтобы визуализация (граф панели) показывала его как намеренный привилегированный доступ, а не как случайную утечку.

### 5.7 Отзыв / сужение

```graphql
revokeFederationSecret(id: <id>)                         # полный отзыв
removeFederationSecretSubgraph(input: { kid, subgraphID }) # убрать один сабграф из scope
```
После отзыва маршрутная KB-нота «висит» — удали/перепиши её `updateNotes`, иначе агент будет получать 401.

### 5.8 Прочитать/проверить состояние

- `instance_federation(instance_id)` (панель) или `federationSecrets` + `allSubgraphs` + `allFrontmatterPatches` (нативно).
- Признаки здоровья: inbound со scope (непустым!), outbound с тем же `kid`, KB-нота на тот же `kbURL`, патч default-private на месте.

---

## 6. Дефолтный сетап «команда» (низкая церемония)

Последовательность, которую агент прогоняет по пулу (через `list_instances`):

1. На КАЖДОМ инстансе: §5.1 default-private (иначе модель дырявая).
2. На КАЖДОМ инстансе: §5.2 сабграф `shared` с регламентом.
3. Выбрать хаб(ы) H. Для каждого члена M: §5.5 (M↔H по `shared`).
4. (Опц.) Oversight O: §5.6 на всех инстансах (scope = все сабграфы).
5. Проверка: §5.8 по всем; на хабе `federated_search` возвращает `shared` членов.

Итог: всё приватно по умолчанию; `shared` виден команде через хаб; oversight видит всё; ключей вручную никто не трогал.

---

## 7. Контракт `instance_federation` / `GET /_system/federation/admin`

Сводный read (admin-only), один JSON:
```json
{
  "self":     { "name", "kb_id", "mcp_url", "subgraphs": [ {id,name} ] },
  "outbound": [ { "id","kid","kb_url","revoked_at","subgraphs":[{id,name}] } ],
  "inbound":  [ { "id","kid","revoked_at","subgraphs":[{id,name}] } ],
  "kb_notes": [ { "kb_id","kb_url","description","path" } ]
}
```
Используется для визуализации (граф simplepanel) и для проверки агентом (§5.8).

---

## 8. Что зависит от развёртывания

- **Глубина фан-аута** — `MCP_FEDERATION_MAX_DEPTH` (дефолт 3). Ограничивает цепочки хаб→хаб→член и предотвращает циклы. Многоуровневые хабы планируй с учётом лимита. См. [selfhosted](https://trip2g.com/docs/en/user/selfhosted).
- **Таймаут на пира** — `MCP_FEDERATION_FANOUT_TIMEOUT` (дефолт 2с). Медленные/недоступные пиры пропускаются.
- **kb_id по умолчанию** = host из public URL инстанса (можно переопределить `mcp_federation_kb_id` в KB-ноте). Public URL — конфиг инстанса.
- **Транспорт** — HMAC-SHA256 only, JWT exp ~30с, без mTLS/OAuth, реплей-кэша нет. TLS не форсится хабом — в проде только HTTPS-URL пиров.
- **Контрольная плоскость** — путь §1.1 требует развёрнутой simplepanel с master-secret пула; §1.2 требует ключа с `enable_mcp_admin_tools`. Без них — ручная админка.
- **`free:true` и публичные сайты** — см. caveat §5.1.

---

## 9. На будущее: тонкий CLI

После обкатки этого плейбука имеет смысл тонкий CLI (python/js, минимум зависимостей) — обёртка над `instance_graphql_request`/`graphql_request` с готовыми под-командами (`default-private`, `make-shared`, `link`, `hub-join`, `oversight`, `status`). Это снижает токены и шанс ошибки у агента: вместо длинных GraphQL-строк — короткие команды. Делать **после** «работает, теперь финальные правки» по основному плейбуку.

---

## 10. Карта ссылок (trip2g.com/docs)

| Тема | Док |
|---|---|
| Модель федерации | /docs/en/user/federation |
| Сабграфы и доступ | /docs/en/user/advanced |
| Frontmatter-патчи | /docs/en/user/frontmatter_patches |
| KB-нота / хаб | /docs/en/hub/_create |
| MCP-сервер и тулы | /docs/en/user/mcp |
| Env (depth/timeout) | /docs/en/user/selfhosted |
