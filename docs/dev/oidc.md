# OIDC login for trip2g — implementation plan

**Status:** plan only, no code yet.
**Goal:** add a generic **OIDC** login provider to trip2g, alongside the existing Google/GitHub OAuth, so the app can be a Relying Party against a corporate IdP. Immediate target: a client who already runs **Authentik**.

This is the trip2g half of the cross-product SSO design. The full picture (panel + trip2g + agent dashboards behind Traefik, IdP choice, authN-vs-authZ split) lives in the simplepanel repo: `docs/sso_box_design.md`. Read it for the "why"; this doc is the trip2g "how".

> trip2g conventions (from `CLAUDE.md`): English everywhere; **SQL migrations require confirmation before creating**; `make sqlc` after editing queries; commits are short one-liners, no `Co-Authored-By`.

---

## 1. How trip2g login works today (the pattern to mirror)

trip2g already has a clean, repeatable OAuth shape. OIDC is a third instance of it. Two reference providers: Google (`internal/case/handlegooglestart` + `handlegooglecallback`) and GitHub (same pair).

**Credentials live in the DB, admin-configured, secret encrypted** — NOT in env (this differs from simplepanel, which uses env flags):
- Table `google_oauth_credentials` (`db/schema.sql:540`): `id, name, client_id, client_secret_encrypted blob, active, created_at, created_by`. Model `db.GoogleOauthCredential` (`internal/db/models.go:322`).
- Loaded per request via `GetActiveGoogleOAuthCredentials` (`internal/db/queries.read.sql.go:1527`).
- Secret decrypted via the encryption manager `DecryptData` (`internal/dataencryption/encryption.go:64`).
- Written via `InsertGoogleOAuthCredentials` (`internal/db/queries.write.sql.go:1414`) + an activate query (`set active = (id = ?)`).

**State / CSRF** is a reusable package — no changes needed:
- `oauthstate.Generate(ctx, redirect, insecure)` sets a 5-min `oauth_state` nonce cookie and returns the encoded state; `oauthstate.Validate(ctx, stateParam, insecure)` checks the nonce and returns the safe redirect (`internal/oauthstate/state.go`). It also sanitizes the post-login redirect (`safeRedirect`).

**Provider client** is a small hand-rolled fasthttp client (no `golang.org/x/oauth2`):
- `internal/googleauth/`: `Config{ClientID, ClientSecret}` + `BuildAuthURL(clientID, redirectURI, state)` + `ExchangeCode(...)` + `GetUserInfo(accessToken)`. Endpoint URLs are hardcoded constants (`internal/googleauth/client.go:12-16`). Scope hardcoded `"email profile"`.

**The start handler** (`handlegooglestart/endpoint.go`): loads active creds, generates state, builds `callbackURL = env.PublicURL() + "/_system/auth/google/callback"`, redirects to the provider authorize URL. `Path()` = `/_system/auth/google`.

**The callback handler** (`handlegooglecallback/endpoint.go`): loads creds → `DecryptData` the secret → check `error` param → `oauthstate.Validate` → `ExchangeCode` → `GetUserInfo` → `env.UserByEmail(email)` → `env.SetupUserToken(userID)` (sets the session JWT cookie) → redirect. On unknown email it redirects with `?berror=user_not_found` — **it does NOT auto-create users** (`db.IsNoFound(err)` branch). `Path()` = `/_system/auth/google/callback`.

**Session** is an HS256 JWT cookie set by `SetupUserToken` (`internal/usertoken/`).

**Routing is generated**, not hand-edited:
- Each case package exposes an `Endpoint` struct with `Handle/Path/Method` and its own `Env interface`.
- `go run ./internal/router/gencmd` scans `internal/case/{"", admin, system}` for any `<name>/endpoint.go` and regenerates `internal/router/endpoints_gen.go` (the file is `DO NOT EDIT`; `//go:generate go run ./gencmd` lives in `internal/router/router.go:12`). So a new endpoint is picked up automatically once the package exists and you re-run gencmd.
- The router's aggregate `Env` must satisfy every endpoint's `Env interface` (where `GetActiveGoogleOAuthCredentials`, `PublicURL`, `Insecure`, `DecryptData`, `UserByEmail`, `SetupUserToken` are implemented).

**Admin config** of Google/GitHub creds is a **GraphQL admin mutation** — `createGoogleOAuthCredentials` / `setActiveGoogleOAuthCredentials` / `deleteGoogleOAuthCredentials` (`internal/graph/schema.graphqls:3110-3112`), backed by case handlers under `internal/case/admin/creategoogleoauthcredentials/` etc. The mutation validates + encrypts the secret and writes the row. OIDC follows the same path (detailed in Step 7).

