---
title: "Frontend: $mol"
free: true
lang_redirect: "[[ru/user/mol]]"
---

The trip2g frontend is built with [$mol](https://mol.hyoo.ru) — a reactive UI framework developed by Dmitry Karlovskiy. It covers two distinct surfaces: the **admin panel** used by site owners and **user-facing widgets** embedded into the default template (login, search, paywall, etc.).

## What is $mol

$mol is a component framework with a reactive system based on fibers. Key differences from React/Vue:

- No virtual DOM — components bind directly to real DOM nodes
- All application code is written synchronously; async operations integrate transparently via `$mol_wire_sync`
- Components are described declaratively in `.view.tree` files; behavior lives in `.view.ts`
- Built-in i18n: strings marked with `@` are extracted to `*.view.tree.locale=en.json` files
- Part of the [MAM monorepo](https://github.com/hyoo-ru/mam) — modules are resolved by directory name

The framework is small (~10k lines), modular, and self-hosting — the build tool is itself written in $mol.

## Directory layout

```
assets/ui/          ← symlink → ../mam/trip2g/
  admin/            ← admin panel component tree
  auth/             ← login / sign-out
  user/             ← reader-facing widgets
    search/         ← site search
    space/          ← subscription space
  graphql/          ← generated types + mol adapter
  monkeypatch/      ← runtime patches (locale, URL)
  externaldeps/     ← Docker layer cache trick
```

`assets/ui/` is a symlink into the MAM workspace. All component names are prefixed `$trip2g_`.

## Admin panel

The admin panel (`$trip2g_admin`) wraps the auth flow (`$trip2g_auth`) and renders a multi-section catalog after login. Sections: dashboard, users, content, telegram, monetization, integrations, SEO, system. Navigation is URL-driven via `$mol_state_arg`.

Each admin section follows the same CRUD pattern:

| Directory | Purpose |
|-----------|---------|
| `catalog/` | Table of all entities |
| `show/` | Detail view of one entity |
| `update/` | Edit form |
| `create/` | Creation form (when applicable) |
| `button/` | Action buttons (delete, run, etc.) |

The footer contains a language switcher (`$mol_locale_select`, supports `ru`/`en`) and a light/dark theme toggle (`$mol_lights_toggle`).

## User-facing widgets

These are embedded by the default template into regular site pages:

| Component | Purpose |
|-----------|---------|
| `$trip2g_auth` | Email → code sign-in; OAuth buttons |
| `$trip2g_user_search` | Full-text + vector site search |
| `$trip2g_user_paywall` | Subscription / access gate |
| `$trip2g_user_signinwall` | Sign-in prompt before paywall |
| `$trip2g_user_favoritenote` | Favorite toggle |

The auth flow: user enters email → server sends a 6-digit code → user enters code → JWT stored in cookie. OAuth (Google, GitHub) is supported as an alternative.

The search widget submits a query to the GraphQL `siteSearch` operation, renders results with highlighted excerpts, and shows a vector-search warning when no exact matches exist.

## GraphQL integration

Trip2g uses [gqlgen](https://gqlgen.com/) on the backend and a custom code generator (`graphqlmol.js`) on the frontend. The generator reads `schema.graphqls` and `queries.ts` and produces:

1. Full TypeScript types for every query/mutation/subscription
2. A `$trip2g_graphql_request` adapter that integrates with $mol's reactive fiber system

Usage pattern — **always call the factory outside the class** so the request object is shared across component instances:

```typescript
// Module-level: creates a reusable reactive request
const data_request = $trip2g_graphql_request(/* GraphQL */ `
    query AdminBackgroundQueues {
        admin {
            allBackgroundQueues {
                nodes { id pendingCount stopped }
            }
        }
    }
`)

export class $trip2g_admin_backgroundqueue_catalog extends ... {
    @$mol_mem
    data(reset?: null) {
        // Calling data_request() here participates in mol's reactive graph.
        // On first call it fires the HTTP request; subsequent calls return
        // the cached result. Passing reset=null invalidates the cache.
        const res = data_request()
        return $trip2g_graphql_make_map(res.admin.allBackgroundQueues.nodes)
    }
}
```

`$trip2g_graphql_make_map` converts the nodes array to a `Map<id, node>` that $mol catalog components can iterate reactively.

After editing `schema.graphqls` or `internal/db/queries.sql`, regenerate frontend types:

```bash
npm run graphqlgen
```

## Local development

The MAM build tool resolves `$trip2g_*` modules by looking for matching directories in the MAM workspace. Setup:

```bash
git clone https://github.com/hyoo-ru/mam.git
ln -s /path/to/trip2g/assets/ui mam/trip2g

cd mam
npm start trip2g/admin   # or trip2g/user, trip2g/forms, etc.
```

**GraphQL variable name collision**: MAM treats every `$name` token in a GraphQL query as a potential module reference and tries to resolve the directory. For variables like `$filter`, `$id`, `$input` you must pre-create empty stub directories:

```bash
cd mam
mkdir filter id input limit format fragment note
```

This is a known quirk — the directories are never used by the build, but their absence causes resolution errors.

## Docker build and layer caching

Building all $mol dependencies is slow. The trick in `Dockerfile` is to split the build into two Docker layers:

1. **Layer 1** — copy only `externaldeps/list.view.tree` and run `npm start trip2g/externaldeps`. This builds the entire mol stdlib and caches it in a Docker layer.
2. **Layer 2** — copy the rest of `assets/ui/` and build the actual app components.

As long as `externaldeps/list.view.tree` doesn't change, layer 1 is reused from cache and subsequent builds skip re-downloading and recompiling mol itself.

`externaldeps/list.view.tree` is a dummy $mol component that lists every mol module the project depends on. Keep it up to date when adding new `$mol_*` primitives.

## Monkeypatches

`assets/ui/monkeypatch/monkeypatch.ts` applies two patches at runtime:

**1. Locale cache busting**

Locale JSON files (`web.locale=ru.json`) are served with long-lived cache headers. The patch appends the JS bundle hash to the locale URL:

```
web.locale=ru.json?h=<bundle-hash>
```

The hash is read from the `?h=` query parameter of the currently executing `<script src="...?h=...">` tag. When the bundle changes, the hash changes, and browsers fetch fresh locale files.

**2. Remove trailing `#!` from links**

$mol uses hashbang URLs (`#!`) for client-side routing. When mol is used for only part of a page (widgets, not a full SPA), every internal link gains a trailing `#!` — visible in the browser address bar. The patch overrides `$mol_state_arg.make_link` to strip the trailing `#!`, producing clean URLs like `https://trip2g.com/ru/user/protocol` instead of `https://trip2g.com/ru/user/protocol#!`.

`$trip2g_monkeypatch_apply()` must be called once during app bootstrap, before any component renders.
