// REPRODUCTION (failing on purpose) of the two-browser DATA-LOSS bug.
//
// A user driving the board in TWO real browsers (same admin, same board) hit:
//   1. Browser A created a column → never appeared in B (no live propagation).
//   2. A created a card → never reached B either.
//   3. B then created a card → "edit conflict" toast → B's card was DELETED.
//
// The existing Board.livesync.integration.test.tsx mocks subscribeNoteChanges and
// only exercises remote events on DIFFERENT columns, so it never hits the two failures
// the user did:
//   (a) CONFLICT→DROP — a local pending add and a remote add to the SAME column:
//       mergeColumns sees both sides changed that column → null → conflict → reload →
//       the local unsaved card is silently dropped. (PRIMARY required reproduction.)
//   (b) SUPPRESSION — after the board's own save, commitBaseline advances the
//       self-echo baseline to fetchLatestVersionId() (the LATEST stored version, which
//       under concurrent same-board editing is the OTHER browser's save). A genuine
//       remote event at that version then satisfies `versionId <= baseline` and is
//       suppressed — the other browser's change never appears.
//
// Both tests assert the DESIRED lossless behaviour, so they FAIL today (the bug).
// Run via the full suite: `npm test` (or `npm run test:integration`).

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, cleanup, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'

// Capture the subscription handler so tests can push remote events.
const sub = vi.hoisted(() => ({ onChange: null as null | ((c: unknown) => void) }))

// ── mock the network edges of ./api; keep the pure change builders real ──
vi.mock('./api', async (importActual) => {
  const actual = await importActual<typeof import('./api')>()
  return {
    ...actual,
    sha256Base64: vi.fn(async () => 'TEST_HASH'),
    updateNotes: vi.fn(async () => ({ ok: true })),
    fetchNoteContent: vi.fn(async () => ({ error: 'fetchNoteContent not mocked' })),
    fetchLatestVersionId: vi.fn(async () => 1000),
    subscribeNoteChanges: vi.fn((_path: string, onChange: (c: unknown) => void) => {
      sub.onChange = onChange
    }),
  }
})

import Board from './Board'
import { sha256Base64, updateNotes, fetchNoteContent, fetchLatestVersionId } from './api'
import type { NoteChange } from './api'
import { t } from './i18n'

const mockHash = vi.mocked(sha256Base64)
const mockUpdate = vi.mocked(updateNotes)
const mockFetch = vi.mocked(fetchNoteContent)
const mockVersionId = vi.mocked(fetchLatestVersionId)

const PATH = 'boards/demo.md'

// ── (a) board: an EMPTY "Backlog" column that BOTH browsers add a card to ──
const SAMPLE_EMPTY = `---
kanban-plugin: basic
---

## Backlog


## In Progress

- [x] Done task


%% kanban:settings
\`\`\`
{"kanban-plugin":"basic"}
\`\`\`
%%`

// FRESH_X = the server content AFTER browser A added "Card X" to the SAME "Backlog".
const FRESH_X = `---
kanban-plugin: basic
---

## Backlog

- [ ] Card X


## In Progress

- [x] Done task


%% kanban:settings
\`\`\`
{"kanban-plugin":"basic"}
\`\`\`
%%`

// ── (b) board + a remote-added "Remote" column (same shape the live-sync suite uses) ──
const SAMPLE = `---
kanban-plugin: basic
---

## To Do

- [ ] Task 1
- [ ] Task 2


## In Progress

- [x] Done task


%% kanban:settings
\`\`\`
{"kanban-plugin":"basic"}
\`\`\`
%%`

const FRESH = SAMPLE.replace(
  '%% kanban:settings',
  '## Remote\n\n- [ ] Remote card\n\n\n%% kanban:settings',
)

function renderBoard(content: string) {
  ;(window as unknown as { __trip2g_kanban: unknown }).__trip2g_kanban = {
    path: PATH, content, editable: true,
  }
  return render(<Board path={PATH} content={content} editable={true} />)
}

/** Fire a remote event through the captured subscription handler and let it settle. */
async function fireRemote(event: unknown) {
  await act(async () => {
    sub.onChange!(event)
    await new Promise(r => setTimeout(r, 0))
  })
}

function changeAt(callIndex: number): NoteChange {
  const call = mockUpdate.mock.calls[callIndex]
  expect(call, `updateNotes call #${callIndex} should exist`).toBeTruthy()
  return call[0][0]
}

function upsertOf(change: NoteChange): string {
  if (!('upsert' in change)) throw new Error('expected an upsert change, got a patch')
  return change.upsert.content
}

beforeEach(() => {
  vi.clearAllMocks()
  sub.onChange = null
  mockUpdate.mockResolvedValue({ ok: true })
  mockHash.mockResolvedValue('TEST_HASH')
  mockVersionId.mockResolvedValue(1000)
  // The conflict path defers location.reload(1500ms); stub it so the deferred timer
  // (fired during a later test) doesn't log jsdom's "Not implemented: navigation".
  try {
    Object.defineProperty(window.location, 'reload', { configurable: true, value: vi.fn() })
  } catch { /* jsdom may forbid redefining; the deferred reload is harmless console noise */ }
})

