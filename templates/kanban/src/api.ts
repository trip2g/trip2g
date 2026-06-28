const UPDATE_MUTATION = `
  mutation($i: UpdateNotesInput!) {
    updateNotes(input: $i) {
      __typename
      ... on UpdateNotesSuccessPayload { paths updated { path versionId } }
      ... on UpdateNotesHashMismatchPayload { path actualHash }
      ... on UpdateNotesPatchNotFoundPayload { path find }
      ... on ErrorPayload { message }
    }
  }
`

// Re-read the note's current raw markdown from the server. The latest stored
// version's content is what the server hashes for optimistic-concurrency checks,
// so it is the exact baseline to rebase onto after a hashMismatch. Uses the admin
// note-version queries (the board only renders editable for admins).
const NOTE_VERSION_HISTORY_QUERY = `
  query($f: AdminNoteVersionHistoryFilter!) {
    admin { noteVersionHistory(filter: $f) { nodes { versionId } } }
  }
`

const NOTE_VERSION_QUERY = `
  query($id: Int64!) {
    admin { noteVersion(versionId: $id) { content } }
  }
`

export async function sha256Base64(str: string): Promise<string> {
  const encoded = new TextEncoder().encode(str)
  const hashBuf = await crypto.subtle.digest('SHA-256', encoded)
  // Server stores hashes as base64url (+ → -, / → _); padding is kept.
  return btoa(String.fromCharCode(...new Uint8Array(hashBuf)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

export interface PatchInput {
  path: string
  find: string
  replace: string
  expectedHash?: string
}

export interface UpsertInput {
  path: string
  content: string
  expectedHash?: string
}

export type NoteChange = { patch: PatchInput } | { upsert: UpsertInput }

export type UpdateNotesResult =
  | { ok: true; versionId?: number }
  | { hashMismatch: true; path: string; actualHash: string }
  | { patchNotFound: true; path: string; find: string }
  | { error: string }

export function patchChange(
  path: string,
  find: string,
  replace: string,
  expectedHash?: string,
): NoteChange {
  return { patch: { path, find, replace, expectedHash } }
}

export function upsertChange(
  path: string,
  content: string,
  expectedHash?: string,
): NoteChange {
  return { upsert: { path, content, expectedHash } }
}

function withHash(change: NoteChange, hash: string): NoteChange {
  if ('patch' in change) {
    return { patch: { ...change.patch, expectedHash: hash } }
  }
  return { upsert: { ...change.upsert, expectedHash: hash } }
}

type GqlResult = {
  __typename: string
  paths?: string[]
  updated?: { path: string; versionId: number | string }[]
  path?: string
  actualHash?: string
  find?: string
  message?: string
}

export async function updateNotes(
  changes: NoteChange[],
  expectedHash?: string,
): Promise<UpdateNotesResult> {
  const mappedChanges = changes.map(c => {
    const ch = expectedHash ? withHash(c, expectedHash) : c
    if ('patch' in ch) {
      const p = ch.patch
      return {
        patch: {
          path: p.path,
          find: p.find,
          replace: p.replace,
          ...(p.expectedHash !== undefined && { expectedHash: p.expectedHash }),
        },
      }
    }
    const u = ch.upsert
    return {
      upsert: {
        path: u.path,
        content: u.content,
        ...(u.expectedHash !== undefined && { expectedHash: u.expectedHash }),
      },
    }
  })

  let res: Response
  try {
    res = await fetch('/_system/graphql', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: UPDATE_MUTATION,
        variables: { i: { changes: mappedChanges } },
      }),
    })
  } catch (err) {
    return { error: err instanceof Error ? err.message : String(err) }
  }

  if (!res.ok) {
    return { error: `GraphQL request failed: HTTP ${res.status}` }
  }

  const body = await res.json() as {
    data?: { updateNotes?: GqlResult }
    errors?: { message: string }[]
  }

  if (body.errors?.length) {
    return { error: body.errors.map((e: { message: string }) => e.message).join(', ') }
  }

  const result = body.data?.updateNotes
  if (!result) return { error: 'No data returned from updateNotes' }

  switch (result.__typename) {
    case 'UpdateNotesSuccessPayload': {
      // The board saves its own single note; surface that save's new version id so the
      // caller can advance the self-echo baseline to OUR version (not "latest stored",
      // which under concurrent editing is a peer's version — the live-sync data loss).
      const updated = result.updated ?? []
      const versionId = updated.length > 0
        ? Math.max(...updated.map(u => Number(u.versionId)))
        : undefined
      return versionId !== undefined ? { ok: true, versionId } : { ok: true }
    }
    case 'UpdateNotesHashMismatchPayload':
      return { hashMismatch: true, path: result.path!, actualHash: result.actualHash! }
    case 'UpdateNotesPatchNotFoundPayload':
      return { patchNotFound: true, path: result.path!, find: result.find! }
    case 'ErrorPayload':
      return { error: result.message ?? 'Unknown error from updateNotes' }
    default:
      return { error: `Unexpected __typename: ${result.__typename}` }
  }
}

