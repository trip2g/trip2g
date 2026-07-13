---
title: "Subgraphs for agents"
free: true
lang_redirect: "[[ru/user/subgraph_for_agents]]"
---

This is a starting point, not a full guide — it will grow as more agent workflows land. Right now it shows the minimum an agent needs to manage subgraphs through GraphQL instead of the admin panel.

### Prerequisite: an API key with admin GraphQL enabled

By default an API key gets read-only MCP tools. To let an agent run admin mutations (create subgraphs, grant access, anything under `admin { }`), the key needs `enable_mcp_admin_tools = true`.

If you're onboarding from the sample vault, add `?enable_admin_graphql` to the onboarding-vault download URL and the issued key already has it. Otherwise flip it by hand in **Admin → API Keys**. See [[en/user/graphql]] for the general auth setup and [[en/user/mcp]] for the MCP tool names (`graphql_introspection`, `graphql_request`).

### Create a subgraph

There's no dedicated "create subgraph" mutation — a subgraph exists once a note carrying it syncs in. An agent creates one by upserting a note through `updateNotes`:

```graphql
mutation {
  updateNotes(input: {
    changes: [{
      upsert: {
        path: "subgraphs/team-status.md"
        content: "---\nsubgraph: team-status\n---\nInternal status notes, team access only."
      }
    }]
  }) {
    __typename
    ... on UpdateNotesSuccessPayload { paths }
    ... on ErrorPayload { message }
  }
}
```

The `team-status` subgraph now exists. Confirm with `admin { allSubgraphs { nodes { id name } } }` to get its `id` for the next step.

### Grant a user access

```graphql
mutation {
  admin {
    createUserSubgraphAccess(input: {
      userId: 6
      subgraphIds: [3]
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

If the user doesn't exist yet, create it first with `admin { createUser(input: { email: "..." }) }` — full walkthrough in [[en/user/user_management]].

### Related

- [[en/user/subgraphs]] — what subgraphs are and how to use them from the admin panel
- [[en/user/user_management]] — the full user/subgraph mutation workflow
- [[en/user/graphql]] — GraphQL endpoint, auth methods
- [[en/user/federation]] — scoping a peer agent's search to specific subgraphs