afterEach(() => {
  cleanup()
})

describe('Board data loss (two-browser reproduction)', () => {
  test('(a) CONCURRENT SAME-COLUMN add: B adds "Card Y" while a remote add of "Card X" to the SAME column arrives → BOTH must survive', async () => {
    // PRIMARY required reproduction. Browser B has a pending add of "Card Y" to the
    // empty "Backlog"; a remote event says browser A added "Card X" to that SAME column.
    // DESIRED: a column-keyed 3-way merge keeps BOTH cards and one save flushes with
    // both. TODAY: mergeColumns(base=[], local=[Card Y], remote=[Card X]) → both sides
    // changed "Backlog" → null → conflict → reload → B's unsaved "Card Y" is dropped
    // and no merged save is ever sent.
    const user = userEvent.setup()
    renderBoard(SAMPLE_EMPTY)
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // The server already holds A's add of "Card X" to "Backlog".
    mockFetch.mockResolvedValue({ content: FRESH_X, versionId: 999 })

    // B adds "Card Y" to the empty "Backlog" column (debouncing, not yet flushed).
    await user.click(screen.getAllByRole('button', { name: t('en', 'addCard') })[0])
    const ta = await screen.findByPlaceholderText(t('en', 'cardPlaceholder'))
    await user.type(ta, 'Card Y{Enter}')
    expect(screen.getByText('Card Y')).toBeTruthy()

    // A's remote add of "Card X" to "Backlog" arrives mid-edit (versionId above baseline 0).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    // Give the 500ms debounce time to flush in the lossless world.
    await act(async () => { await new Promise(r => setTimeout(r, 700)) })

    // DESIRED (FAILS today): exactly one merged save carrying BOTH cards is sent, and
    // the destructive conflict→reload toast is never shown.
    expect(
      mockUpdate,
      "no merged save was sent — B's unsaved Card Y was discarded by the conflict→reload path",
    ).toHaveBeenCalledTimes(1)
    const saved = upsertOf(changeAt(0))
    expect(saved, 'A\'s "Card X" must survive the merge').toContain('Card X')
    expect(saved, 'B\'s "Card Y" must survive the merge (currently DROPPED)').toContain('Card Y')
    expect(
      screen.queryByText(t('en', 'boardConflictReloading')),
      'the conflict→reload path (data loss) must NOT be taken',
    ).toBeNull()
  })

  test('(b) SELF-ECHO CONTRACT (#46/#47): the self-echo baseline advances to OUR OWN save version, not "latest stored", so a higher peer version is NOT suppressed', async () => {
    // Regression guard for the #46/#47 fix. PRE-fix, commitBaseline advanced
    // baselineVersionIdRef to fetchLatestVersionId() — the LATEST stored version, which
    // under concurrent same-board editing is the OTHER browser's save. A genuine remote
    // event at that version then satisfied `versionId <= baseline` and was suppressed as a
    // self-echo. POST-fix, updateNotes returns OUR OWN save's versionId and commitBaseline
    // advances to THAT, so a strictly-higher peer version is never falsely suppressed.
    //
    // This must exercise the versionId-CARRYING path: with no own versionId, commitBaseline
    // takes the fetchLatestVersionId fallback and DISCARDS the result, so baselineVersionIdRef
    // would stay 0 and the gate would pass trivially — the test would no longer prove the
    // contract. We therefore give B's own save its own versionId (1000) and make the peer
    // version (1005) strictly higher: pre-fix the baseline would have jumped to the latest
    // stored 1005 and suppressed the peer; post-fix it sits at our own 1000 and the peer is
    // adopted.
    const user = userEvent.setup()
    renderBoard(SAMPLE)
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // B's own save returns ITS OWN versionId (1000). The latest stored version is 1005 —
    // actually A's concurrent save, which B never reconciled to; pre-fix the buggy
    // fetchLatestVersionId path would have adopted 1005 as B's self-echo baseline.
    mockUpdate.mockResolvedValue({ ok: true, versionId: 1000 })
    mockVersionId.mockResolvedValue(1005)

    // B toggles Task 1 → debounced save → flush OK → commitBaseline sets baseline to 1000.
    await user.click(screen.getAllByRole('checkbox')[0])
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    await act(async () => { await new Promise(r => setTimeout(r, 25)) })

    // A's GENUINE remote change (a new "Remote" column) arrives at version 1005 (> 1000).
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 1005 })
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 1005 })

    // Post-fix: 1005 > our own baseline of 1000 → NOT suppressed → B re-fetches and adopts
    // A's "Remote" column. (Pre-fix the baseline was 1005, so 1005 <= 1005 suppressed it.)
    expect(
      mockFetch,
      'genuine remote change must NOT be suppressed as a self-echo (versionId > own baseline)',
    ).toHaveBeenCalled()
    await waitFor(() => expect(screen.getByText('Remote')).toBeTruthy(), { timeout: 1000 })
  })
})