export type FetchNoteContentResult = { content: string; versionId?: number } | { error: string }

/**
 * Re-read the current server content of a note (latest stored version). Used to
 * rebase an in-flight change after a hashMismatch, and to adopt a remote change
 * pushed over the live subscription, instead of reloading and losing the edit.
 * Two round-trips: history (latest versionId) → version content. The versionId is
 * returned alongside the content so the caller can advance its self-echo baseline.
 */
export async function fetchNoteContent(path: string): Promise<FetchNoteContentResult> {
  try {
    const histRes = await fetch('/_system/graphql', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: NOTE_VERSION_HISTORY_QUERY,
        variables: { f: { path, limit: 1 } },
      }),
    })
    if (!histRes.ok) return { error: `GraphQL request failed: HTTP ${histRes.status}` }
    const histBody = await histRes.json() as {
      data?: { admin?: { noteVersionHistory?: { nodes?: { versionId: number }[] } } }
      errors?: { message: string }[]
    }
    if (histBody.errors?.length) return { error: histBody.errors.map(e => e.message).join(', ') }
    const versionId = histBody.data?.admin?.noteVersionHistory?.nodes?.[0]?.versionId
    if (versionId === undefined || versionId === null) return { error: 'note version not found' }

    const verRes = await fetch('/_system/graphql', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: NOTE_VERSION_QUERY,
        variables: { id: versionId },
      }),
    })
    if (!verRes.ok) return { error: `GraphQL request failed: HTTP ${verRes.status}` }
    const verBody = await verRes.json() as {
      data?: { admin?: { noteVersion?: { content?: string } } }
      errors?: { message: string }[]
    }
    if (verBody.errors?.length) return { error: verBody.errors.map(e => e.message).join(', ') }
    const content = verBody.data?.admin?.noteVersion?.content
    if (content === undefined || content === null) return { error: 'note content not found' }
    return { content, versionId: Number(versionId) }
  } catch (err) {
    return { error: err instanceof Error ? err.message : String(err) }
  }
}

/**
 * Fetch just the latest stored versionId for a note (single round-trip). Used to
 * advance the live-subscription baseline after our own save, so the save's own
 * echo event is suppressed (its versionId is <= this baseline). Returns null on
 * any error — the worst case is a spurious "Board updated" re-fetch later.
 */
export async function fetchLatestVersionId(path: string): Promise<number | null> {
  try {
    const res = await fetch('/_system/graphql', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: NOTE_VERSION_HISTORY_QUERY,
        variables: { f: { path, limit: 1 } },
      }),
    })
    if (!res.ok) return null
    const body = await res.json() as {
      data?: { admin?: { noteVersionHistory?: { nodes?: { versionId: number }[] } } }
      errors?: { message: string }[]
    }
    if (body.errors?.length) return null
    const versionId = body.data?.admin?.noteVersionHistory?.nodes?.[0]?.versionId
    return versionId === undefined || versionId === null ? null : Number(versionId)
  } catch {
    return null
  }
}

// ── live change subscription (noteChanges over GraphQL-SSE) ─────────────────────

/** A normalised note-change event delivered to a subscribeNoteChanges listener. */
export type NoteChangeItem =
  | { type: 'upsert'; path: string; pathId: number; versionId: number }
  | { type: 'hide'; path: string }

