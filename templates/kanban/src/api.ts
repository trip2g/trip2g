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

export async function sha256Base64(str: string): Promise<string> {
  const encoded = new TextEncoder().encode(str)
  const hashBuf = await crypto.subtle.digest('SHA-256', encoded)
  return btoa(String.fromCharCode(...new Uint8Array(hashBuf)))
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
