const MUTATION = `
  mutation($i: PushNotesInput!) {
    pushNotes(input: $i) {
      __typename
      ... on ErrorPayload { message }
    }
  }
`

/** Push updated note content to trip2g via the GraphQL API. Throws on error. */
export async function saveNote(path: string, content: string): Promise<void> {
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query: MUTATION,
      variables: { i: { updates: [{ path, content }] } },
    }),
  })

  if (!res.ok) {
    throw new Error(`GraphQL request failed: HTTP ${res.status}`)
  }

  const body = await res.json() as { data?: { pushNotes?: { __typename: string; message?: string } }; errors?: { message: string }[] }

  if (body.errors?.length) {
    throw new Error(body.errors.map(e => e.message).join(', '))
  }

  const result = body.data?.pushNotes
  if (result?.__typename === 'ErrorPayload') {
    throw new Error(result.message ?? 'Unknown error from pushNotes')
  }
}
