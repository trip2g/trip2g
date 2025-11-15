# API Contracts Documentation

**Generated:** 2025-11-15
**Source:** `/internal/graph/schema.graphqls`
**API Type:** GraphQL

## Overview

trip2g exposes a comprehensive GraphQL API with two main entry points:
- **Public API** - User-facing queries and mutations for content access, authentication, and payments
- **Admin API** - Administrative interface for platform management

## Public API

### Authentication & User Management

**Sign In Flow:**
```graphql
mutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {
  requestEmailSignInCode(input: $input)
}

mutation SignInByEmail($input: SignInByEmailInput!) {
  signInByEmail(input: $input) {
    ... on SignInPayload {
      token
      viewer { id role }
    }
  }
}

mutation SignOut {
  signOut {
    ... on SignOutPayload {
      viewer { id role }
    }
  }
}
```

### Content Access

**Retrieve Notes:**
```graphql
query GetNote($input: NoteInput!) {
  note(input: $input) {
    pathId
    title
    html
    toc { id title level }
  }
}

query Search($input: SearchInput!) {
  search(input: $input) {
    totalCount
    nodes {
      highlightedTitle
      highlightedContent
      url
      document { ... on PublicNote { title html } }
    }
  }
}
```

**API Key Protected Endpoints:**
```graphql
# X-Api-Key header required
query NotePaths($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    value
    latestContentHash
    latestNoteView { title content }
  }
}

mutation PushNotes($input: PushNotesInput!) {
  pushNotes(input: $input) {
    ... on PushNotesPayload {
      notes { id path assets { path sha256Hash } }
    }
  }
}

mutation UploadNoteAsset($input: UploadNoteAssetInput!) {
  uploadNoteAsset(input: $input) {
    ... on UploadNoteAssetPayload {
      uploadSkipped
    }
  }
}

mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input)
}
```

### Payments & Subscriptions

```graphql
query GetViewer {
  viewer {
    offers(filter: { pageId: 123 }) {
      ... on ActiveOffers {
        nodes {
          id
          priceUSD
          subgraphs { name }
        }
      }
      ... on SubgraphWaitList {
        tgBotUrl
        emailAllowed
      }
    }
    activePurchases {
      id
      status
      successful
    }
  }
}

mutation CreatePaymentLink($input: CreatePaymentLinkInput!) {
  createPaymentLink(input: $input) {
    ... on CreatePaymentLinkPayload {
      redirectUrl
      token
    }
  }
}
```

**Payment Types:**
- `CRYPTO` - NowPayments cryptocurrency payments

### User Features

```graphql
mutation ToggleFavoriteNote($input: ToggleFavoriteNoteInput!) {
  toggleFavoriteNote(input: $input) {
    ... on ToggleFavoriteNotePayload {
      success
      favoriteNotes { title html }
    }
  }
}

mutation GenerateTgAttachCode($input: GenerateTgAttachCodeInput!) {
  generateTgAttachCode(input: $input) {
    ... on GenerateTgAttachCodePayload {
      code
      url
    }
  }
}

mutation CreateEmailWaitListRequest($input: CreateEmailWaitListRequestInput!) {
  createEmailWaitListRequest(input: $input)
}
```

## Admin API

Accessed via `query { admin { ... } }` and `mutation { admin { ... } }`

### User & Access Management

**Users:**
```graphql
query AllUsers {
  admin {
    allUsers {
      nodes {
        id
        email
        createdAt
        ban { reason bannedBy { email } }
      }
    }
  }
}

mutation BanUser($input: BanUserInput!) {
  admin {
    banUser(input: $input) {
      ... on BanUserPayload {
        userId
        user { email }
      }
    }
  }
}

mutation UnbanUser($input: UnbanUserInput!) {
  admin {
    unbanUser(input: $input)
  }
}
```

