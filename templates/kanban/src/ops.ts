import type { KanbanBoard, KanbanCard } from './format'

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
