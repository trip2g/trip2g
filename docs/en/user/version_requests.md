---
title: Recovering overwritten notes
free: true
lang_redirect: "[[ru/user/version_requests]]"
---

A note was accidentally overwritten — content is gone, the current version is wrong. trip2g keeps every version of every note. You can retrieve any past version using two GraphQL queries via `graphql_request`.

This requires admin tools enabled on the API key. See [[en/user/agent_admin]]. The queries can also be run directly in the GraphiQL interface at `/_system/graphql` — see [[en/user/graphql]].

### Step 1. Find the overwrite

List all saved versions for the note. Versions are returned newest-first.

```graphql
query {
  admin {
    noteVersionHistory(filter: { path: "folder/my-note.md" }) {
      totalCount
      nodes {
        versionId
        version
        contentLength
        createdAt
      }
    }
  }
}
```

Example response:

```json
{
  "noteVersionHistory": {
    "totalCount": 5,
    "nodes": [
      { "versionId": 204, "version": 5, "contentLength": 312,  "createdAt": "2026-05-25T14:02:00Z" },
      { "versionId": 198, "version": 4, "contentLength": 318,  "createdAt": "2026-05-24T09:15:00Z" },
      { "versionId": 171, "version": 3, "contentLength": 4821, "createdAt": "2026-05-23T11:40:00Z" },
      { "versionId": 155, "version": 2, "contentLength": 4790, "createdAt": "2026-05-22T08:30:00Z" },
      { "versionId": 138, "version": 1, "contentLength": 3102, "createdAt": "2026-05-20T17:10:00Z" }
    ]
  }
}
```

Scan `contentLength` by date. The drop from 4821 bytes (version 3) to 318 bytes (version 4) marks the overwrite. Version 3 (`versionId: 171`) is the last good copy.

### Step 2. Fetch the content

```graphql
query {
  admin {
    noteVersion(versionId: 171) {
      versionId
      path
      version
      content
      createdAt
    }
  }
}
```

The `content` field contains the raw markdown. Copy it back into Obsidian and sync.

### Pagination

For notes with many versions, use `limit` and `offset`:

```graphql
noteVersionHistory(filter: { path: "folder/my-note.md", limit: 20, offset: 40 })
```

The default page size is 50. `totalCount` tells you how many versions exist in total.
