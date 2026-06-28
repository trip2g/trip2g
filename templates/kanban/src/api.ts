const UPDATE_MUTATION = `
  mutation($i: UpdateNotesInput!) {
    updateNotes(input: $i) {
      __typename
      ... on UpdateNotesSuccessPayload { paths }
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
  | { ok: true }
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
    case 'UpdateNotesSuccessPayload':
      return { ok: true }
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

export type FetchNoteContentResult = { content: string } | { error: string }

/**
 * Re-read the current server content of a note (latest stored version). Used to
 * rebase an in-flight change after a hashMismatch instead of reloading and losing
 * the edit. Two round-trips: history (latest versionId) → version content.
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
    return { content }
  } catch (err) {
    return { error: err instanceof Error ? err.message : String(err) }
  }
}