const NOTE_CHANGES_SUBSCRIPTION = `subscription($filter: NoteChangesFilter!){ noteChanges(filter:$filter){ changes{ __typename ... on NoteUpsertEvent{ path pathId versionId } ... on NoteHideEvent{ path } } } }`

function toChangeItem(ch: {
  __typename?: string
  path?: string
  pathId?: number
  versionId?: number
}): NoteChangeItem | null {
  if (ch.__typename === 'NoteUpsertEvent') {
    return { type: 'upsert', path: ch.path ?? '', pathId: Number(ch.pathId), versionId: Number(ch.versionId) }
  }
  if (ch.__typename === 'NoteHideEvent') {
    return { type: 'hide', path: ch.path ?? '' }
  }
  return null
}

/**
 * Open one GraphQL-over-SSE noteChanges connection and emit each change. POST is
 * required (EventSource is GET-only and cannot carry the query body), so we read
 * the response body stream and parse the SSE frames by hand. Mirrors the editor's
 * transport (assets/ui/sse/sse.ts). Returns when the server sends `complete` or
 * the stream ends; throws on transport errors so the caller can reconnect.
 */
async function connectNoteChanges(
  query: string,
  variables: Record<string, unknown>,
  onChange: (change: NoteChangeItem) => void,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Accept': 'text/event-stream' },
    credentials: 'include',
    body: JSON.stringify({ query, variables }),
    signal,
  })
  if (!res.ok || !res.body) {
    throw new Error(`subscribeNoteChanges: ${res.status} ${res.statusText}`)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let eventType = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const line of lines) {
        if (line.startsWith('event:')) {
          eventType = line.slice(6).trim()
        } else if (line.startsWith('data:')) {
          const payload = line.slice(5).trim()
          if (eventType === 'next') {
            let parsed: { data?: { noteChanges?: { changes?: unknown[] } } } | null = null
            try { parsed = JSON.parse(payload) } catch { parsed = null }
            const changes = parsed?.data?.noteChanges?.changes
            if (Array.isArray(changes)) {
              for (const ch of changes) {
                const item = toChangeItem(ch as Parameters<typeof toChangeItem>[0])
                if (item) onChange(item)
              }
            }
          } else if (eventType === 'complete') {
            return
          }
          eventType = ''
        }
      }
    }
  } finally {
    reader.releaseLock()
  }
}

/** Sleep for `ms`, resolving early if `signal` aborts (so unmount cancels at once). */
function sleepOrAbort(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise<void>(resolve => {
    if (signal?.aborted) { resolve(); return }
    const id = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    function onAbort() { clearTimeout(id); resolve() }
    signal?.addEventListener('abort', onAbort, { once: true })
  })
}

/** Consecutive transport failures before we surface a "disconnected" state once. */
const DISCONNECT_THRESHOLD = 3

/**
 * Subscribe to live changes for `path` over the core noteChanges SSE stream.
 * Auth is the admin cookie (credentials:'include') — no API key needed, since the
 * board only renders editable for admins. Reconnects with a fixed backoff until
 * `signal` aborts (the backoff is raced against the abort so unmount cancels it
 * immediately). Each change is normalised and handed to `onChange`. After
 * `DISCONNECT_THRESHOLD` consecutive transport failures, `onDisconnected` is invoked
 * once so the UI can surface a "live updates disconnected" state.
 */
export async function subscribeNoteChanges(
  path: string,
  onChange: (change: NoteChangeItem) => void,
  signal?: AbortSignal,
  onDisconnected?: () => void,
): Promise<void> {
  const variables = { filter: { includePatterns: [path] } }
  let failures = 0
  let notified = false
  while (!signal?.aborted) {
    try {
      await connectNoteChanges(NOTE_CHANGES_SUBSCRIPTION, variables, onChange, signal)
      failures = 0  // a clean completion (server `complete` / stream end) is not a failure
    } catch {
      if (signal?.aborted) return
      failures++
      if (failures >= DISCONNECT_THRESHOLD && !notified) {
        notified = true
        onDisconnected?.()
      }
    }
    if (signal?.aborted) return
    await sleepOrAbort(3000, signal)
  }
}