---

## 2. What OIDC adds beyond the Google pattern

1. **Discovery instead of hardcoded endpoints.** Google hardcodes authorize/token/userinfo URLs. OIDC derives them from a single **issuer** via `GET {issuer}/.well-known/openid-configuration` → `authorization_endpoint`, `token_endpoint`, `userinfo_endpoint`, `jwks_uri`. So the credential needs one extra field: `issuer`.
2. **`openid` scope + groups.** Use scope `openid email profile` (+ a `groups` scope if the IdP maps it) to get the user's email and group memberships.
3. **`id_token`.** The token response carries an `id_token` (signed JWT). Proper OIDC verifies its signature against `jwks_uri` and reads claims from it. Minimal path: ignore the id_token and call `userinfo` exactly like Google does (simpler, matches existing code). Recommended path: verify the id_token signature (defense in depth) and read `email`/`groups` from it. See §7 open questions.
4. **Issuer string must match exactly** (Authentik per-provider issuer ends with a trailing slash — see §5).

---

## 3. Implementation steps

### Step 1 — DB migration: `oidc_credentials` table

Mirror `google_oauth_credentials`, add `issuer`. dbmate format. **Show this SQL and get confirmation before `dbmate new` (project rule).**

```sql
-- migrate:up
create table oidc_credentials (
    id integer primary key,
    name text not null,
    issuer text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    scopes text not null default 'openid email profile',
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);

-- migrate:down
drop table oidc_credentials;
```

### Step 2 — sqlc queries (`queries.read.sql` / `queries.write.sql` → `make sqlc`)

Mirror the Google credential queries:
- `GetActiveOIDCCredentials :one` — `select ... from oidc_credentials where active = 1 limit 1`.
- `InsertOIDCCredentials :one` — insert (`name, issuer, client_id, client_secret_encrypted, scopes, active, created_by`).
- `ActivateOIDCCredentials :exec` — `update oidc_credentials set active = (id = ?)` (single-active invariant, like Google).
- `DeleteOIDCCredentials :exec`.

Run `make sqlc` to regenerate `internal/db/{models,queries.read.sql,queries.write.sql}.go`.

### Step 3 — new package `internal/oidcauth`

Mirror `internal/googleauth` (fasthttp, no x/oauth2), but discovery-driven:
- `Config{ Issuer, ClientID, ClientSecret, Scopes }`, `DefaultConfig()`, `IsConfigured()`.
- `Discover(issuer) (Endpoints, error)` — fetch `{issuer}/.well-known/openid-configuration`, return `{AuthorizationEndpoint, TokenEndpoint, UserInfoEndpoint, JWKSURI}`. Cache per-issuer with a TTL (avoid a network call on every login).
- `BuildAuthURL(clientID, redirectURI, state, scopes, authzEndpoint)` — `response_type=code`, `scope=openid email profile [groups]`.
- `ExchangeCode(clientID, clientSecret, code, redirectURI, tokenEndpoint) (*TokenResponse, error)` — returns `access_token` + `id_token`.
- `GetUserInfo(accessToken, userInfoEndpoint) (*UserInfo, error)` — `UserInfo{ Sub, Email, EmailVerified, Name, Groups []string }`.
- *(recommended)* `VerifyIDToken(idToken, jwksURI, clientID, issuer) (*Claims, error)` — verify signature + `iss`/`aud`/`exp`. If you skip this initially, leave a TODO; do not silently trust an unverified id_token.

### Step 4 — case handlers

Create two packages, copied from the Google ones with provider renamed and creds loaded from `oidc_credentials`:

`internal/case/handleoidcstart/endpoint.go`
- `Env interface { GetActiveOIDCCredentials(ctx) (db.OidcCredential, error); PublicURL() string; Insecure() bool }`
- `Handle`: load creds → if unconfigured redirect `/?berror=oauth_not_configured` → read `redirect` query → `oauthstate.Generate` → `Discover(creds.Issuer)` → `callbackURL = env.PublicURL() + "/_system/auth/oidc/callback"` → `BuildAuthURL(...)` → redirect.
- `Path() = "/_system/auth/oidc"`, `Method() = GET`.

`internal/case/handleoidccallback/endpoint.go`
- `Env interface { GetActiveOIDCCredentials(ctx) (db.OidcCredential, error); DecryptData([]byte) ([]byte, error); PublicURL() string; Insecure() bool; Logger() logger.Logger; UserByEmail(ctx, email) (db.User, error); SetupUserToken(ctx, userID) (string, error) }` (+ a create-user method if auto-provisioning, see Step 8).
- `Handle`: load creds → `DecryptData(creds.ClientSecretEncrypted)` → check `error` param → `oauthstate.Validate` → `Discover` → `ExchangeCode` → (verify id_token, recommended) → `GetUserInfo` → `UserByEmail` → `SetupUserToken` → redirect. Same `?berror=...` redirect convention as Google. Log with the same fields (`provider="oidc"`, ip, email, user_id).
- `Path() = "/_system/auth/oidc/callback"`, `Method() = GET`.

