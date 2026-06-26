import React, {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'

import {
  closestCorners,
  DndContext,
  DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  useDroppable,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'

import { parseBoard, serializeBoard, KanbanCard, KanbanList, KanbanBoard } from './format'
import { saveNote } from './api'
import { renderMarkdown } from './markdown'

// ── augmented types (IDs for React / dnd-kit, stripped before serialising) ──

interface AugCard extends KanbanCard { id: string }
interface AugList extends Omit<KanbanList, 'cards'> { id: string; cards: AugCard[] }
interface AugBoard extends Omit<KanbanBoard, 'lists'> { lists: AugList[] }

let _counter = 0
const nextId = () => `c${++_counter}`

function augment(b: KanbanBoard): AugBoard {
  return {
    ...b,
    lists: b.lists.map((l, i) => ({
      ...l,
      id: `col-${i}`,
      cards: l.cards.map(c => ({ ...c, id: nextId() })),
    })),
  }
}

function strip(b: AugBoard): KanbanBoard {
  return {
    frontmatter: b.frontmatter,
    settings: b.settings,
    lists: b.lists.map(({ id: _id, ...l }) => ({
      ...l,
      cards: l.cards.map(({ id: _id2, ...c }) => c),
    })),
  }
}

// ── SortableCard ─────────────────────────────────────────────────────────────

interface CardProps {
  card: AugCard
  listIdx: number
  cardIdx: number
  editable: boolean
  onToggle: () => void
  onEdit: (text: string) => void
  onDelete: () => void
}

function SortableCard({ card, editable, onToggle, onEdit, onDelete }: CardProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(card.text)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: card.id, disabled: editing || !editable })

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  function startEdit() {
    if (!editable) return
    setDraft(card.text)
    setEditing(true)
  }

  function commitEdit() {
    const trimmed = draft.trim()
    if (trimmed && trimmed !== card.text) onEdit(trimmed)
    setEditing(false)
  }

  // Auto-grow textarea
  useEffect(() => {
    if (editing && textareaRef.current) {
      const el = textareaRef.current
      el.style.height = 'auto'
      el.style.height = `${el.scrollHeight}px`
      el.focus()
      el.setSelectionRange(el.value.length, el.value.length)
    }
  }, [editing])

  const cardClass = [
    'kanban-card',
    isDragging ? 'is-dragging' : '',
    !editable ? 'is-read-only' : '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} className={cardClass}>
      <input
        type="checkbox"
        className="kanban-card-check"
        checked={card.checked}
        onChange={onToggle}
        // Don't propagate mousedown so it doesn't accidentally start a drag
        onPointerDown={e => e.stopPropagation()}
        disabled={!editable}
        tabIndex={-1}
      />

      {editing ? (
        <textarea
          ref={textareaRef}
          className="kanban-card-edit"
          value={draft}
          onChange={e => {
            setDraft(e.target.value)
            const el = e.target
            el.style.height = 'auto'
            el.style.height = `${el.scrollHeight}px`
          }}
          onBlur={commitEdit}
          onKeyDown={e => {
            if (e.key === 'Escape') { setEditing(false) }
            // Shift+Enter = newline; plain Enter = commit
            if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); commitEdit() }
          }}
          // Prevent drag from starting while we type
          onPointerDown={e => e.stopPropagation()}
        />
      ) : (
        <span
          className={`kanban-card-text${card.checked ? ' is-checked' : ''}`}
          onDoubleClick={startEdit}
          role={editable ? 'button' : undefined}
          title={editable ? 'Double-click to edit' : undefined}
        >
          {renderMarkdown(card.text)}
        </span>
      )}

      {editable && !editing && (
        <button
          className="kanban-card-delete"
          onClick={e => { e.stopPropagation(); onDelete() }}
          onPointerDown={e => e.stopPropagation()}
          title="Delete card"
          tabIndex={-1}
        >
          ×
        </button>
      )}
    </div>
  )
}

// ── DroppableColumn ───────────────────────────────────────────────────────────

interface ColumnProps {
  list: AugList
  listIdx: number
  editable: boolean
  addingTo: number | null
  newCardText: string
  onNewCardTextChange: (t: string) => void
  onStartAdd: (listIdx: number) => void
  onCommitAdd: (listIdx: number) => void
  onCancelAdd: () => void
  onToggle: (cardIdx: number) => void
  onEdit: (cardIdx: number, text: string) => void
  onDelete: (cardIdx: number) => void
}

