# Google & GitHub OAuth Authentication

## Implementation Status

### Database — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| Migration | `db/migrations/20260119020631_create_oauth_credentials_tables.sql` | ✅ |
| SQL queries | `queries.read.sql`, `queries.write.sql` | ✅ |
| Generated code | `make sqlc` | ✅ |

### GraphQL — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| Auth URL queries | `googleAuthUrl(input)`, `githubAuthUrl(input)` → `OAuthUrlPayload` | ✅ |
| `OAuthUrlInput` | `redirectUrl`, `dry` (for getting callbackUrl without active creds) | ✅ |
| `OAuthUrlPayload` | `authUrl`, `callbackUrl` | ✅ |
| Admin types | `AdminGoogleOAuthCredentials`, `AdminGitHubOAuthCredentials` | ✅ |
| Admin queries | `allGoogleOAuthCredentials`, `allGitHubOAuthCredentials` | ✅ |
| Admin mutations | create/delete/setActive/deactivate for both providers | ✅ |
| Resolvers | `internal/graph/schema.resolvers.go` | ✅ |
| Removed | `PublicSettings` type (was `googleAuthEnabled`, `githubAuthEnabled`) | ✅ |

### Admin Cases — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| Create Google | `internal/case/admin/creategoogleoauthcredentials/` | ✅ |
| Delete Google | `internal/case/admin/deletegoogleoauthcredentials/` | ✅ |
| SetActive Google | `internal/case/admin/setactivegoogleoauthcredentials/` | ✅ |
| Deactivate Google | `internal/case/admin/deactivategoogleoauth/` | ✅ |
| Create GitHub | `internal/case/admin/creategithuboauthcredentials/` | ✅ |
| Delete GitHub | `internal/case/admin/deletegithuboauthcredentials/` | ✅ |
| SetActive GitHub | `internal/case/admin/setactivegithuboauthcredentials/` | ✅ |
| Deactivate GitHub | `internal/case/admin/deactivategithuboauth/` | ✅ |

### OAuth Clients — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| Google standalone functions | `internal/googleauth/client.go` | ✅ |
| GitHub standalone functions | `internal/githubauth/client.go` | ✅ |
| `BuildAuthURL()` | Both packages | ✅ |
| `ExchangeCode()` | Both packages | ✅ |
| `GetUserInfo()` / `GetPrimaryVerifiedEmail()` | Both packages | ✅ |

### Endpoints — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| Google start | `internal/case/handlegooglestart/endpoint.go` | ✅ |
| Google callback | `internal/case/handlegooglecallback/endpoint.go` | ✅ |
| GitHub start | `internal/case/handlegithubstart/endpoint.go` | ✅ |
| GitHub callback | `internal/case/handlegithubcallback/endpoint.go` | ✅ |

### Server Integration — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| `BuildGoogleAuthURL()` | `cmd/server/main.go` | ✅ |
| `BuildGitHubAuthURL()` | `cmd/server/main.go` | ✅ |
| Remove old `GoogleAuthEnabled()` | `cmd/server/main.go` | ✅ |
| Remove old `GitHubAuthEnabled()` | `cmd/server/main.go` | ✅ |
| Remove CLI flags | `internal/appconfig/config.go` | ✅ |
| Remove embedded OAuth clients | `cmd/server/main.go` | ✅ |

### Frontend — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| OAuth buttons component | `assets/ui/auth/oauth/buttons/` | ✅ |
| Query `googleAuthUrl`/`githubAuthUrl` | Uses `OAuthUrlInput` with `authUrl` response | ✅ |
| Remove old settings cache | `assets/ui/settings/settings.ts` — removed `$trip2g_settings_public` | ✅ |
| Admin Google OAuth catalog | `assets/ui/admin/oauth/google/catalog/` | ✅ |
| Admin Google OAuth create | `assets/ui/admin/oauth/google/create/` | ✅ |
| Admin GitHub OAuth catalog | `assets/ui/admin/oauth/github/catalog/` | ✅ |
| Admin GitHub OAuth create | `assets/ui/admin/oauth/github/create/` | ✅ |
| Disable all pages | `assets/ui/admin/oauth/*/disableall/` | ✅ |
| Register app links | Links to Google Cloud Console / GitHub Developer Settings | ✅ |
| Callback URL display | Shows callback URL with Copy button (from backend via `dry: true`) | ✅ |

### Tests — DONE ✅

