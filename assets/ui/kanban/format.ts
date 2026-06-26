export type KanbanCard = { text: string; checked: boolean }
export type KanbanList = { title: string; complete: boolean; cards: KanbanCard[] }
/**
 * Only basic single-line `- [ ]`/`- [x]` cards are modeled;
 * multi-line card bodies / sub-content under a card are not preserved yet.
 */
export type KanbanBoard = {
  frontmatter: string // raw bytes from start up to (excluding) the first `## ` heading
  lists: KanbanList[]
  settings: string    // raw bytes from the `%% kanban:settings` run (incl. leading blank lines) to EOF; '' if none
}

const SETTINGS_RE = /\n*^%% kanban:settings[\s\S]*$/m
const COMPLETE_MARKER = '**Complete**'

export function parseBoard(md: string): KanbanBoard {
  md = md.replace(/\r\n/g, '\n')
  let settings = '', rest = md
  const sm = md.match(SETTINGS_RE)
  if (sm && sm.index !== undefined) {
    settings = md.slice(sm.index)
    rest = md.slice(0, sm.index)
  }
  const firstLaneOffset = rest.search(/^## /m)
  const frontmatter = firstLaneOffset === -1 ? rest : rest.slice(0, firstLaneOffset)
  const body = firstLaneOffset === -1 ? '' : rest.slice(firstLaneOffset)
  const lists: KanbanList[] = []
  if (body) {
    for (const chunk of body.split(/(?=^## )/m)) {
      if (!chunk.startsWith('## ')) continue
      const lines = chunk.split('\n')
      const title = lines[0].slice(3).trimEnd()
      let complete = false
      const cards: KanbanCard[] = []
      for (const line of lines.slice(1)) {
        if (line.trim() === COMPLETE_MARKER) { complete = true; continue }
        const m = line.match(/^- \[([ xX])\] ?(.*)$/)
        if (m) cards.push({ checked: m[1].toLowerCase() === 'x', text: m[2] })
      }
      lists.push({ title, complete, cards })
    }
  }
  return { frontmatter, lists, settings }
}

export function serializeBoard(b: KanbanBoard): string {
  const lists = b.lists.map(list => {
    const header = '## ' + list.title
    const completeMarker = list.complete ? COMPLETE_MARKER + '\n' : ''
    const cards = list.cards.map(c => '- [' + (c.checked ? 'x' : ' ') + '] ' + c.text).join('\n')
    return header + '\n\n' + completeMarker + cards
  })
  return b.frontmatter + lists.join('\n\n\n') + b.settings
}