function DroppableColumn({
  list,
  listIdx,
  editable,
  addingTo,
  newCardText,
  onNewCardTextChange,
  onStartAdd,
  onCommitAdd,
  onCancelAdd,
  onToggle,
  onEdit,
  onDelete,
}: ColumnProps) {
  const { setNodeRef } = useDroppable({ id: list.id })
  const inputRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (addingTo === listIdx && inputRef.current) {
      inputRef.current.focus()
    }
  }, [addingTo, listIdx])

  const headerClass = `kanban-column-header${list.complete ? ' is-complete' : ''}`

  return (
    <div className="kanban-column">
      <div className={headerClass}>{list.title}</div>

      <SortableContext
        items={list.cards.map(c => c.id)}
        strategy={verticalListSortingStrategy}
      >
        <div ref={setNodeRef} className="kanban-cards">
          {list.cards.map((card, cardIdx) => (
            <SortableCard
              key={card.id}
              card={card}
              listIdx={listIdx}
              cardIdx={cardIdx}
              editable={editable}
              onToggle={() => onToggle(cardIdx)}
              onEdit={text => onEdit(cardIdx, text)}
              onDelete={() => onDelete(cardIdx)}
            />
          ))}
        </div>
      </SortableContext>

      {editable && (
        <div className="kanban-column-footer">
          {addingTo === listIdx ? (
            <textarea
              ref={inputRef}
              className="kanban-add-input"
              value={newCardText}
              placeholder="Card text..."
              onChange={e => onNewCardTextChange(e.target.value)}
              onBlur={() => onCommitAdd(listIdx)}
              onKeyDown={e => {
                if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); onCommitAdd(listIdx) }
                if (e.key === 'Escape') { e.preventDefault(); onCancelAdd() }
              }}
            />
          ) : (
            <button className="kanban-add-btn" onClick={() => onStartAdd(listIdx)}>
              + Add card
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// ── Board ─────────────────────────────────────────────────────────────────────

export interface BoardProps {
  path: string
  content: string
  editable: boolean
}

export default function Board({ path, content, editable }: BoardProps) {
  const [board, setBoard] = useState<AugBoard>(() => augment(parseBoard(content)))
  const [toast, setToast] = useState<string | null>(null)
  const [addingTo, setAddingTo] = useState<number | null>(null)
  const [newCardText, setNewCardText] = useState('')

  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => () => { if (saveTimer.current) clearTimeout(saveTimer.current) }, [])

  const debouncedSave = useCallback(
    (next: AugBoard) => {
      if (saveTimer.current) clearTimeout(saveTimer.current)
      saveTimer.current = setTimeout(() => {
        saveNote(path, serializeBoard(strip(next))).catch(err => {
          const msg = err instanceof Error ? err.message : String(err)
          setToast(msg)
          setTimeout(() => setToast(null), 3500)
        })
      }, 500)
    },
    [path]
  )

  function update(fn: (b: AugBoard) => AugBoard) {
    setBoard(prev => {
      const next = fn(prev)
      debouncedSave(next)
      return next
    })
  }

  // ── drag ──

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over) return
    const activeId = String(active.id)
    const overId = String(over.id)
    if (activeId === overId) return

    update(prev => {
      const srcListIdx = prev.lists.findIndex(l => l.cards.some(c => c.id === activeId))
      if (srcListIdx === -1) return prev
      const srcCardIdx = prev.lists[srcListIdx].cards.findIndex(c => c.id === activeId)

      // Determine destination list and card position
      const colIdx = prev.lists.findIndex(l => l.id === overId)
      let dstListIdx: number, dstCardIdx: number

      if (colIdx !== -1) {
        // Dropped onto an empty column droppable
        dstListIdx = colIdx
        dstCardIdx = prev.lists[colIdx].cards.length
      } else {
        dstListIdx = prev.lists.findIndex(l => l.cards.some(c => c.id === overId))
        if (dstListIdx === -1) return prev
        dstCardIdx = prev.lists[dstListIdx].cards.findIndex(c => c.id === overId)
      }

      const lists = prev.lists.map(l => ({ ...l, cards: [...l.cards] }))

      if (srcListIdx === dstListIdx) {
        // Within the same column: use arrayMove for correct index handling
        lists[srcListIdx] = {
          ...lists[srcListIdx],
          cards: arrayMove(lists[srcListIdx].cards, srcCardIdx, dstCardIdx),
        }
      } else {
        // Cross-column: splice out then splice in
        const [card] = lists[srcListIdx].cards.splice(srcCardIdx, 1)
        lists[dstListIdx].cards.splice(dstCardIdx, 0, card)
      }

      return { ...prev, lists }
    })
  }

  // ── card ops ──

  function handleToggle(listIdx: number, cardIdx: number) {
    update(prev => {
      const lists = prev.lists.map((l, li) =>
        li !== listIdx
          ? l
          : {
              ...l,
              cards: l.cards.map((c, ci) =>
                ci !== cardIdx ? c : { ...c, checked: !c.checked }
              ),
            }
      )
      return { ...prev, lists }
    })
  }

  function handleEdit(listIdx: number, cardIdx: number, text: string) {
    update(prev => {
      const lists = prev.lists.map((l, li) =>
        li !== listIdx
          ? l
          : {
              ...l,
              cards: l.cards.map((c, ci) => (ci !== cardIdx ? c : { ...c, text })),
            }
      )
      return { ...prev, lists }
    })
  }

  function handleDelete(listIdx: number, cardIdx: number) {
    update(prev => {
      const lists = prev.lists.map((l, li) =>
        li !== listIdx ? l : { ...l, cards: l.cards.filter((_, ci) => ci !== cardIdx) }
      )
      return { ...prev, lists }
    })
  }

  // ── add card ──

  function handleCommitAdd(listIdx: number) {
    const text = newCardText.trim()
    setAddingTo(null)
    setNewCardText('')
    if (!text) return
    update(prev => {
      const lists = prev.lists.map((l, li) =>
        li !== listIdx ? l : { ...l, cards: [...l.cards, { id: nextId(), text, checked: false }] }
      )
      return { ...prev, lists }
    })
  }

  function handleCancelAdd() {
    setAddingTo(null)
    setNewCardText('')
  }

  return (
    <>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragEnd={handleDragEnd}
      >
        <div className="kanban-board">
          {board.lists.map((list, listIdx) => (
            <DroppableColumn
              key={list.id}
              list={list}
              listIdx={listIdx}
              editable={editable}
              addingTo={addingTo}
              newCardText={newCardText}
              onNewCardTextChange={setNewCardText}
              onStartAdd={setAddingTo}
              onCommitAdd={handleCommitAdd}
              onCancelAdd={handleCancelAdd}
              onToggle={cardIdx => handleToggle(listIdx, cardIdx)}
              onEdit={(cardIdx, text) => handleEdit(listIdx, cardIdx, text)}
              onDelete={cardIdx => handleDelete(listIdx, cardIdx)}
            />
          ))}
        </div>
      </DndContext>

      {toast && <div className="kanban-toast">{toast}</div>}
    </>
  )
}
