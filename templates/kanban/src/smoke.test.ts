import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parseBoard, serializeBoard } from './format'
import { moveCard, addCard, editCard, deleteCard, toggleCard } from './ops'

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