**Subgraphs & Access Control:**
```graphql
query AllSubgraphs {
  admin {
    allSubgraphs {
      nodes {
        id
        name
        color
        hidden
      }
    }
    allUserSubgraphAccesses {
      nodes {
        id
        userId
        subgraphId
        createdAt
        expiresAt
        user { email }
        subgraph { name }
      }
    }
  }
}

mutation UpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {
  admin {
    updateUserSubgraphAccess(input: $input)
  }
}
```

### Content Management

**Notes & Releases:**
```graphql
query AllLatestNoteViews($filter: AdminLatestNoteViewsFilter) {
  admin {
    allLatestNoteViews(filter: $filter) {
      nodes {
        id
        path
        title
        content
        html
        permalink
        free
        subgraphNames
        warnings { level message }
      }
    }
  }
}

query AllReleases {
  admin {
    allReleases {
      nodes {
        id
        title
        createdAt
        createdBy { email }
        isLive
        homeNote { title }
      }
    }
  }
}

mutation CreateRelease($input: CreateReleaseInput!) {
  admin {
    createRelease(input: $input)
  }
}

mutation MakeReleaseLive($input: MakeReleaseLiveInput!) {
  admin {
    makeReleaseLive(input: $input)
  }
}
```

**Graph Positions:**
```graphql
mutation UpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {
  admin {
    updateNoteGraphPositions(input: $input) {
      ... on UpdateNoteGraphPositionsPayload {
        success
        updatedNoteViews { path graphPosition { x y } }
      }
    }
  }
}
```

### Offers & Payments

```graphql
query AllOffers {
  admin {
    allOffers {
      nodes {
        id
        publicId
        priceUSD
        lifetime
        startsAt
        endsAt
        subgraphs { name }
      }
    }
    allPurchases {
      nodes {
        id
        createdAt
        paymentProvider
        status
        successful
        email
        offer { priceUSD }
        user { email }
      }
    }
  }
}

mutation CreateOffer($input: CreateOfferInput!) {
  admin {
    createOffer(input: $input)
  }
}

mutation UpdateOffer($input: UpdateOfferInput!) {
  admin {
    updateOffer(input: $input)
  }
}
```

### Telegram Integration

**Bots & Chats:**
```graphql
query AllTgBots {
  admin {
    allTgBots {
      nodes {
        id
        enabled
        name
        description
      }
    }
    tgBotChats(filter: { botId: 1 }) {
      nodes {
        id
        chatType
        chatTitle
        memberCount
        subgraphAccesses { subgraph { name } }
        subgraphInvites { subgraph { name } }
      }
    }
  }
}

mutation CreateTgBot($input: CreateTgBotInput!) {
  admin {
    createTgBot(input: $input)
  }
}

mutation SetTgChatSubgraphs($input: SetTgChatSubgraphsInput!) {
  admin {
    setTgChatSubgraphs(input: $input)
  }
}

mutation RemoveExpiredTgChatMembers($input: RemoveExpiredTgChatMembersInput!) {
  admin {
    removeExpiredTgChatMembers(input: $input) {
      ... on RemoveExpiredTgChatMembersPayload {
        removedCount
        errors
      }
    }
  }
}
```

**Telegram Publishing:**
```graphql
query AllTelegramPublishNotes($filter: AdminTelegramPublishNotesFilter) {
  admin {
    allTelegramPublishNotes(filter: $filter) {
      nodes {
        id
        publishAt
        secondsUntilPublish
        publishedAt
        status
        errorCount
        noteView { title path }
        post { content warnings }
        tags { label }
        chats { chatTitle }
      }
      count
    }
  }
}

mutation SetTgChatPublishTags($input: SetTgChatPublishTagsInput!) {
  admin {
    setTgChatPublishTags(input: $input)
  }
}

mutation ResetTelegramPublishNote($input: ResetTelegramPublishNoteInput!) {
  admin {
    resetTelegramPublishNote(input: $input)
  }
}

mutation SendTelegramPublishNoteNow($input: SendTelegramPublishNoteNowInput!) {
  admin {
    sendTelegramPublishNoteNow(input: $input)
  }
}
```

### Patreon Integration

