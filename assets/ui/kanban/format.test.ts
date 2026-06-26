import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { parseBoard, serializeBoard } from './format.js'

const FIXTURE = [
  '---', '', 'kanban-plugin: basic', '', '---', '',
  '## Todo', '', '- [ ] Create project specification', '- [ ] Research implementation approach', '', '',
  '## In Progress', '', '- [ ] Set up development environment', '', '',
  '## Done', '', '**Complete**', '- [x] Initialize repository', '', '', '',
  '%% kanban:settings', '```', '{"kanban-plugin":"basic","lane-width":270}', '```', '%%',
].join('\n')

describe('parseBoard', () => {
  it('parses lane titles', () => {
    const board = parseBoard(FIXTURE)
    assert.deepEqual(board.lists.map(l => l.title), ['Todo', 'In Progress', 'Done'])
  })

  it('parses Todo cards', () => {
    const board = parseBoard(FIXTURE)
    assert.deepEqual(board.lists[0].cards, [
      { text: 'Create project specification', checked: false },
      { text: 'Research implementation approach', checked: false },
    ])
  })

  it('marks Done list as complete', () => {
    const board = parseBoard(FIXTURE)
    assert.equal(board.lists[2].complete, true)
  })

  it('parses Done cards', () => {
    const board = parseBoard(FIXTURE)
    assert.deepEqual(board.lists[2].cards, [
      { text: 'Initialize repository', checked: true },
    ])
  })

  it('frontmatter includes kanban-plugin', () => {
    const board = parseBoard(FIXTURE)
    assert.ok(board.frontmatter.includes('kanban-plugin: basic'))
  })

  it('settings includes marker and lane-width', () => {
    const board = parseBoard(FIXTURE)
    assert.ok(board.settings.includes('%% kanban:settings'))
    assert.ok(board.settings.includes('lane-width'))
  })

  it('non-kanban doc has empty lists', () => {
    const board = parseBoard('# Just a note\n\nText.')
    assert.deepEqual(board.lists, [])
  })

  it('CRLF line endings parse same cards as LF', () => {
    const lf = parseBoard(FIXTURE)
    const crlf = parseBoard(FIXTURE.replace(/\n/g, '\r\n'))
    assert.deepEqual(crlf.lists.map(l => l.title), lf.lists.map(l => l.title))
    assert.deepEqual(crlf.lists[0].cards, lf.lists[0].cards)
    assert.deepEqual(crlf.lists[2].cards, lf.lists[2].cards)
  })

  it('[X] parses as checked; empty card body parses', () => {
    const board = parseBoard('## Misc\n\n- [X] done\n- [ ] ')
    assert.deepEqual(board.lists[0].cards, [
      { text: 'done', checked: true },
      { text: '', checked: false },
    ])
  })
})

describe('serializeBoard', () => {
  it('byte-identity round-trip', () => {
    const result = serializeBoard(parseBoard(FIXTURE))
    if (result !== FIXTURE) {
      // Show a character-level diff to aid debugging
      const ra = [...result], fa = [...FIXTURE]
      const len = Math.max(ra.length, fa.length)
      for (let i = 0; i < len; i++) {
        if (ra[i] !== fa[i]) {
          const ctx = 20
          console.error(`First diff at char ${i}:`)
          console.error('got:      ', JSON.stringify(result.slice(Math.max(0, i - ctx), i + ctx)))
          console.error('expected: ', JSON.stringify(FIXTURE.slice(Math.max(0, i - ctx), i + ctx)))
          break
        }
      }
    }
    assert.equal(result, FIXTURE)
  })

  it('model stability after card move', () => {
    const b = parseBoard(FIXTURE)
    // move Todo[0] → In Progress
    const moved = b.lists[0].cards.shift()!
    b.lists[1].cards.push(moved)

    const b2 = parseBoard(serializeBoard(b))
    assert.deepEqual(b2.lists[0].cards.map(c => c.text), ['Research implementation approach'])
    assert.deepEqual(b2.lists[1].cards.map(c => c.text), ['Set up development environment', 'Create project specification'])
  })
})