| Component | File | Status |
|-----------|------|--------|
| Create Google tests | `internal/case/admin/creategoogleoauthcredentials/resolve_test.go` | ✅ |
| Create GitHub tests | `internal/case/admin/creategithuboauthcredentials/resolve_test.go` | ✅ |

## Architecture

### Why Encrypted Storage?

Secrets stored encrypted in DB so that **database backups are safe**. Without the encryption key, backup contains only ciphertext.

| Storage | Backup contains | Risk |
|---------|-----------------|------|
| Env vars | N/A (not in DB) | Secrets in deploy configs |
| DB plaintext | Raw secrets | Backup leak = compromise |
| DB encrypted | Ciphertext | Backup leak = useless |

### Encryption

- Uses `internal/dataencryption/` package (AES-256-GCM)
- Key from `--data-encryption-key` flag (32 bytes)
- One master key encrypts all secrets

### Loading Credentials

Query DB on every OAuth call (no caching):
- OAuth logins are infrequent (user action)
- SQLite handles queries in microseconds
- No cache invalidation complexity

## Database Schema

### google_oauth_credentials

| Field | Type | Description |
|-------|------|-------------|
| id | integer | Primary key |
| name | text | Display name (e.g. "Production") |
| client_id | text | Google OAuth Client ID |
| client_secret_encrypted | blob | Encrypted Client Secret |
| active | boolean | Only one should be active |
| created_at | datetime | Creation timestamp |
| created_by | integer | FK to users.id |

### github_oauth_credentials

Same structure as Google.

## GraphQL API

### Queries (public)

```graphql
input OAuthUrlInput {
  redirectUrl: String!
  dry: Boolean  # If true, returns callbackUrl even if not configured
}

type OAuthUrlPayload {
  authUrl: String      # Full OAuth URL or null if not configured
  callbackUrl: String! # Always returned (for admin UI)
}

googleAuthUrl(input: OAuthUrlInput!): OAuthUrlPayload!
githubAuthUrl(input: OAuthUrlInput!): OAuthUrlPayload!
```

### Admin Queries

```graphql
allGoogleOAuthCredentials: AdminGoogleOAuthCredentialsConnection!
allGitHubOAuthCredentials: AdminGitHubOAuthCredentialsConnection!
```

### Admin Mutations

```graphql
# Google
createGoogleOAuthCredentials(input: CreateGoogleOAuthCredentialsInput!): ...
deleteGoogleOAuthCredentials(input: DeleteGoogleOAuthCredentialsInput!): ...
setActiveGoogleOAuthCredentials(input: SetActiveGoogleOAuthCredentialsInput!): ...
deactivateGoogleOAuth: DeactivateGoogleOAuthOrErrorPayload!

# GitHub
createGitHubOAuthCredentials(input: CreateGitHubOAuthCredentialsInput!): ...
deleteGitHubOAuthCredentials(input: DeleteGitHubOAuthCredentialsInput!): ...
setActiveGitHubOAuthCredentials(input: SetActiveGitHubOAuthCredentialsInput!): ...
deactivateGitHubOAuth: DeactivateGitHubOAuthOrErrorPayload!
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/_system/auth/google?redirect=/path` | Start Google OAuth |
| GET | `/_system/auth/google/callback` | Google callback |
| GET | `/_system/auth/github?redirect=/path` | Start GitHub OAuth |
| GET | `/_system/auth/github/callback` | GitHub callback |

## Flow

```
Frontend queries googleAuthUrl(redirectUrl: "/current/page")
    ↓
Server returns "https://domain/_system/auth/google?redirect=/current/page"
(or null if not configured)
    ↓
User clicks button, navigates to URL
    ↓
Endpoint loads credentials from DB, generates state, sets cookie
    ↓
Redirect to Google/GitHub
    ↓
User authorizes
    ↓
Callback validates state, exchanges code, gets email
    ↓
UserByEmail(email) → set JWT cookie → redirect
```

## Error Handling

| Error | Redirect | Description |
|-------|----------|-------------|
| OAuth not configured | `/?berror=oauth_not_configured` | No active credentials in DB |
| User not found | `/?berror=user_not_found` | Email not in database |
| OAuth error | `/?berror=oauth_failed` | Provider error |
| Invalid state | `/?berror=invalid_state` | CSRF validation failed |

## Getting OAuth Credentials

### Google

1. [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials
2. Create OAuth client ID (Web application)
3. Add redirect URI: `https://yourdomain.com/_system/auth/google/callback`

### GitHub

1. [GitHub Developer Settings](https://github.com/settings/developers) → New OAuth App
2. Set callback URL: `https://yourdomain.com/_system/auth/github/callback`
