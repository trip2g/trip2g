import type { KanbanBoard, KanbanCard, KanbanList } from './format'
import { parseBoard, serializeBoard } from './format'

export interface MoveCardArgs {
  /** source list index */
  from: number
  /** card index within the source list */
  card: number
  /** destination list index */
  to: number
  /** insert-before index in destination list, computed after the removal */
  at: number
}

/** Move a card between (or within) lists. Never mutates the input board. */
export function moveCard(b: KanbanBoard, { from, card, to, at }: MoveCardArgs): KanbanBoard {
  const lists = b.lists.map(l => ({ ...l, cards: [...l.cards] }))
  const [removed] = lists[from].cards.splice(card, 1)
  lists[to].cards.splice(at, 0, removed)
  return { ...b, lists }
}

/** Append a new card to the given list. */
export function addCard(b: KanbanBoard, listIndex: number, text: string): KanbanBoard {
  const lists = b.lists.map((l, i) =>
    i === listIndex ? { ...l, cards: [...l.cards, { text, checked: false }] } : l
  )
  return { ...b, lists }
}

/** Replace fields on a card. */
export function editCard(
  b: KanbanBoard,
  listIndex: number,
  cardIndex: number,
  patch: Partial<KanbanCard>
): KanbanBoard {
  const lists = b.lists.map((l, i) =>
    i === listIndex
      ? { ...l, cards: l.cards.map((c, j) => (j === cardIndex ? { ...c, ...patch } : c)) }
      : l
  )
  return { ...b, lists }
}

/** Remove a card from a list. */
export function deleteCard(b: KanbanBoard, listIndex: number, cardIndex: number): KanbanBoard {
  const lists = b.lists.map((l, i) =>
    i === listIndex ? { ...l, cards: l.cards.filter((_, j) => j !== cardIndex) } : l
  )
  return { ...b, lists }
}

/** Toggle the checked state of a card. */
export function toggleCard(b: KanbanBoard, listIndex: number, cardIndex: number): KanbanBoard {
  return editCard(b, listIndex, cardIndex, {
    checked: !b.lists[listIndex].cards[cardIndex].checked,
  })
}

// ── list (column) reducers ────────────────────────────────────────────────

/** Append a new empty list (column). */
export function addList(b: KanbanBoard, title: string): KanbanBoard {
  return { ...b, lists: [...b.lists, { title, complete: false, cards: [] }] }
}

/** Rename a list. */
export function renameList(b: KanbanBoard, listIndex: number, title: string): KanbanBoard {
  const lists = b.lists.map((l, i) => (i === listIndex ? { ...l, title } : l))
  return { ...b, lists }
}

/** Remove a list and all its cards. */
export function deleteList(b: KanbanBoard, listIndex: number): KanbanBoard {
  return { ...b, lists: b.lists.filter((_, i) => i !== listIndex) }
}

/** Reorder lists: move the list at `from` to `to`. */
export function moveList(b: KanbanBoard, from: number, to: number): KanbanBoard {
  const lists = b.lists.map(l => ({ ...l, cards: [...l.cards] }))
  const [removed] = lists.splice(from, 1)
  lists.splice(to, 0, removed)
  return { ...b, lists }
}

/** The exact markdown line for a card: `- [x] text` or `- [ ] text`. */
export function cardLine(card: { checked: boolean; text: string }): string {
  return `- [${card.checked ? 'x' : ' '}] ${card.text}`
}

const CARD_RE = /^- \[([ xX])\] /

/**
 * Apply `lists` (current board state) surgically to `baselineMd`.
 * Only card lines (`- [ ] …` / `- [x] …`) in each column section are replaced.
 * All other content — frontmatter, blank lines, headings, sub-bullets, blockquotes,
 * the settings block, anything format.ts does not model — is preserved verbatim.
 */
export function applyBoardToBaseline(baselineMd: string, lists: KanbanList[]): string {
  // Split off settings block first (same regex as parseBoard)
  const SETTINGS_RE = /\n*^%% kanban:settings[\s\S]*$/m
  const sm = baselineMd.match(SETTINGS_RE)
  let body = baselineMd
  let settingsSuffix = ''
  if (sm && sm.index !== undefined) {
    settingsSuffix = baselineMd.slice(sm.index)
    body = baselineMd.slice(0, sm.index)
  }

  // Find where column sections start
  const firstLaneOffset = body.search(/^## /m)
  if (firstLaneOffset === -1) return baselineMd  // no columns — nothing to do
  const frontmatter = body.slice(0, firstLaneOffset)
  const bodyFromFirstLane = body.slice(firstLaneOffset)

  // Split into per-column chunks (each starts with ## Heading)
  const chunks = bodyFromFirstLane.split(/(?=^## )/m)
  const resultChunks: string[] = [frontmatter]

  for (const chunk of chunks) {
    if (!chunk.startsWith('## ')) {
      resultChunks.push(chunk)
      continue
    }

    const headingLine = chunk.split('\n')[0]
    const title = headingLine.slice(3).trimEnd()
    const listIdx = lists.findIndex(l => l.title === title)

    if (listIdx === -1) {
      // Unknown column — keep verbatim
      resultChunks.push(chunk)
      continue
    }

    const newCards = lists[listIdx].cards.map(c => cardLine(c))
    const chunkLines = chunk.split('\n')
    const outputLines: string[] = []
    let cardsInserted = false

    for (const line of chunkLines) {
      if (CARD_RE.test(line)) {
        if (!cardsInserted) {
          outputLines.push(...newCards)
          cardsInserted = true
        }
        // Skip old card line (already replaced above)
        continue
      }
      outputLines.push(line)
    }

    // Column had no card lines in baseline but now has cards (e.g. move into empty col)
    if (!cardsInserted && newCards.length > 0) {
      // Insert after the heading line + one trailing blank (canonical gap after ## heading)
      let insertIdx = 1
      if (outputLines[insertIdx] === '') insertIdx++
      outputLines.splice(insertIdx, 0, ...newCards)
      // Remove one trailing blank to keep inter-column whitespace consistent
      const lastIdx = outputLines.length - 1
      if (outputLines[lastIdx] === '') outputLines.splice(lastIdx, 1)
    }

    resultChunks.push(outputLines.join('\n'))
  }

  return resultChunks.join('') + settingsSuffix
}

/**
 * Re-serialize the columns region for a structural change (add/rename/delete/move
 * a column) and splice it into `baselineMd`. The surgical per-card patch
 * (`applyBoardToBaseline`) matches columns by heading and only rewrites card
 * lines, so it cannot express heading-structure changes; structural ops re-emit
 * the whole columns region instead.
 *
 * The frontmatter (everything before the first `## ` heading) and the trailing
 * `%% kanban:settings %%` block are taken verbatim from `baselineMd` — so prior
 * card-op patches, frontmatter, and the settings block all survive byte-for-byte.
 *
 * Caveat: an obsidian-kanban `***`-separated archive is not modeled by parseBoard
 * (its bare cards are absorbed into the last column and the `***` separator is
 * dropped), so the separator line is not re-emitted here — the same limitation as
 * a plain `serializeBoard`. Frontmatter and settings are always preserved.
 */
export function applyStructuralChange(baselineMd: string, lists: KanbanList[]): string {
  const base = parseBoard(baselineMd)
  return serializeBoard({ frontmatter: base.frontmatter, lists, settings: base.settings })
}
