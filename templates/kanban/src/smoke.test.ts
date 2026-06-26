import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parseBoard, serializeBoard } from './format'
import { moveCard, addCard, editCard, deleteCard, toggleCard, cardLine, applyBoardToBaseline } from './ops'

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
