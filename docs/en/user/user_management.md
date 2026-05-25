---
title: Managing users via GraphQL
lang_redirect: "[[ru/user/user_management]]"
---

You need to provision a subscriber, grant a team member access to a subgraph, or make someone an admin — without opening the browser. All three operations are available as GraphQL mutations. They're designed to be driven by an agent or a script.

This requires admin API access. See [[en/user/graphql]] for authentication setup.

### Typical workflow

1. Create a note with `subgraph: name` frontmatter → sync → subgraph appears
2. `createUser(email)` → get `userId`
3. `allSubgraphs` → get `subgraphIds`
4. `createUserSubgraphAccess(userId, subgraphIds)` → grant content access
5. Optionally `createAdmin(userId)` → full admin rights

### Step 1. Create a user

```graphql
mutation {
  admin {
    createUser(input: { email: "user@example.com" }) {
      __typename
      ... on CreateUserPayload {
        user { id email }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

The response includes the new user's `id`. You need it for every subsequent mutation.

### Step 2. Get subgraph IDs

Subgraphs are created by syncing a note that declares one in its frontmatter:

```yaml
---
subgraph: premium
---
```

Or multiple subgraphs at once:

```yaml
---
subgraphs: [premium, beta]
---
```

After syncing from Obsidian, the subgraph appears in the system. List all subgraphs to get their IDs:

```graphql
{ admin { allSubgraphs { nodes { id name } } } }
```

### Step 3. Grant subgraph access

```graphql
mutation {
  admin {
    createUserSubgraphAccess(input: {
      userId: 6
      subgraphIds: [1]
      expiresAt: null
    }) {
      __typename
      ... on CreateUserSubgraphAccessPayload {
        accesses { id userId subgraphId expiresAt }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

`subgraphIds` is an array — you can grant access to multiple subgraphs in one call. Omit `expiresAt` (or pass `null`) for permanent access. Pass an ISO 8601 timestamp for time-limited access.

### Step 4. Make a user admin (optional)

```graphql
mutation {
  admin {
    createAdmin(input: { userId: 6 }) {
      __typename
      ... on CreateAdminPayload {
        admin { id user { id email } }
      }
      ... on ErrorPayload { message }
    }
  }
}
```

Admin status gives full access to the admin panel and all GraphQL mutations. Use it for team members who need to manage the site, not for regular subscribers.

### Warning: notes without a subgraph are visible to all signed-in users

By default, any note that has no `subgraph` or `subgraphs` in its frontmatter is visible to every authenticated user — including users you grant access to only one specific subgraph.

**Fix: hide unassigned notes behind a system subgraph.**

Create a vault-based frontmatter patch (`_private.md` or similar):

```yaml
---
type: frontmatter-patch
include: ["**"]
priority: 0
---
```

````jsonnet
if !std.objectHas(meta, 'subgraph') && !std.objectHas(meta, 'subgraphs')
then meta + { subgraph: 'notes_without_subgraph' }
else meta
````

This assigns every untagged note to the `notes_without_subgraph` subgraph. Since you never grant anyone access to it, those notes are invisible to regular users. Admins always see everything regardless of subgraph assignment.

See [[en/user/frontmatter-patches]] for the full patch syntax.

### Use case: share a single page with one person by email

You want to give someone access to one specific note — not the whole site, not a paid subgraph. The cleanest way:

**1. Create the note**

```
shares/alice/intro.md
```

Frontmatter:
```yaml
---
subgraph: alice
---
```

Sync from Obsidian. The `alice` subgraph appears automatically.

**2. Provision the user**

```graphql
# Create the user
mutation { admin { createUser(input: { email: "alice@example.com" }) {
  ... on CreateUserPayload { user { id } }
} } }

# Get the subgraph ID (look for name = "alice")
{ admin { allSubgraphs { nodes { id name } } } }

# Grant access
mutation { admin { createUserSubgraphAccess(input: { userId: <id>, subgraphIds: [<subgraphId>] }) {
  ... on CreateUserSubgraphAccessPayload { accesses { id } }
} } }
```

**3. Send the user a message**

> Hi Alice,
>
> I've shared a page with you: **https://yoursite.com/shares/alice/intro**
>
> To open it, click the link and sign in with this email. You'll receive a one-time code — no password needed.

That's it. Alice sees only the `alice` subgraph. Add more notes to `shares/alice/` as needed.

### Lookup queries

If you don't know a user's `id` or a subgraph's `id`, look them up first.

```graphql
# Find a user's ID by email
{ admin { allUsers { nodes { id email } } } }

# List subgraphs with their IDs
{ admin { allSubgraphs { nodes { id name } } } }

# Check a user's current accesses
{ admin { allUserSubgraphAccesses { nodes { id userId subgraphId expiresAt } } } }
```