### Step 5 — Env method

Add `GetActiveOIDCCredentials(ctx) (db.OidcCredential, error)` to the router's aggregate Env (the same type that already implements `GetActiveGoogleOAuthCredentials`). It is a thin call into the read queries from Step 2.

### Step 6 — regenerate routes

```
go run ./internal/router/gencmd   # auto-discovers handleoidcstart / handleoidccallback
```
Commit the regenerated `internal/router/endpoints_gen.go` together with the new packages (project rule: generated files committed with their source).

### Step 7 — admin config (the "OAuth provider request") + login button

In trip2g an OAuth provider is **not** configured via env — an admin **requests** it through a **GraphQL admin mutation**, which encrypts the secret and writes the DB row. Mirror the existing Google flow exactly.

**Reference (existing Google provider request):**
- GraphQL mutations (`internal/graph/schema.graphqls:3110-3112`): `createGoogleOAuthCredentials`, `setActiveGoogleOAuthCredentials`, `deleteGoogleOAuthCredentials`; queries `allGoogleOAuthCredentials` / `googleOAuthCredentials(id)` (`:1176-1177`). Input is `CreateGoogleOAuthCredentialsInput { name, clientId, clientSecret }` (`:2666`); type `AdminGoogleOAuthCredentials @goModel(db.GoogleOauthCredential)` (`:597`).
- Resolver-backing case handler: `internal/case/admin/creategoogleoauthcredentials/resolve.go` (+ `setactivegoogleoauthcredentials/`, `deletegoogleoauthcredentials/`). Its `Env` needs: `CurrentAdminUserToken` (admin gate), `ValidateGoogleOAuthCredentials` (probe the creds), `EncryptData` (encrypt secret), `DeactivateAllGoogleOAuthCredentials` (single-active invariant), `InsertGoogleOAuthCredentials`.

**OIDC additions (mirror the above):**
1. Schema (`internal/graph/schema.graphqls`): add
   ```graphql
   input CreateOIDCCredentialsInput { name: String!, issuer: String!, clientId: String!, clientSecret: String!, scopes: String }
   type AdminOIDCCredentials @goModel(model: "trip2g/internal/db.OidcCredential") { id: Int!, name: String!, issuer: String!, clientId: String!, active: Boolean!, createdAt: Time!, createdBy: User! }
   # + CreateOIDCCredentialsPayload / ...OrErrorPayload, and Delete/SetActive inputs+payloads
   ```
   plus mutations `createOIDCCredentials` / `setActiveOIDCCredentials` / `deleteOIDCCredentials` and queries `allOIDCCredentials` / `oidcCredentials(id)`.
2. Case handlers: `internal/case/admin/createoidccredentials/resolve.go` (+ setactive/delete), copied from the Google ones; `Env` mirrors Google's with `ValidateOIDCCredentials` doing a discovery probe (`{issuer}/.well-known/openid-configuration` → 200) instead of a Google token check.
3. Regenerate: `make gqlgen` (gqlgen) and `moq` for the `*_test.go` mocks; commit generated files with their source.

**The actual request an admin sends** (the "запрос на oauth provider"):
```graphql
mutation {
  createOIDCCredentials(input: {
    name: "Authentik",
    issuer: "https://authentik.company/application/o/trip2g/",
    clientId: "...", clientSecret: "...",
    scopes: "openid email profile"
  }) { ... on CreateOIDCCredentialsPayload { credentials { id name active } } ... on ErrorPayload { message } }
}
```
Then `setActiveOIDCCredentials(input: {id})` to make it the active provider. This is the same admin surface that manages Google/GitHub creds today (whatever admin page calls those mutations gets an OIDC section).

- UI: add a "Sign in with SSO" button pointing at `/_system/auth/oidc` (next to the existing Google/GitHub buttons).

### Step 8 — account policy (decision required)

Today an unknown email → `?berror=user_not_found` (no auto-create). For corporate SSO you usually want **auto-provisioning** on first login. Options:
- **A (mirror current):** keep `user_not_found`; users must be pre-created. Safest, least surprising.
- **B (auto-provision):** create a user from the verified OIDC email on first login, optionally gated by a per-credential flag (e.g. `oidc_credentials.auto_provision`) and/or an allowed email-domain / required group. Recommended for the box, but it changes who can get an account — confirm with the owner before enabling.

