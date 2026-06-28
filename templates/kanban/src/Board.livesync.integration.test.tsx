// Integration tests for the LIVE-SYNC wiring (jsdom + @testing-library/react).
//
// These render the real <Board> with a mocked ./api module, capture the onChange
// handler passed to subscribeNoteChanges, and fire remote noteChanges events to
// assert the clean-vs-dirty apply logic:
//   • dirty  — a local edit is in flight → the edit is NOT lost and the queued save
//     rebases onto the remote baseline (converges).
//   • clean  — no local edit → the board adopts the remote content wholesale.
//
// Run via the full suite: `npm test` (or `npm run test:integration`).

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, cleanup, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'

// Capture the subscription handler so tests can push remote events. vi.hoisted lets
// the (hoisted) vi.mock factory share state with the test body.
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
import { sha256Base64, updateNotes, fetchNoteContent } from './api'
import type { NoteChange } from './api'

const mockHash = vi.mocked(sha256Base64)
const mockUpdate = vi.mocked(updateNotes)
const mockFetch = vi.mocked(fetchNoteContent)

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

// FRESH = SAMPLE plus a column added by a *remote* editor.
const FRESH = SAMPLE.replace(
  '%% kanban:settings',
  '## Remote\n\n- [ ] Remote card\n\n\n%% kanban:settings',
)

const PATH = 'boards/demo.md'

function renderBoard() {
  ;(window as unknown as { __trip2g_kanban: unknown }).__trip2g_kanban = {
    path: PATH, content: SAMPLE, editable: true,
  }
  return render(<Board path={PATH} content={SAMPLE} editable={true} />)
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

beforeEach(() => {
  vi.clearAllMocks()
  sub.onChange = null
  mockUpdate.mockResolvedValue({ ok: true })
  mockHash.mockResolvedValue('TEST_HASH')
})

afterEach(() => {
  cleanup()
})

describe('Board live sync', () => {
  test('dirty: a remote bump while a local edit is pending does not drop the edit and converges', async () => {
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // A remote editor added a column; the re-fetch returns that fresh content.
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })

    // Start a local edit that is still debouncing (toggle Task 1 → checked).
    const checkbox = screen.getAllByRole('checkbox')[0]
    await user.click(checkbox)

    // Remote bump arrives mid-edit (higher versionId than our baseline of 0).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    // Dirty path must NOT replace the board: the local toggle is kept and the remote
    // column is not shown until the next clean reconcile.
    expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(true)
    expect(screen.queryByText('Remote')).toBeNull()

    // The debounced save flushes, carrying the local toggle, rebased onto FRESH.
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    const change = changeAt(0)
    if (!('patch' in change)) throw new Error('expected a surgical patch, got an upsert (edit/remote clobbered)')
    expect(change.patch.find).toBe('- [ ] Task 1')
    expect(change.patch.replace).toBe('- [x] Task 1')
    // Converged: the change was hashed against the FRESH remote baseline, so it will
    // not clobber the remote column.
    expect(mockHash).toHaveBeenCalledWith(FRESH)
  })

  test('clean: a remote bump with no local edit adopts the remote board', async () => {
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })

    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    // Board adopted the remote content (the new column is now visible)…
    await waitFor(() => expect(screen.getByText('Remote')).toBeTruthy())
    // …and a subtle toast announces it.
    expect(screen.getByText('Board updated')).toBeTruthy()
    // Adopt is a local reconcile, not a save.
    expect(mockUpdate).not.toHaveBeenCalled()
  })
})
