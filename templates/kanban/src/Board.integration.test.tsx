// Integration tests for the Board save WIRING (jsdom + @testing-library/react).
//
// These render the real <Board> with a mocked ./api module and assert that the
// right NoteChange reaches updateNotes for each column-management action — the gap
// that let the "Add list didn't persist" bug through. Pure-logic (ops/format) is
// covered by smoke.test.ts; here we exercise the React side effects end to end.
//
// Run: npm run test:integration   (or the full suite via `npm test`).

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'

import type { NoteChange } from './api'

// ── mock the network/crypto edges of ./api; keep the pure change builders real ──
vi.mock('./api', async (importActual) => {
  const actual = await importActual<typeof import('./api')>()
  return {
    ...actual,
    sha256Base64: vi.fn(async () => 'TEST_HASH'),
    updateNotes: vi.fn(async () => ({ ok: true })),
    fetchNoteContent: vi.fn(async () => ({ error: 'fetchNoteContent not mocked' })),
    // Live-sync edges are stubbed so the save-wiring tests stay hermetic.
    fetchLatestVersionId: vi.fn(async () => null),
    subscribeNoteChanges: vi.fn(),
  }
})

import Board from './Board'
import { updateNotes, fetchNoteContent } from './api'

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

const PATH = 'boards/demo.md'

function renderBoard() {
  // The real app reads window.__trip2g_kanban; Board takes the same shape as props.
  ;(window as unknown as { __trip2g_kanban: unknown }).__trip2g_kanban = {
    path: PATH,
    content: SAMPLE,
    editable: true,
  }
  return render(<Board path={PATH} content={SAMPLE} editable={true} />)
}

/** The single change handed to the Nth updateNotes call (changes are sent one at a time). */
function changeAt(callIndex: number): NoteChange {
  const call = mockUpdate.mock.calls[callIndex]
  expect(call, `updateNotes call #${callIndex} should exist`).toBeTruthy()
  return call[0][0]
}

function upsertContent(change: NoteChange): string {
  if (!('upsert' in change)) throw new Error('expected an upsert change, got a patch')
  return change.upsert.content
}

beforeEach(() => {
  vi.clearAllMocks()
  mockUpdate.mockResolvedValue({ ok: true })
})

afterEach(() => {
  cleanup()
})

describe('Board save wiring', () => {
  test('does not save on mount / initial render', async () => {
    renderBoard()
    // Render committed; no user action → nothing should be scheduled or sent.
    expect(mockUpdate).not.toHaveBeenCalled()
    // Wait past the debounce window to be sure no deferred save fires either.
    await new Promise(r => setTimeout(r, 600))
    expect(mockUpdate).not.toHaveBeenCalled()
  })

  test('Add list → exactly one upsert containing the new column + the existing ones', async () => {
    const user = userEvent.setup()
    renderBoard()

    await user.click(screen.getByRole('button', { name: '+ Add list' }))
    const input = screen.getByPlaceholderText('Column title...')
    await user.type(input, 'Review{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    const content = upsertContent(changeAt(0))
    expect(content).toContain('## Review')
    expect(content).toContain('## To Do')
    expect(content).toContain('## In Progress')
    // The new column must appear exactly once (no double-commit from Enter + blur).
    expect(content.match(/^## Review$/gm)?.length).toBe(1)
    // Pre-existing cards + frontmatter survive.
    expect(content).toContain('- [ ] Task 1')
    expect(content).toContain('kanban-plugin: basic')

    // No further save sneaks in after the first.
    await new Promise(r => setTimeout(r, 200))
    expect(mockUpdate).toHaveBeenCalledTimes(1)
  })

  test('Rename column → upsert with the renamed heading, others intact', async () => {
    const user = userEvent.setup()
    renderBoard()

    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    const content = upsertContent(changeAt(0))
    expect(content).toContain('## Backlog')
    expect(content).toContain('## In Progress')
    expect(content).not.toContain('## To Do')
    // Cards travel with the renamed column.
    expect(content).toContain('- [ ] Task 1')
    expect(content).toContain('- [x] Done task')
  })

  test('Delete column → upsert without that heading', async () => {
    const user = userEvent.setup()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderBoard()

    // First column ("To Do") delete button.
    const deleteButtons = screen.getAllByLabelText('Delete column')
    await user.click(deleteButtons[0])

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    const content = upsertContent(changeAt(0))
    expect(content).not.toContain('## To Do')
    expect(content).not.toContain('- [ ] Task 1')
    expect(content).toContain('## In Progress')
    expect(content).toContain('- [x] Done task')
  })

  test('Card toggle → a patch flipping the checkbox of the right line', async () => {
    const user = userEvent.setup()
    renderBoard()

    // First checkbox is "Task 1" (unchecked) in To Do.
    const checkbox = screen.getAllByRole('checkbox')[0]
    await user.click(checkbox)

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    const change = changeAt(0)
    if (!('patch' in change)) throw new Error('expected a patch change, got an upsert')
    expect(change.patch.find).toBe('- [ ] Task 1')
    expect(change.patch.replace).toBe('- [x] Task 1')
  })

  test('hashMismatch → re-reads server content, rebases, retries once (no edit lost)', async () => {
    const user = userEvent.setup()
    renderBoard()

    // First send hits a (spurious) hashMismatch; the rebased retry then succeeds.
    mockUpdate
      .mockResolvedValueOnce({ hashMismatch: true, path: PATH, actualHash: 'SERVER_HASH' })
      .mockResolvedValueOnce({ ok: true })

    // Fresh server content differs only in frontmatter (e.g. a sync writeback).
    const FRESH = SAMPLE.replace('kanban-plugin: basic', 'kanban-plugin: basic\nsynced: true')
    mockFetch.mockResolvedValue({ content: FRESH })

    await user.click(screen.getByRole('button', { name: '+ Add list' }))
    await user.type(screen.getByPlaceholderText('Column title...'), 'Review{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(2), { timeout: 2000 })
    expect(mockFetch).toHaveBeenCalledTimes(1)

    // The retry is rebased onto the FRESH baseline (keeps its frontmatter) and still
    // carries our in-flight "Add list" edit.
    const retried = upsertContent(changeAt(1))
    expect(retried).toContain('synced: true')
    expect(retried).toContain('## Review')
    expect(retried).toContain('## To Do')
  })
})

describe('Board not-a-board state', () => {
  test('renders a friendly explainer when a layout:kanban note has no columns and no marker', () => {
    const NOT_A_BOARD = `---\nlayout: kanban\n---\n\nJust a note with no columns.\n`
    render(<Board path={PATH} content={NOT_A_BOARD} editable={true} />)
    expect(screen.getByText('Not a Kanban board')).toBeTruthy()
    // No board chrome (no add-list button).
    expect(screen.queryByRole('button', { name: '+ Add list' })).toBeNull()
  })

  test('a legitimately-empty board WITH the kanban-plugin marker still renders as a board', () => {
    const EMPTY_BOARD = `---\nkanban-plugin: basic\nlayout: kanban\n---\n\n%% kanban:settings\n\`\`\`\n{"kanban-plugin":"basic"}\n\`\`\`\n%%`
    render(<Board path={PATH} content={EMPTY_BOARD} editable={true} />)
    // Not the friendly state …
    expect(screen.queryByText('Not a Kanban board')).toBeNull()
    // … the editable board chrome renders instead.
    expect(screen.getByRole('button', { name: '+ Add list' })).toBeTruthy()
  })
})
