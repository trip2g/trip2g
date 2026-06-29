import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { sha256Base64 } from './api'
import { parseBoard, serializeBoard } from './format'
import { moveCard, addCard, editCard, deleteCard, toggleCard, cardLine, applyBoardToBaseline, addList, renameList, deleteList, moveList, applyStructuralChange, mergeColumns, mergeCards, mergeChangedColumn } from './ops'
import type { KanbanCard, KanbanList } from './format'

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

// ── sha256Base64: must produce base64url (+ → -, / → _) ─────────────────────

// Helper: reference base64url via Node's built-in crypto (server-side encoding).
// Node's digest('base64url') strips padding; the server keeps it.
// Mirror the same translation the fixed sha256Base64 applies so padding is preserved.
function refHash(input: string): string {
  return createHash('sha256').update(input).digest('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

// These inputs have `+` or `/` in their standard base64 SHA-256 digest, so they
// would fail against the old btoa-only implementation.
const BASE64URL_INPUTS = [
  '1',      // standard: a4ayc/80/OGda4BO/1o/V0etpOqiLx1JwB5S3beHW0s=  (has /)
  'test',   // standard: n4bQgYhMfWWaL+qgxVrQFaO/TxsrC4Is0V1sFbDwCgg=  (has + and /)
  'hello',  // standard: LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ=  (has +)
  'a longer example string that exercises the full hash',
]

for (const input of BASE64URL_INPUTS) {
  test(`sha256Base64 matches node base64url for ${JSON.stringify(input)}`, async () => {
    const result = await sha256Base64(input)
    assert.equal(result, refHash(input), `base64url mismatch for ${JSON.stringify(input)}`)
    assert.ok(!result.includes('+'), `result must not contain + for ${JSON.stringify(input)}`)
    assert.ok(!result.includes('/'), `result must not contain / for ${JSON.stringify(input)}`)
  })
}

test('round-trip: serializeBoard(parseBoard(md)) === md', () => {
  assert.equal(serializeBoard(parseBoard(SAMPLE)), SAMPLE)
})

test('parseBoard extracts correct structure', () => {
  const b = parseBoard(SAMPLE)
  assert.equal(b.lists.length, 2)
  assert.equal(b.lists[0].title, 'To Do')
  assert.equal(b.lists[0].cards.length, 2)
  assert.equal(b.lists[0].cards[0].text, 'Task 1')
  assert.equal(b.lists[0].cards[0].checked, false)
  assert.equal(b.lists[1].cards[0].checked, true)
  assert.equal(b.lists[1].cards[0].text, 'Done task')
})

test('moveCard: cross-list move', () => {
  const b = parseBoard(SAMPLE)
  const moved = moveCard(b, { from: 0, card: 0, to: 1, at: 1 })
  assert.equal(moved.lists[0].cards.length, 1)
  assert.equal(moved.lists[0].cards[0].text, 'Task 2')
  assert.equal(moved.lists[1].cards.length, 2)
  assert.equal(moved.lists[1].cards[1].text, 'Task 1')
})

test('moveCard: same-list reorder', () => {
  const b = parseBoard(SAMPLE)
  // Move Task 1 (index 0) after Task 2 (index 1): at=1 means insert before index 1 in post-removal list
  const moved = moveCard(b, { from: 0, card: 0, to: 0, at: 1 })
  assert.equal(moved.lists[0].cards[0].text, 'Task 2')
  assert.equal(moved.lists[0].cards[1].text, 'Task 1')
})

test('moveCard: does not mutate the original board', () => {
  const b = parseBoard(SAMPLE)
  const moved = moveCard(b, { from: 0, card: 0, to: 1, at: 0 })
  assert.equal(b.lists[0].cards.length, 2, 'original must be unchanged')
  assert.equal(moved.lists[0].cards.length, 1)
})

test('addCard appends a card to the list', () => {
  const b = parseBoard(SAMPLE)
  const next = addCard(b, 0, 'New card')
  assert.equal(next.lists[0].cards.length, 3)
  assert.equal(next.lists[0].cards[2].text, 'New card')
  assert.equal(next.lists[0].cards[2].checked, false)
})

test('editCard updates card text', () => {
  const b = parseBoard(SAMPLE)
  const next = editCard(b, 0, 0, { text: 'Updated' })
  assert.equal(next.lists[0].cards[0].text, 'Updated')
  // other cards untouched
  assert.equal(next.lists[0].cards[1].text, 'Task 2')
})

test('deleteCard removes the card', () => {
  const b = parseBoard(SAMPLE)
  const next = deleteCard(b, 0, 0)
  assert.equal(next.lists[0].cards.length, 1)
  assert.equal(next.lists[0].cards[0].text, 'Task 2')
})

test('toggleCard flips checked', () => {
  const b = parseBoard(SAMPLE)
  const next = toggleCard(b, 0, 0)
  assert.equal(next.lists[0].cards[0].checked, true)
  const back = toggleCard(next, 0, 0)
  assert.equal(back.lists[0].cards[0].checked, false)
})

test('frontmatter and settings are preserved through all ops', () => {
  const b = parseBoard(SAMPLE)
  const ops = [
    addCard(b, 0, 'X'),
    moveCard(b, { from: 0, card: 0, to: 1, at: 0 }),
    toggleCard(b, 0, 0),
    deleteCard(b, 0, 0),
  ]
  for (const result of ops) {
    assert.equal(result.frontmatter, b.frontmatter, 'frontmatter preserved')
    assert.equal(result.settings, b.settings, 'settings preserved')
  }
})

test('serialize after moveCard produces valid parseable markdown', () => {
  const b = parseBoard(SAMPLE)
  const moved = moveCard(b, { from: 0, card: 0, to: 1, at: 0 })
  const md = serializeBoard(moved)
  const reparsed = parseBoard(md)
  assert.equal(reparsed.lists[0].cards.length, 1)
  assert.equal(reparsed.lists[1].cards.length, 2)
  assert.equal(reparsed.lists[1].cards[0].text, 'Task 1')
})

// ── Surgical edit tests (unmodeled content preservation) ─────────────────────

// A baseline that contains lines format.ts does not model:
// a blockquote and a sub-bullet interspersed among cards.
const BASELINE_WITH_UNMODELED = `---
kanban-plugin: basic
---

## To Do

- [ ] Task 1
> a blockquote that the parser ignores
- [ ] Task 2

## In Progress

- [x] Done task
  - sub-bullet under Done task


%% kanban:settings
\`\`\`
{"kanban-plugin":"basic"}
\`\`\`
%%`

test('patch toggle preserves unmodeled blockquote', () => {
  // The surgical patch for a toggle is: find = oldLine, replace = newLine
  const card = { checked: false, text: 'Task 1' }
  const oldLine = cardLine(card)
  const newLine = cardLine({ ...card, checked: true })

  assert.equal(oldLine, '- [ ] Task 1')
  assert.equal(newLine, '- [x] Task 1')

  // Applying the find/replace to baseline must preserve the blockquote
  const result = BASELINE_WITH_UNMODELED.replace(oldLine, newLine)
  assert.ok(result.includes('> a blockquote that the parser ignores'), 'blockquote preserved after toggle patch')
  assert.ok(result.includes('- [x] Task 1'), 'card is now checked')
  assert.ok(result.includes('- [ ] Task 2'), 'other card untouched')
  // Make sure we only replaced the right line
  assert.ok(!result.includes('- [ ] Task 1'), 'old unchecked Task 1 gone')
})

test('applyBoardToBaseline (upsert-surgical move) preserves unmodeled content', () => {
  // Move Task 1 from To Do to In Progress (at index 0)
  const lists = [
    { title: 'To Do', complete: false, cards: [{ text: 'Task 2', checked: false }] },
    {
      title: 'In Progress',
      complete: false,
      cards: [
        { text: 'Task 1', checked: false },
        { text: 'Done task', checked: true },
      ],
    },
  ]

  const newMd = applyBoardToBaseline(BASELINE_WITH_UNMODELED, lists)

  // Unmodeled lines must survive
  assert.ok(newMd.includes('> a blockquote that the parser ignores'), 'blockquote preserved after move upsert')
  assert.ok(newMd.includes('  - sub-bullet under Done task'), 'sub-bullet preserved after move upsert')
  assert.ok(newMd.includes('%% kanban:settings'), 'settings block preserved')

  // Card arrangement must reflect the move
  const todoSection = newMd.split('## In Progress')[0]
  const progressSection = newMd.split('## In Progress')[1]
  assert.ok(!todoSection.includes('- [ ] Task 1'), 'Task 1 no longer in To Do')
  assert.ok(todoSection.includes('- [ ] Task 2'), 'Task 2 still in To Do')
  assert.ok(progressSection.includes('- [ ] Task 1'), 'Task 1 now in In Progress')
  assert.ok(progressSection.includes('- [x] Done task'), 'Done task still in In Progress')
})

test('applyBoardToBaseline move to empty column preserves content', () => {
  const baselineWithEmptyCol = `---
kanban-plugin: basic
---

## Backlog

- [ ] Ticket A
> important note

## Done


`

  const lists = [
    { title: 'Backlog', complete: false, cards: [] },
    { title: 'Done', complete: false, cards: [{ text: 'Ticket A', checked: false }] },
  ]

  const newMd = applyBoardToBaseline(baselineWithEmptyCol, lists)

  assert.ok(newMd.includes('> important note'), 'unmodeled line preserved when moving out of col')
  assert.ok(!newMd.split('## Done')[0].includes('- [ ] Ticket A'), 'Ticket A no longer in Backlog section')
  assert.ok(newMd.split('## Done')[1].includes('- [ ] Ticket A'), 'Ticket A appears in Done section')
})

// ── List (column) reducers: round-trip through serialize/parse ───────────────

test('addList adds a ## heading that round-trips', () => {
  const b = parseBoard(SAMPLE)
  const next = addList(b, 'Review')
  assert.equal(next.lists.length, 3)
  assert.equal(next.lists[2].title, 'Review')
  assert.equal(next.lists[2].cards.length, 0)
  const round = parseBoard(serializeBoard(next))
  assert.deepEqual(round.lists.map(l => l.title), ['To Do', 'In Progress', 'Review'])
  assert.equal(round.lists[2].cards.length, 0)
})

test('renameList changes the ## heading and round-trips (cards travel)', () => {
  const b = parseBoard(SAMPLE)
  const next = renameList(b, 0, 'Backlog')
  assert.equal(next.lists[0].title, 'Backlog')
  const round = parseBoard(serializeBoard(next))
  assert.deepEqual(round.lists.map(l => l.title), ['Backlog', 'In Progress'])
  assert.deepEqual(round.lists[0].cards.map(c => c.text), ['Task 1', 'Task 2'])
})

test('deleteList removes the ## heading and round-trips', () => {
  const b = parseBoard(SAMPLE)
  const next = deleteList(b, 0)
  assert.equal(next.lists.length, 1)
  const round = parseBoard(serializeBoard(next))
  assert.deepEqual(round.lists.map(l => l.title), ['In Progress'])
})

test('moveList reorders the ## headings and round-trips (cards travel)', () => {
  const b = parseBoard(SAMPLE)
  const next = moveList(b, 0, 1)
  assert.deepEqual(next.lists.map(l => l.title), ['In Progress', 'To Do'])
  const round = parseBoard(serializeBoard(next))
  assert.deepEqual(round.lists.map(l => l.title), ['In Progress', 'To Do'])
  assert.deepEqual(round.lists[1].cards.map(c => c.text), ['Task 1', 'Task 2'])
})

test('list reducers do not mutate the original board', () => {
  const b = parseBoard(SAMPLE)
  addList(b, 'X')
  renameList(b, 0, 'Y')
  deleteList(b, 0)
  moveList(b, 0, 1)
  assert.equal(b.lists.length, 2, 'original must be unchanged')
  assert.equal(b.lists[0].title, 'To Do')
  assert.equal(b.lists[1].title, 'In Progress')
})

test('frontmatter and settings survive list reducers', () => {
  const b = parseBoard(SAMPLE)
  for (const result of [addList(b, 'X'), renameList(b, 0, 'Y'), deleteList(b, 1), moveList(b, 0, 1)]) {
    assert.equal(result.frontmatter, b.frontmatter, 'frontmatter preserved')
    assert.equal(result.settings, b.settings, 'settings preserved')
  }
})

test('applyStructuralChange splices columns while preserving frontmatter + settings', () => {
  const b = parseBoard(SAMPLE)
  const next = addList(b, 'Review')
  const md = applyStructuralChange(SAMPLE, next.lists)
  assert.ok(md.startsWith('---\nkanban-plugin: basic\n---'), 'frontmatter preserved')
  assert.ok(md.includes('## Review'), 'new column heading spliced in')
  assert.ok(md.includes('%% kanban:settings'), 'settings block preserved')
  const round = parseBoard(md)
  assert.deepEqual(round.lists.map(l => l.title), ['To Do', 'In Progress', 'Review'])
})

// ── mergeColumns: column-keyed 3-way merge (live-sync rebase) ─────────────────

const col = (title: string, cards: string[] = [], complete = false): KanbanList => ({
  title,
  complete,
  cards: cards.map(text => ({ text, checked: false })),
})
const titles = (lists: KanbanList[] | null) => (lists ? lists.map(l => l.title) : null)

test('mergeColumns: local rename + remote column add → both survive', () => {
  const base = [col('To Do', ['Task 1']), col('In Progress', ['Done'])]
  const local = [col('Backlog', ['Task 1']), col('In Progress', ['Done'])]       // renamed To Do→Backlog
  const remote = [col('To Do', ['Task 1']), col('In Progress', ['Done']), col('Remote', ['R'])] // added Remote
  const merged = mergeColumns(base, local, remote)
  assert.deepEqual(titles(merged), ['Backlog', 'In Progress', 'Remote'])
})

test('mergeColumns: only-local card edit kept; only-remote column kept', () => {
  const base = [col('A', ['x']), col('B', ['y'])]
  const local = [col('A', ['x EDITED']), col('B', ['y'])]              // edited A's card
  const remote = [col('A', ['x']), col('B', ['y']), col('C', ['z'])]   // added C
  const merged = mergeColumns(base, local, remote)
  assert.deepEqual(titles(merged), ['A', 'B', 'C'])
  assert.equal(merged![0].cards[0].text, 'x EDITED', 'local card edit preserved')
  assert.equal(merged![2].cards[0].text, 'z', 'remote-added column preserved')
})

test('mergeColumns: only-remote card add to an untouched column is kept', () => {
  const base = [col('A', ['x']), col('B', ['y'])]
  const local = [col('A', ['x']), col('B', ['y']), col('New', [])]     // local added empty column
  const remote = [col('A', ['x', 'x2']), col('B', ['y'])]             // remote added a card to A
  const merged = mergeColumns(base, local, remote)
  assert.deepEqual(titles(merged), ['A', 'B', 'New'])
  assert.deepEqual(merged![0].cards.map(c => c.text), ['x', 'x2'], 'remote card-add to A preserved')
})

test('mergeColumns: both edit the same column differently → conflict (null)', () => {
  const base = [col('A', ['x'])]
  const local = [col('A', ['x LOCAL'])]
  const remote = [col('A', ['x REMOTE'])]
  assert.equal(mergeColumns(base, local, remote), null)
})

test('mergeColumns: local delete honoured when remote untouched', () => {
  const base = [col('A', ['x']), col('B', ['y'])]
  const local = [col('A', ['x'])]                 // deleted B
  const remote = [col('A', ['x']), col('B', ['y'])]
  assert.deepEqual(titles(mergeColumns(base, local, remote)), ['A'])
})

test('mergeColumns: local delete vs remote edit of same column → conflict (null)', () => {
  const base = [col('A', ['x']), col('B', ['y'])]
  const local = [col('A', ['x'])]                 // deleted B
  const remote = [col('A', ['x']), col('B', ['y', 'y2'])] // remote edited B
  assert.equal(mergeColumns(base, local, remote), null)
})

// ── mergeCards / mergeChangedColumn: direct positional card-level 3-way merge ──
// These exercise the card merge in isolation — the exact cases the PR #46 reviewer
// traced by hand. Every assertion pins the full resulting card list, so any silent
// drop or duplicate fails the test.

const cards = (texts: string[]): KanbanCard[] => texts.map(text => ({ text, checked: false }))
const cardTexts = (cs: KanbanCard[] | null) => (cs ? cs.map(c => c.text) : null)

test('mergeCards: local moves a card within the column, remote untouched → move kept', () => {
  // base a,b,c → local moves a to the end
  const merged = mergeCards(cards(['a', 'b', 'c']), cards(['b', 'c', 'a']), cards(['a', 'b', 'c']))
  assert.deepEqual(cardTexts(merged), ['b', 'c', 'a'])
})

test('mergeCards: local deletes a card, remote untouched → delete kept (no resurrection)', () => {
  const merged = mergeCards(cards(['a', 'b', 'c']), cards(['a', 'c']), cards(['a', 'b', 'c']))
  assert.deepEqual(cardTexts(merged), ['a', 'c'])
})

test('mergeCards: local reorders (swap), remote untouched → reorder kept', () => {
  const merged = mergeCards(cards(['a', 'b']), cards(['b', 'a']), cards(['a', 'b']))
  assert.deepEqual(cardTexts(merged), ['b', 'a'])
})

test('mergeCards: duplicate card text preserved when a local add keeps both originals', () => {
  const merged = mergeCards(cards(['dup', 'dup']), cards(['dup', 'dup', 'new']), cards(['dup', 'dup']))
  assert.deepEqual(cardTexts(merged), ['dup', 'dup', 'new'], 'both duplicates and the new card survive')
})

test('mergeCards: deleting one of two identical cards leaves exactly one', () => {
  const merged = mergeCards(cards(['dup', 'dup']), cards(['dup']), cards(['dup', 'dup']))
  assert.deepEqual(cardTexts(merged), ['dup'], 'exactly one duplicate remains — no over-delete, no resurrection')
})

test('mergeCards: both sides add distinct cards into the same region → both survive', () => {
  const merged = mergeCards(cards(['c']), cards(['c', 'A']), cards(['c', 'B']))
  assert.deepEqual(cardTexts(merged), ['c', 'A', 'B'])
})

test('mergeCards: both sides add the same-text card → deduped to one', () => {
  const merged = mergeCards(cards(['c']), cards(['c', 'A']), cards(['c', 'A']))
  assert.deepEqual(cardTexts(merged), ['c', 'A'], 'identical add on both sides is not duplicated')
})

test('mergeCards: both add into an empty column → both kept, local-first order', () => {
  const merged = mergeCards(cards([]), cards(['X']), cards(['Y']))
  assert.deepEqual(cardTexts(merged), ['X', 'Y'])
})

test('mergeCards: position-shift conflict (remote delete shifts across a local edit) → null', () => {
  // local edits b→B; remote deletes a, shifting every index. The positional
  // prefix/suffix reduction cannot isolate the two changes → genuine conflict.
  const merged = mergeCards(cards(['a', 'b', 'c']), cards(['a', 'B', 'c']), cards(['b', 'c']))
  assert.equal(merged, null)
})

test('mergeCards: same base card edited differently on each side → null', () => {
  const merged = mergeCards(cards(['x']), cards(['x LOCAL']), cards(['x REMOTE']))
  assert.equal(merged, null)
})

test('mergeChangedColumn: local toggles complete, remote adds a card → both survive', () => {
  const base: KanbanList = { title: 'A', complete: false, cards: cards(['a']) }
  const local: KanbanList = { title: 'A', complete: true, cards: cards(['a']) }
  const remote: KanbanList = { title: 'A', complete: false, cards: cards(['a', 'b']) }
  const merged = mergeChangedColumn(base, local, remote)
  assert.ok(merged)
  assert.equal(merged!.complete, true, 'local complete-toggle preserved')
  assert.deepEqual(cardTexts(merged!.cards), ['a', 'b'], 'remote card-add preserved')
})

test('mergeChangedColumn: conflicting card edits on each side → null', () => {
  const base: KanbanList = { title: 'A', complete: false, cards: cards(['x']) }
  const local: KanbanList = { title: 'A', complete: false, cards: cards(['x LOCAL']) }
  const remote: KanbanList = { title: 'A', complete: false, cards: cards(['x REMOTE']) }
  assert.equal(mergeChangedColumn(base, local, remote), null)
})
