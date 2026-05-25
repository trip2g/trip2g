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