```graphql
query AllPatreonCredentials($filter: AdminPatreonCredentialsFilterInput) {
  admin {
    allPatreonCredentials(filter: $filter) {
      nodes {
        id
        createdAt
        syncedAt
        state
        tiers {
          nodes {
            id
            tierID
            title
            amountCents
            subgraphs { name }
          }
        }
        members {
          nodes {
            id
            patreonID
            email
            status
            currentTier { title }
          }
        }
      }
    }
  }
}

mutation CreatePatreonCredentials($input: CreatePatreonCredentialsInput!) {
  admin {
    createPatreonCredentials(input: $input)
  }
}

mutation RefreshPatreonData($input: RefreshPatreonDataInput!) {
  admin {
    refreshPatreonData(input: $input)
  }
}

mutation SetPatreonTierSubgraphs($input: SetPatreonTierSubgraphsInput!) {
  admin {
    setPatreonTierSubgraphs(input: $input)
  }
}
```

### Boosty Integration

```graphql
query AllBoostyCredentials($filter: AdminBoostyCredentialsFilterInput) {
  admin {
    allBoostyCredentials(filter: $filter) {
      nodes {
        id
        blogName
        state
        tiers {
          nodes {
            id
            boostyId
            name
            subgraphs { name }
          }
        }
        members {
          nodes {
            id
            boostyId
            email
            status
            currentTier { name }
          }
        }
      }
    }
  }
}

mutation CreateBoostyCredentials($input: CreateBoostyCredentialsInput!) {
  admin {
    createBoostyCredentials(input: $input)
  }
}

mutation RefreshBoostyData($input: RefreshBoostyDataInput!) {
  admin {
    refreshBoostyData(input: $input)
  }
}

mutation SetBoostyTierSubgraphs($input: SetBoostyTierSubgraphsInput!) {
  admin {
    setBoostyTierSubgraphs(input: $input)
  }
}
```

### System Management

**API Keys & Git Tokens:**
```graphql
query AllApiKeys {
  admin {
    allApiKeys {
      nodes {
        id
        description
        createdAt
        createdBy { email }
        disabledAt
      }
    }
    apiKeyLogs(filter: { apiKeyId: 1 }) {
      nodes {
        createdAt
        actionName
        ip
      }
    }
  }
}

mutation CreateApiKey($input: CreateApiKeyInput!) {
  admin {
    createApiKey(input: $input) {
      ... on CreateApiKeyPayload {
        value
        apiKey { id description }
      }
    }
  }
}

mutation CreateGitToken($input: CreateGitTokenInput!) {
  admin {
    createGitToken(input: $input) {
      ... on CreateGitTokenPayload {
        value
        gitToken { id canPull canPush }
      }
    }
  }
}
```

**Cron Jobs:**
```graphql
query AllCronJobs {
  admin {
    allCronJobs {
      nodes {
        id
        name
        enabled
        expression
        lastExecAt
        executions {
          id
          startedAt
          finishedAt
          status
          errorMessage
        }
      }
    }
  }
}

mutation UpdateCronJob($input: UpdateCronJobInput!) {
  admin {
    updateCronJob(input: $input)
  }
}

mutation RunCronJob($input: RunCronJobInput!) {
  admin {
    runCronJob(input: $input)
  }
}
```

**Background Queues:**
```graphql
query AllBackgroundQueues {
  admin {
    allBackgroundQueues {
      nodes {
        id
        pendingCount
        retryCount
        stopped
        jobs {
          id
          name
          params
          priority
          retryCount
        }
      }
    }
  }
}

mutation StopBackgroundQueue($input: StopBackgroundQueueInput!) {
  admin {
    stopBackgroundQueue(input: $input)
  }
}

mutation StartBackgroundQueue($input: StartBackgroundQueueInput!) {
  admin {
    startBackgroundQueue(input: $input)
  }
}

mutation ClearBackgroundQueue($input: ClearBackgroundQueueInput!) {
  admin {
    clearBackgroundQueue(input: $input)
  }
}
```