---

## 4. URL configuration — issuer vs redirect_uri

Two URLs in opposite directions; do not confuse them.

| What | Where it is set | Value | Who provides it |
|---|---|---|---|
| **issuer** (we call them) | `oidc_credentials.issuer` (admin/GraphQL) | `https://authentik.company/application/o/<app-slug>/` | client's Authentik admin (the Application slug) |
| **client_id / client_secret** | `oidc_credentials` (admin, secret encrypted) | from the Authentik provider | client's Authentik admin |
| **redirect_uri** (they call us back) | **registered in Authentik**, computed by us | `{PublicURL}/_system/auth/oidc/callback` | we tell the Authentik admin |
| `PublicURL` | trip2g config (existing `env.PublicURL()`) | e.g. `https://trip2g.client.com` | deployment |

- You only store the **issuer**; discovery derives authorize/token/userinfo/jwks.
- ⚠️ **Trailing slash matters** — Authentik per-provider `iss` is exactly `…/o/<app-slug>/`. If the stored issuer omits the slash, id_token `iss` verification fails.
- The **redirect_uri** must be added to the Authentik Application's allowed redirect URIs (exact match), or Authentik returns `redirect_uri mismatch`.

---

## 5. Authentik side (done once by the client's admin)

Mirrors `docs/sso_box_design.md` §4.1 in the simplepanel repo:
1. Applications → Applications → **New Provider** → **OAuth2/OpenID Provider**; create the linked **Application** (its slug becomes the issuer path).
2. Set the **Redirect URI** to `{PublicURL}/_system/auth/oidc/callback`.
3. Copy `client_id` / `client_secret` into trip2g's `oidc_credentials` (issuer = `https://authentik.company/application/o/<slug>/`).
4. To get **groups** in the token: Customization → Property Mappings → new **OAuth2 Scope Mapping** returning `[g.name for g in request.user.ak_groups.all()]`, attach it to the provider; add its scope to `oidc_credentials.scopes`.

---

## 6. Groups → authorization (separate concern, defer)

Authentication (who the user is) is what this plan delivers. **Authorization** (what subgraphs/instances the user may see) is trip2g's existing access model (subgraph access, e.g. `UserSubgraphAccess`). Mapping OIDC `groups` → subgraph access is a follow-up: read `groups` from the verified token in the callback and grant/deny accordingly. Not required for basic SSO login; design it after Step 8.

---

## 7. Open questions / risks

1. **id_token verification vs userinfo-only.** Minimal mirror of Google uses `userinfo` only. Proper OIDC verifies the id_token signature against `jwks_uri`. Recommend verifying; if deferred, mark a clear TODO and never trust unverified id_token claims for authZ.
2. **Discovery caching.** Cache `.well-known` per issuer with a TTL; do not fetch on every login. Handle issuer/IdP downtime gracefully (fall back to a clear `?berror=`).
3. **Auto-provisioning (Step 8).** Policy + security decision (who gets an account). Confirm before enabling B.
4. **Authentik reachability.** trip2g must reach the issuer's token/userinfo/jwks over HTTPS at login time, and the user's browser must reach authorize. If the client's Authentik is network-isolated, plan connectivity (see `sso_box_design.md` §6 risk 8).
5. **No `golang.org/x/oauth2` in trip2g.** Keep the hand-rolled fasthttp style for consistency; that means implementing discovery + (optional) JWKS verification by hand, or pulling a minimal JOSE dependency for verification only.
6. **Single active credential.** Reuse the Google single-active invariant (`set active = (id = ?)`); decide whether multiple IdPs can be active at once (Google/GitHub/OIDC are independent rows in separate tables, so OIDC active is orthogonal).

---

## 8. Effort & file checklist

**Size: M** (no provider abstraction to reuse; a new package + two endpoints + migration + sqlc + GraphQL + gencmd).

- [ ] `db/migrations/<ts>_create_oidc_credentials.sql` (confirm SQL first)
- [ ] `queries.read.sql` / `queries.write.sql` + `make sqlc`
- [ ] `internal/oidcauth/{config.go, client.go, discovery.go, models.go}`
- [ ] `internal/case/handleoidcstart/endpoint.go`
- [ ] `internal/case/handleoidccallback/endpoint.go`
- [ ] Env: `GetActiveOIDCCredentials` on the router Env
- [ ] `go run ./internal/router/gencmd` → commit `endpoints_gen.go`
- [ ] GraphQL admin mutation/query for OIDC creds + UI login button
- [ ] account policy decision (Step 8)
- [ ] `go test ./...`, manual login flow against a test Authentik, acceptance smoke (`{issuer}/.well-known/openid-configuration` → 200, full code→callback→session)
