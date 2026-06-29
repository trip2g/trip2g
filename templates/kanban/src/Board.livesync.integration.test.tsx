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
import { sha256Base64, updateNotes, fetchNoteContent, fetchLatestVersionId } from './api'
import type { NoteChange } from './api'

const mockHash = vi.mocked(sha256Base64)
const mockUpdate = vi.mocked(updateNotes)
const mockFetch = vi.mocked(fetchNoteContent)
const mockVersionId = vi.mocked(fetchLatestVersionId)

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

function upsertOf(change: NoteChange): string {
  if (!('upsert' in change)) throw new Error('expected an upsert change, got a patch')
  return change.upsert.content
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

    // Dirty path keeps the local toggle AND adopts the remote column, so the visible
    // board stays in sync with the baseline (otherwise a later eager op clobbers it).
    expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(true)
    expect(screen.getByText('Remote')).toBeTruthy()

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

  test('saveTimer reset: after a completed save a remote bump is still adopted (board not wedged dirty)', async () => {
    // Regression for the saveTimer-leak CRITICAL: the debounce ref was never nulled,
    // so after the FIRST save the board looked permanently "dirty" and remote changes
    // were silently swallowed (clean-adopt path went dead).
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // A local toggle that debounces and saves successfully.
    const checkbox = screen.getAllByRole('checkbox')[0]
    await user.click(checkbox)
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    // Let the flush (and its commitBaseline round-trip) fully settle → board is CLEAN.
    await act(async () => { await new Promise(r => setTimeout(r, 25)) })

    // A remote editor adds a column (versionId above our just-saved baseline of 1000).
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 1001 })
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 1001 })

    // Board adopts the remote board — the path the leak killed.
    await waitFor(() => expect(screen.getByText('Remote')).toBeTruthy())
    expect(screen.getByText('Board updated')).toBeTruthy()
  })

  test('dirty STRUCTURAL rebase: local column rename + remote column add → both survive', async () => {
    // FIX 2: the structural rebase used to re-emit only local columns, dropping any
    // column the remote added/renamed/moved. A column-keyed 3-way merge keeps both.
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Remote editor added a "Remote" column; the re-fetch returns that fresh content.
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })

    // Local structural edit: rename "To Do" → "Backlog" (debouncing, not yet flushed).
    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')

    // Remote bump arrives mid-edit.
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    // The debounced structural save flushes carrying BOTH the local rename and the
    // remote column (3-way merge), hashed against the FRESH remote baseline.
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    const change = changeAt(0)
    if (!('upsert' in change)) throw new Error('expected a merged upsert (structural rebase)')
    expect(change.upsert.content).toContain('## Backlog')    // local rename survived
    expect(change.upsert.content).toContain('## Remote')      // remote column survived
    expect(change.upsert.content).not.toContain('## To Do')   // renamed away
    expect(mockHash).toHaveBeenCalledWith(FRESH)              // merged onto the remote baseline
  })

  test('TOCTOU: an edit fired during the re-fetch await is not lost and does not clobber remote', async () => {
    // FIX 3: dirtiness was sampled BEFORE the re-fetch await; an edit landing during
    // the fetch was missed → wiped, and its pending change clobbered the remote.
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Make the re-fetch hang so we can inject an edit DURING the await.
    let resolveFetch!: (v: { content: string; versionId: number }) => void
    mockFetch.mockImplementation(() => new Promise(res => { resolveFetch = res }))

    // Remote bump arrives; the handler starts and suspends on fetchNoteContent.
    await act(async () => {
      sub.onChange!({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })
      await Promise.resolve()
    })

    // EDIT during the await: toggle Task 1 (sets pendingRef + the debounce timer).
    const checkbox = screen.getAllByRole('checkbox')[0]
    await user.click(checkbox)
    expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(true)

    // The re-fetch now resolves with the remote column added.
    await act(async () => {
      resolveFetch({ content: FRESH, versionId: 999 })
      await new Promise(r => setTimeout(r, 0))
    })

    // Dirty was re-sampled AFTER the await → the edit is NOT wiped (toggle kept) …
    expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(true)
    // … and the remote column is adopted alongside the kept edit (board in sync).
    expect(screen.getByText('Remote')).toBeTruthy()

    // The debounced save flushes the toggle, hashed onto the FRESH remote baseline
    // (no clobber of the remote column).
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    const change = changeAt(0)
    if (!('patch' in change)) throw new Error('expected a surgical patch (edit kept)')
    expect(change.patch.find).toBe('- [ ] Task 1')
    expect(change.patch.replace).toBe('- [x] Task 1')
    expect(mockHash).toHaveBeenCalledWith(FRESH)
  })

  test('NoteHideEvent for this board disables editing and toasts', async () => {
    // FIX 5b: a hide of the current path must disable editing (not just toast).
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())
    expect(screen.getByText('Editable')).toBeTruthy()

    await fireRemote({ type: 'hide', path: PATH })

    expect(screen.getByText('Board deleted')).toBeTruthy()
    // Editing affordances are gone.
    expect(screen.queryByText('Editable')).toBeNull()
    expect(screen.queryByRole('button', { name: '+ Add list' })).toBeNull()
  })

  test('dirty-adopt then a later eager RENAME preserves the remote-added column (no silent clobber)', async () => {
    // Surviving HIGH: after a dirty-adopt the board must adopt the remote column, else
    // a CLEAN eager structural op re-serialises a stale board and silently drops it
    // (its hash matches the server, so the merge never runs).
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Remote adds "Remote"; the dirty re-fetch returns it.
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })

    // Step 1: local structural rename To Do → Backlog (debouncing).
    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')

    // Step 2: remote bump mid-edit → dirty-adopt (board adopts Remote + keeps rename).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    // Step 3: the debounced merged save flushes; let commitBaseline settle.
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    expect(upsertOf(changeAt(0))).toContain('## Remote')
    await act(async () => { await new Promise(r => setTimeout(r, 25)) })

    // The board itself adopted the remote column (stale-UI fixed).
    expect(screen.getByText('Remote')).toBeTruthy()
    expect(screen.getByText('Backlog')).toBeTruthy()

    // Step 4: a SECOND eager structural op (rename In Progress) — NO further remote
    // event. The pre-fix bug serialised a board missing Remote → dropped it silently.
    await user.dblClick(screen.getByText('In Progress'))
    const input2 = screen.getByDisplayValue('In Progress')
    await user.clear(input2)
    await user.type(input2, 'Doing{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(2), { timeout: 2000 })
    const final = upsertOf(changeAt(1))
    expect(final).toContain('## Remote')    // remote column STILL survives
    expect(final).toContain('## Backlog')   // first rename survives
    expect(final).toContain('## Doing')     // second rename applied
  })

  test('dirty-adopt then a later eager DELETE of a different column preserves the remote-added column', async () => {
    const user = userEvent.setup()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })

    // Local rename To Do → Backlog (debouncing) then remote adds Remote → dirty-adopt.
    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    await act(async () => { await new Promise(r => setTimeout(r, 25)) })
    expect(screen.getByText('Remote')).toBeTruthy()

    // Eager DELETE of a DIFFERENT column (In Progress, index 1) — Remote must survive.
    const delButtons = screen.getAllByLabelText('Delete column')
    await user.click(delButtons[1])

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(2), { timeout: 2000 })
    const final = upsertOf(changeAt(1))
    expect(final).toContain('## Remote')          // remote column survives the delete
    expect(final).toContain('## Backlog')
    expect(final).not.toContain('## In Progress') // the deleted column is gone
  })

  test('in-flight-flush window: a remote add during the network round-trip is adopted (no later clobber)', async () => {
    // HIGH residual: dirty can be true via inFlightRef alone (pendingRef nulled at
    // flushSave start), so handleRemoteChange's `if (pending)` adopt is skipped during
    // the network window. The adopt must therefore also run in flushSave's hashMismatch
    // retry success, or a later eager structural op clobbers the remote column.
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Both the dirty re-fetch and flushSave's hashMismatch re-read return the remote board.
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })

    // First POST hangs (in flight) so a remote event can land during its window; it then
    // resolves to a (spurious-looking) hashMismatch. Later POSTs succeed.
    let resolveFirstUpdate!: (v: Awaited<ReturnType<typeof updateNotes>>) => void
    mockUpdate
      .mockImplementationOnce(() => new Promise(res => { resolveFirstUpdate = res }))
      .mockResolvedValue({ ok: true })

    // Step 1: rename To Do → Backlog (structural). Debounce → flushSave starts the POST.
    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    // Step 2: remote adds Remote DURING the in-flight POST (dirty via inFlightRef, no
    // pending → handleRemoteChange's own adopt is skipped; board still lacks Remote).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })
    expect(screen.queryByText('Remote')).toBeNull()

    // Step 3: the POST returns a hashMismatch → flushSave rebases, retries, commits the
    // merged board, and now adopts it. The board gains Remote (invariant restored).
    await act(async () => {
      resolveFirstUpdate({ hashMismatch: true, path: PATH, actualHash: 'SERVER' })
      await new Promise(r => setTimeout(r, 0))
    })
    await waitFor(() => expect(screen.getByText('Remote')).toBeTruthy(), { timeout: 2000 })

    // Step 4: a later CLEAN eager structural op (rename In Progress) — the pre-fix bug
    // serialised a board missing Remote and silently dropped it.
    await user.dblClick(screen.getByText('In Progress'))
    const input2 = screen.getByDisplayValue('In Progress')
    await user.clear(input2)
    await user.type(input2, 'Doing{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(3), { timeout: 2000 })
    const final = upsertOf(changeAt(2))
    expect(final).toContain('## Remote')   // remote column STILL survives
    expect(final).toContain('## Backlog')
    expect(final).toContain('## Doing')
  })

  test('id-stable adopt: a remote column-add does not remount unchanged cards (open editor survives)', async () => {
    // MEDIUM: augment minted fresh ids on every adopt → every card/column remounted →
    // an open card editor lost its uncommitted draft + focus. augmentStable reuses ids
    // for value-identical items so unchanged subtrees (and the editor) don't remount.
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Open the editor on Task 1 and type an uncommitted draft (no save scheduled).
    await user.dblClick(screen.getByText('Task 1'))
    const ta = screen.getByDisplayValue('Task 1')
    await user.clear(ta)
    await user.type(ta, 'Task 1 DRAFT')
    expect(screen.getByDisplayValue('Task 1 DRAFT')).toBeTruthy()

    // A clean remote adopt that only ADDS a column — Task 1 itself is unchanged.
    mockFetch.mockResolvedValue({ content: FRESH, versionId: 999 })
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 999 })

    // The remote column is adopted …
    expect(screen.getByText('Remote')).toBeTruthy()
    // … and the unchanged Task 1 card did NOT remount: its open editor + uncommitted
    // draft survived (its id was reused, so React kept the component instance).
    expect(screen.getByDisplayValue('Task 1 DRAFT')).toBeTruthy()
  })

  test('OK-path race: a remote add during the commitBaseline await is healed by flushSave finally', async () => {
    // Narrow residual: a flush can exit via the OK path, whose commitBaseline awaits
    // fetchLatestVersionId. A remote event during THAT await advances baselineMdRef with
    // no adopt on any per-exit path. The finally heal reconciles the board regardless of
    // exit path, so a later eager structural op can't clobber the remote column.
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Server state after Alice's rename lands AND Bob adds Remote: [Backlog, In Progress, Remote].
    const FRESH2 = SAMPLE
      .replace('## To Do', '## Backlog')
      .replace('%% kanban:settings', '## Remote\n\n- [ ] Remote card\n\n\n%% kanban:settings')
    mockFetch.mockResolvedValue({ content: FRESH2, versionId: 1001 })

    // Alice's POST succeeds (OK path), but make commitBaseline's fetchLatestVersionId hang
    // so Bob's event lands DURING the commitBaseline await (inFlightRef still true).
    let resolveVid!: (v: number | null) => void
    mockVersionId.mockImplementationOnce(() => new Promise(res => { resolveVid = res }))

    // Step 1: rename To Do → Backlog → flush → POST OK → commitBaseline awaits the vid.
    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    await act(async () => { await new Promise(r => setTimeout(r, 0)) })

    // Step 2: Bob adds Remote DURING the commitBaseline await (dirty via inFlightRef,
    // pending null → handleRemoteChange's adopt skipped; baselineMdRef → FRESH2).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 1001 })
    expect(screen.queryByText('Remote')).toBeNull()

    // Step 3: the vid resolves → flushSave returns via OK → finally adopts FRESH2.
    await act(async () => {
      resolveVid(1001)
      await new Promise(r => setTimeout(r, 0))
    })
    await waitFor(() => expect(screen.getByText('Remote')).toBeTruthy(), { timeout: 2000 })

    // Step 4: a later eager rename — Remote must survive (pre-fix: board lacked R → clobber).
    await user.dblClick(screen.getByText('In Progress'))
    const input2 = screen.getByDisplayValue('In Progress')
    await user.clear(input2)
    await user.type(input2, 'Doing{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(2), { timeout: 2000 })
    const final = upsertOf(changeAt(1))
    expect(final).toContain('## Remote')   // remote column STILL survives
    expect(final).toContain('## Backlog')
    expect(final).toContain('## Doing')
  })

  test('no flicker: a 2nd edit queued during a save does not blink off, while a concurrent remote column heals in', async () => {
    // Cosmetic regression of the finally heal: it adopted RAW baselineMdRef, which lacks
    // a still-pending 2nd edit, so that edit reverted in the UI for ~500ms. The heal now
    // applies pendingRef on top, so the 2nd edit never blinks while the remote column is
    // still pulled in.
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Server after Alice's Task 1 toggle commits AND Bob adds Remote (Task 2 still ✗).
    const FRESH3 = SAMPLE
      .replace('- [ ] Task 1', '- [x] Task 1')
      .replace('%% kanban:settings', '## Remote\n\n- [ ] Remote card\n\n\n%% kanban:settings')
    mockFetch.mockResolvedValue({ content: FRESH3, versionId: 1001 })

    // Hold commitBaseline's fetchLatestVersionId open so the 2nd toggle + remote land
    // during the 1st save's in-flight window.
    let resolveVid!: (v: number | null) => void
    mockVersionId.mockImplementationOnce(() => new Promise(res => { resolveVid = res }))

    // Step 1: toggle Task 1 → flush POST OK → commitBaseline awaits the held vid.
    await user.click(screen.getAllByRole('checkbox')[0])
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })
    await act(async () => { await new Promise(r => setTimeout(r, 0)) })

    // Step 2: toggle Task 2 DURING the in-flight window → pendingRef set, Task 2 now ✓.
    await user.click(screen.getAllByRole('checkbox')[1])
    expect((screen.getAllByRole('checkbox')[1] as HTMLInputElement).checked).toBe(true)

    // Step 3: a remote column lands during the same window (dirty via inFlightRef+pending).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 1001 })

    // Step 4: release the 1st save → its finally heals. The heal applies the still-pending
    // Task 2 toggle, so it must NOT blink off, and the remote column is pulled in.
    await act(async () => {
      resolveVid(1001)
      await new Promise(r => setTimeout(r, 0))
    })
    // Task 2 stayed checked across the heal (no flicker) — pre-fix it reverted to ✗ here …
    expect((screen.getAllByRole('checkbox')[1] as HTMLInputElement).checked).toBe(true)
    // … and the concurrently-added remote column was healed in.
    expect(screen.getByText('Remote')).toBeTruthy()
  })

  test('OK-path clobber: a remote add during the SAVE round-trip (post-#47 versionId) still renders live', async () => {
    // Residual #2 surviving #47: under the post-#47 contract updateNotes returns OUR own
    // versionId, so commitBaseline takes the synchronous branch (no fetchLatestVersionId
    // await). The only await left is the SAVE POST itself. A peer event landing during
    // THAT window passes the version gate (we haven't committed), so handleRemoteChange
    // sets baselineMdRef = peer content but SKIPS its adopt (pendingRef is null). The OK
    // branch then rebuilds baselineMdRef from the PRE-await snapshot, clobbering the peer
    // content; the finally heal reads the already-clobbered ref → no-op. Net: the peer
    // change never renders live (a refresh re-fetches it — the user-reported symptom).
    const user = userEvent.setup()
    renderBoard()
    await waitFor(() => expect(sub.onChange).not.toBeNull())

    // Server after Alice's rename lands AND Bob adds Remote: [Backlog, In Progress, Remote].
    const FRESH2 = SAMPLE
      .replace('## To Do', '## Backlog')
      .replace('%% kanban:settings', '## Remote\n\n- [ ] Remote card\n\n\n%% kanban:settings')
    mockFetch.mockResolvedValue({ content: FRESH2, versionId: 1001 })

    // First POST hangs (in flight) so Bob's event lands during the save round-trip; it then
    // resolves OK carrying OUR OWN versionId (post-#47 contract). Later POSTs also succeed.
    let resolveFirstUpdate!: (v: Awaited<ReturnType<typeof updateNotes>>) => void
    mockUpdate
      .mockImplementationOnce(() => new Promise(res => { resolveFirstUpdate = res }))
      .mockResolvedValue({ ok: true, versionId: 1002 })

    // Step 1: rename To Do → Backlog (structural). Debounce → flushSave starts the POST.
    await user.dblClick(screen.getByText('To Do'))
    const input = screen.getByDisplayValue('To Do')
    await user.clear(input)
    await user.type(input, 'Backlog{Enter}')
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1), { timeout: 2000 })

    // Step 2: Bob adds Remote DURING the in-flight POST (dirty via inFlightRef, pendingRef
    // null → handleRemoteChange's adopt is skipped; baselineMdRef → FRESH2, board lacks it).
    await fireRemote({ type: 'upsert', path: PATH, pathId: 1, versionId: 1001 })
    expect(screen.queryByText('Remote')).toBeNull()

    // Step 3: the POST resolves OK with our own versionId → OK branch. Pre-fix: it rebuilds
    // baselineMdRef from the stale pre-await snapshot, clobbering FRESH2, and the finally
    // heal no-ops. The peer column must render live (post-fix: the OK branch respects the
    // remote-advanced baseline, the finally heal adopts it).
    await act(async () => {
      resolveFirstUpdate({ ok: true, versionId: 1000 })
      await new Promise(r => setTimeout(r, 0))
    })
    await waitFor(() => expect(screen.getByText('Remote')).toBeTruthy(), { timeout: 2000 })

    // Step 4: a later eager rename — Remote must survive (pre-fix: board lacked R → clobber).
    await user.dblClick(screen.getByText('In Progress'))
    const input2 = screen.getByDisplayValue('In Progress')
    await user.clear(input2)
    await user.type(input2, 'Doing{Enter}')

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(2), { timeout: 2000 })
    const final = upsertOf(changeAt(1))
    expect(final).toContain('## Remote')   // remote column STILL survives
    expect(final).toContain('## Backlog')
    expect(final).toContain('## Doing')
  })
})