**Audit Logs:**
```graphql
query AuditLogs($filter: AdminAuditLogsFilterInput!) {
  admin {
    auditLogs(filter: $filter) {
      nodes {
        id
        createdAt
        level
        message
        params
      }
    }
  }
}
```

**Redirects & 404 Tracking:**
```graphql
query AllRedirects {
  admin {
    allRedirects {
      nodes {
        id
        pattern
        ignoreCase
        isRegex
        target
        createdBy { email }
      }
    }
    allNotFoundPaths {
      nodes {
        id
        path
        totalHits
        lastHitAt
      }
    }
  }
}

mutation CreateRedirect($input: CreateRedirectInput!) {
  admin {
    createRedirect(input: $input)
  }
}

mutation CreateNotFoundIgnoredPattern($input: CreateNotFoundIgnoredPatternInput!) {
  admin {
    createNotFoundIgnoredPattern(input: $input)
  }
}
```

**Configuration:**
```graphql
query LatestConfig {
  admin {
    latestConfig {
      id
      showDraftVersions
      defaultLayout
      timezone
      robotsTxt
      createdBy { email }
    }
  }
}

mutation CreateConfigVersion($input: CreateConfigVersionInput!) {
  admin {
    createConfigVersion(input: $input)
  }
}
```

**HTML Injections:**
```graphql
query AllHtmlInjections {
  admin {
    allHtmlInjections {
      nodes {
        id
        description
        position
        placement
        content
        activeFrom
        activeTo
      }
    }
  }
}

mutation CreateHtmlInjection($input: CreateHtmlInjectionInput!) {
  admin {
    createHtmlInjection(input: $input)
  }
}
```

**Wait Lists:**
```graphql
query AllWaitListRequests {
  admin {
    allWaitListEmailRequests {
      nodes {
        email
        createdAt
        ip
        notePath
      }
    }
    allWaitListTgBotRequests {
      nodes {
        chatId
        createdAt
        notePath
        botName
      }
    }
  }
}
```

## Core Types

### NoteView
Primary content model representing a published note:
```graphql
type NoteView {
  id: String!
  path: String!
  pathId: Int64!
  title: String!
  content: String!         # Raw markdown
  html: String!            # Rendered HTML
  permalink: String!
  free: Boolean!
  versionId: Int64!
  subgraphNames: [String!]!
  warnings: [NoteWarning!]!
  inLinks: [NoteView!]!    # Backlinks
  graphPosition: Vector2   # Graph visualization position
  isHomePage: Boolean!
  description: String
  meta: [NoteViewMeta!]!   # Frontmatter metadata
  toc: [NoteTocItem!]!     # Table of contents
}
```

### Viewer
Current user context:
```graphql
type Viewer {
  id: ID!
  role: Role!  # GUEST, USER, ADMIN
  user: User
  offers(filter: ViewerOffersFilter!): ViewerOffers
  activePurchases: [Purchase!]!
  lastNoteReadAt(input: LastNoteReadAtInput!): Time
  tgBots: [TgBot!]!
}
```

### Error Handling
All mutations return union types with error handling:
```graphql
union MutationOrErrorPayload = MutationPayload | ErrorPayload

type ErrorPayload {
  message: String!
  byFields: [FieldMessage!]!
}

type FieldMessage {
  name: String!
  value: String!
}
```

## Authentication

**Header-based:**
- `Authorization: Bearer <token>` - User JWT tokens
- `X-Api-Key: <key>` - API key for programmatic access (Obsidian plugin)

**Roles:**
- `GUEST` - Unauthenticated
- `USER` - Authenticated user
- `ADMIN` - Admin privileges

## Rate Limiting & Security

- Email sign-in codes have rate limiting
- Admin mutations require admin role verification
- API key actions are logged
- Audit log for all administrative actions

## Scalars

- `Int64` - 64-bit integers
- `Time` - RFC3339 timestamps
- `Upload` - File upload type for multipart requests
