# Caddy (Dev Proxy)

## Why Caddy

Two purposes in development:

1. **Frontend dev server isolation** — `http://localhost:9081` proxies `/graphql`, `/api/*`, `/_system/*`, `/assets/*` to the Go server (`:8081`), and everything else to the $mol dev server (`:9080`). This lets you develop frontend with hot reload while still talking to the real backend.

2. **HTTPS for third-party integrations** — `https://localhost:7081` provides TLS for features that require secure context (Cloudflare Turnstile captcha, OAuth callbacks, etc.). All traffic goes directly to the Go server.

3. **Same-origin service proxies** — `/_fleet/*` and `/_codellm/*` reverse-proxy to the fleet and codellm services on their private ports, forwarding the session cookie. Caddy does NO auth on these routes; each service gates itself with the delegated admin middleware (`internal/delegatedadmin`), which forwards the caller's cookie to the monolith's `viewer{role}` and allows only admins (fail-closed). The upstream addresses in the Caddyfile are placeholders — the real private addrs come from deploy config. See `fleet_graphql.md`.

## Setup

```bash
# Install Caddy
brew install caddy  # macOS
# or: https://caddyserver.com/docs/install

# Generate local TLS certs (one-time)
mkcert -install
mkcert localhost 127.0.0.1

# Run
caddy run
```

## Ports

| Port | Protocol | Backend | Use |
|------|----------|---------|-----|
| `9081` | HTTP | Go (`:8081`) + $mol (`:9080`) | Frontend development with hot reload |
| `7081` | HTTPS | Go (`:8081`) only | Testing Turnstile, OAuth, secure-context features |
| `8081` | HTTP | Go server directly | API, backend-only work |
| `9080` | HTTP | $mol dev server | Frontend build/serve (not used directly) |

## When to use which

- **Day-to-day frontend dev**: `http://localhost:9081`
- **Testing captcha / OAuth / HTTPS features**: `https://localhost:7081`
- **Backend-only work**: `http://localhost:8081` (no Caddy needed)
