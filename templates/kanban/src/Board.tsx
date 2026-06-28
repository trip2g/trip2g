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
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  arrayMove,
  horizontalListSortingStrategy,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'

import { parseBoard, KanbanCard, KanbanList, KanbanBoard } from './format'
import { cardLine, applyBoardToBaseline, applyStructuralChange } from './ops'
import {
  sha256Base64,
  updateNotes,
  fetchNoteContent,
  patchChange,
  upsertChange,
  NoteChange,
  UpdateNotesResult,
} from './api'
import { renderMarkdown } from './markdown'

// ── augmented types (IDs for React / dnd-kit, stripped before serialising) ──

interface AugCard extends KanbanCard { id: string }
interface AugList extends Omit<KanbanList, 'cards'> { id: string; cards: AugCard[] }
interface AugBoard extends Omit<KanbanBoard, 'lists'> { lists: AugList[] }

let _counter = 0
const nextId = () => `c${++_counter}`

// Lists get their own stable id namespace so columns stay distinct from card ids
// and survive add/delete/reorder without dnd-kit/React key churn.
let _listCounter = 0
const nextListId = () => `col-${++_listCounter}`

function augment(b: KanbanBoard): AugBoard {
  return {
    ...b,
    lists: b.lists.map(l => ({
      ...l,
      id: nextListId(),
      cards: l.cards.map(c => ({ ...c, id: nextId() })),
    })),
  }
}

function stripLists(lists: AugList[]): KanbanList[] {
  return lists.map(({ id: _id, ...l }) => ({
    ...l,
    cards: l.cards.map(({ id: _id2, ...c }) => c),
  }))
}

// ── save plumbing (pure, no component state) ───────────────────────────────────

/** How a pending change should be rebased if the server baseline shifts under it. */
type RebaseKind = 'structural' | 'surgical'

/** A queued save: the change to send plus how to re-derive it against a fresh baseline. */
interface PendingSave {
  change: NoteChange
  rebase: (freshBaseline: string) => NoteChange
}

/** Attach the expected hash (of the baseline the change is built on) and send it. */
async function sendChange(change: NoteChange, baselineForHash: string): Promise<UpdateNotesResult> {
  const hash = await sha256Base64(baselineForHash)
  const withHash: NoteChange =
    'patch' in change
      ? { patch: { ...change.patch, expectedHash: hash } }
      : { upsert: { ...change.upsert, expectedHash: hash } }
  return updateNotes([withHash])
}

/** The markdown that results from applying `change` to `base` (the new local baseline). */
function applyChangeToBaseline(change: NoteChange, base: string): string {
  return 'patch' in change
    ? base.replace(change.patch.find, change.patch.replace)
    : change.upsert.content
}

/** Derive a display title from a file path. */
function titleFromPath(path: string): string {
  const segment = path.split('/').pop() ?? path
  const name = segment.replace(/\.[^.]+$/, '')
  return name
    .replace(/[-_]/g, ' ')
    .split(' ')
    .map(w => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

// ── GridIcon ──────────────────────────────────────────────────────────────────

function GridIcon() {
  return (
    <div className="kanban-header-icon" aria-hidden="true">
      {Array.from({ length: 9 }).map((_, i) => (
        <span key={i} />
      ))}
    </div>
  )
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
          aria-label="Delete card"
        >
          ×
        </button>
      )}
    </div>
  )
}

// ── SortableColumn ────────────────────────────────────────────────────────────

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
  onRenameList: (title: string) => void
  onDeleteList: () => void
}

function SortableColumn({
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
  onRenameList,
  onDeleteList,
}: ColumnProps) {
  const [renaming, setRenaming] = useState(false)
  const [titleDraft, setTitleDraft] = useState(list.title)
  const renameRef = useRef<HTMLInputElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  // Guards against the rename input's Enter-keydown and its blur-on-unmount both
  // committing the same rename.
  const renamingRef = useRef(false)

  // The column itself is sortable (drag by the header handle); its cards are a
  // nested vertical SortableContext. Card-drag starts from the card body, so the
  // column drag is gated to the handle's listeners only.
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
    isOver,
  } = useSortable({ id: list.id, disabled: !editable || renaming })

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : undefined,
  }

  useEffect(() => {
    if (addingTo === listIdx && inputRef.current) {
      inputRef.current.focus()
    }
  }, [addingTo, listIdx])

  useEffect(() => {
    if (renaming && renameRef.current) {
      renameRef.current.focus()
      renameRef.current.select()
    }
  }, [renaming])

  function startRename() {
    if (!editable) return
    setTitleDraft(list.title)
    renamingRef.current = true
    setRenaming(true)
  }

  function commitRename() {
    if (!renamingRef.current) return  // already committed/cancelled this rename
    renamingRef.current = false
    const trimmed = titleDraft.trim()
    if (trimmed && trimmed !== list.title) onRenameList(trimmed)
    setRenaming(false)
  }

  function cancelRename() {
    renamingRef.current = false
    setRenaming(false)
  }

  function handleDeleteClick() {
    if (list.cards.length > 0) {
      const n = list.cards.length
      const ok = window.confirm(
        `Delete column "${list.title}" and its ${n} ${n === 1 ? 'card' : 'cards'}?`
      )
      if (!ok) return
    }
    onDeleteList()
  }

  const colClass = [
    'kanban-column',
    isOver ? 'is-over' : '',
    isDragging ? 'is-dragging' : '',
  ].filter(Boolean).join(' ')

  return (
    <div ref={setNodeRef} style={style} className={colClass}>
      <div className="kanban-column-header">
        {editable && (
          <button
            ref={setActivatorNodeRef}
            className="kanban-column-drag"
            {...attributes}
            {...listeners}
            title="Drag to reorder column"
            aria-label="Drag to reorder column"
          >
            ⠿
          </button>
        )}

        {renaming ? (
          <input
            ref={renameRef}
            className="kanban-column-rename"
            value={titleDraft}
            onChange={e => setTitleDraft(e.target.value)}
            onBlur={commitRename}
            onKeyDown={e => {
              if (e.key === 'Enter') { e.preventDefault(); commitRename() }
              if (e.key === 'Escape') { e.preventDefault(); cancelRename() }
            }}
          />
        ) : (
          <span
            className={`kanban-column-title${list.complete ? ' is-complete' : ''}`}
            onDoubleClick={startRename}
            role={editable ? 'button' : undefined}
            title={editable ? 'Double-click to rename' : undefined}
          >
            {list.title}
          </span>
        )}

        <span className="kanban-column-count">{list.cards.length}</span>

        {editable && !renaming && (
          <button
            className="kanban-column-delete"
            onClick={handleDeleteClick}
            title="Delete column"
            aria-label="Delete column"
            tabIndex={-1}
          >
            ×
          </button>
        )}
      </div>

      <SortableContext
        items={list.cards.map(c => c.id)}
        strategy={verticalListSortingStrategy}
      >
        <div className="kanban-cards">
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
  const [addingList, setAddingList] = useState(false)
  const [newListTitle, setNewListTitle] = useState('')
  const addListRef = useRef<HTMLInputElement>(null)
  // Tracks an in-flight "add list" so the Enter-keydown and the focused input's
  // blur-on-unmount don't both commit the same new column.
  const addingListRef = useRef(false)

  useEffect(() => {
    if (addingList && addListRef.current) addListRef.current.focus()
  }, [addingList])

  // Baseline tracking — the last-saved markdown (source of truth for patches/upserts)
  const baselineMdRef = useRef<string>(content)
  // Latest board state for use inside async save callbacks
  const boardRef = useRef<AugBoard>(board)
  useEffect(() => { boardRef.current = board })

  const pendingRef = useRef<PendingSave | null>(null)
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => () => { if (saveTimer.current) clearTimeout(saveTimer.current) }, [])

  function showToast(msg: string) {
    setToast(msg)
    setTimeout(() => setToast(null), 3500)
  }

  function revertBoard() {
    setBoard(augment(parseBoard(baselineMdRef.current)))
  }

  // Flush the pending change to the server.
  const flushSave = useCallback(async () => {
    const pending = pendingRef.current
    if (!pending) return
    pendingRef.current = null

    const baselineMd = baselineMdRef.current

    // Guard: if a patch's find string is no longer in the baseline (rapid consecutive
    // edits to the same card within 500ms), fall back to a full surgical upsert.
    let effectiveChange = pending.change
    if ('patch' in effectiveChange && !baselineMd.includes(effectiveChange.patch.find)) {
      effectiveChange = upsertChange(path, applyBoardToBaseline(baselineMd, stripLists(boardRef.current.lists)))
    }

    const result = await sendChange(effectiveChange, baselineMd)

    if ('ok' in result) {
      baselineMdRef.current = applyChangeToBaseline(effectiveChange, baselineMd)
      return
    }

    if ('hashMismatch' in result) {
      // A hashMismatch is often spurious: the two-way sync's own writeback re-normalises
      // the note and bumps its hash even though the board is semantically unchanged.
      // Re-read the current server content, rebase our change onto it, and retry once —
      // so an in-flight edit (e.g. a just-added list) isn't lost to a blind reload.
      const fresh = await fetchNoteContent(path)
      if ('content' in fresh) {
        const rebased = pending.rebase(fresh.content)
        const retry = await sendChange(rebased, fresh.content)
        if ('ok' in retry) {
          baselineMdRef.current = applyChangeToBaseline(rebased, fresh.content)
          return
        }
      }
      // Rebase/retry failed (could not re-read, or a genuine concurrent edit) — only now
      // fall back to a reload, which is the last resort that can lose the in-flight edit.
      showToast('Board changed elsewhere — reloading…')
      setTimeout(() => location.reload(), 1500)
      return
    }

    if ('patchNotFound' in result && 'patch' in effectiveChange) {
      // Patch string not found on server — retry as a surgical upsert of the current board.
      const newMd = applyBoardToBaseline(baselineMd, stripLists(boardRef.current.lists))
      const retryResult = await sendChange(upsertChange(path, newMd), baselineMd)
      if ('ok' in retryResult) {
        baselineMdRef.current = newMd
        return
      }
      showToast('Save failed: ' + ('error' in retryResult ? retryResult.error : 'unknown'))
      revertBoard()
      return
    }

    showToast('Save failed: ' + ('error' in result ? result.error : 'unknown'))
    revertBoard()
  }, [path])  // eslint-disable-line react-hooks/exhaustive-deps

  // Queue a save. `kind` records how to rebase the change if the server baseline
  // shifts under us (hashMismatch). The rebase reads the latest board at flush time.
  function scheduleSave(change: NoteChange, kind: RebaseKind) {
    const rebase = (freshBaseline: string): NoteChange => {
      const lists = stripLists(boardRef.current.lists)
      if (kind === 'structural') {
        return upsertChange(path, applyStructuralChange(freshBaseline, lists))
      }
      // surgical: re-apply the exact card patch if its anchor still exists (least
      // destructive), otherwise re-emit the card lines onto the fresh baseline.
      if ('patch' in change && freshBaseline.includes(change.patch.find)) return change
      return upsertChange(path, applyBoardToBaseline(freshBaseline, lists))
    }
    pendingRef.current = { change, rebase }
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      flushSave().catch(err => {
        showToast(err instanceof Error ? err.message : String(err))
        revertBoard()
      })
    }, 500)
  }

  // Apply a freshly-computed board state and queue its save. The setBoard updater
  // stays pure — the next state and its change are computed here (from the current
  // board), and the save (a side effect) is scheduled outside any updater.
  function applyAndSave(next: AugBoard, change: NoteChange, kind: RebaseKind) {
    setBoard(next)
    scheduleSave(change, kind)
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

    const prev = boardRef.current

    // ── column reorder ── (active id is a column, not a card)
    const activeColIdx = prev.lists.findIndex(l => l.id === activeId)
    if (activeColIdx !== -1) {
      let toIdx = prev.lists.findIndex(l => l.id === overId)
      if (toIdx === -1) toIdx = prev.lists.findIndex(l => l.cards.some(c => c.id === overId))
      if (toIdx === -1 || toIdx === activeColIdx) return
      const lists = arrayMove(prev.lists, activeColIdx, toIdx)
      const newMd = applyStructuralChange(baselineMdRef.current, stripLists(lists))
      applyAndSave({ ...prev, lists }, upsertChange(path, newMd), 'structural')
      return
    }

    // ── card move ──
    const srcListIdx = prev.lists.findIndex(l => l.cards.some(c => c.id === activeId))
    if (srcListIdx === -1) return
    const srcCardIdx = prev.lists[srcListIdx].cards.findIndex(c => c.id === activeId)

    const colIdx = prev.lists.findIndex(l => l.id === overId)
    let dstListIdx: number, dstCardIdx: number

    if (colIdx !== -1) {
      dstListIdx = colIdx
      dstCardIdx = prev.lists[colIdx].cards.length
    } else {
      dstListIdx = prev.lists.findIndex(l => l.cards.some(c => c.id === overId))
      if (dstListIdx === -1) return
      dstCardIdx = prev.lists[dstListIdx].cards.findIndex(c => c.id === overId)
    }

    const lists = prev.lists.map(l => ({ ...l, cards: [...l.cards] }))

    if (srcListIdx === dstListIdx) {
      lists[srcListIdx] = {
        ...lists[srcListIdx],
        cards: arrayMove(lists[srcListIdx].cards, srcCardIdx, dstCardIdx),
      }
    } else {
      const [card] = lists[srcListIdx].cards.splice(srcCardIdx, 1)
      lists[dstListIdx].cards.splice(dstCardIdx, 0, card)
    }

    // Move: always use surgical upsert (single unique patch isn't guaranteed)
    const newMd = applyBoardToBaseline(baselineMdRef.current, stripLists(lists))
    applyAndSave({ ...prev, lists }, upsertChange(path, newMd), 'surgical')
  }

  // ── card ops ──

  function handleToggle(listIdx: number, cardIdx: number) {
    const prev = boardRef.current
    const card = prev.lists[listIdx].cards[cardIdx]
    const oldLine = cardLine(card)
    const newCard = { ...card, checked: !card.checked }
    const newLine = cardLine(newCard)

    const lists = prev.lists.map((l, li) =>
      li !== listIdx
        ? l
        : { ...l, cards: l.cards.map((c, ci) => (ci !== cardIdx ? c : newCard)) }
    )
    applyAndSave({ ...prev, lists }, patchChange(path, oldLine, newLine), 'surgical')
  }

  function handleEdit(listIdx: number, cardIdx: number, text: string) {
    const prev = boardRef.current
    const card = prev.lists[listIdx].cards[cardIdx]
    const oldLine = cardLine(card)
    const newCard = { ...card, text }
    const newLine = cardLine(newCard)

    const lists = prev.lists.map((l, li) =>
      li !== listIdx
        ? l
        : { ...l, cards: l.cards.map((c, ci) => (ci !== cardIdx ? c : newCard)) }
    )
    applyAndSave({ ...prev, lists }, patchChange(path, oldLine, newLine), 'surgical')
  }

  function handleDelete(listIdx: number, cardIdx: number) {
    const prev = boardRef.current
    const card = prev.lists[listIdx].cards[cardIdx]
    const line = cardLine(card)
    const baseline = baselineMdRef.current

    const nextLists = prev.lists.map((l, li) =>
      li !== listIdx ? l : { ...l, cards: l.cards.filter((_, ci) => ci !== cardIdx) }
    )

    // Prefer removing "\nline" (card preceded by another line)
    // Fall back to "line\n" for first card in section
    // Fall back to surgical upsert if neither is unambiguous
    let change: NoteChange
    const findPrev = '\n' + line
    const findNext = line + '\n'
    if (baseline.split(findPrev).length === 2) {
      // Exactly one occurrence with preceding newline
      change = patchChange(path, findPrev, '')
    } else if (baseline.split(findNext).length === 2) {
      change = patchChange(path, findNext, '')
    } else {
      // Ambiguous or missing — use surgical upsert
      change = upsertChange(path, applyBoardToBaseline(baseline, stripLists(nextLists)))
    }
    applyAndSave({ ...prev, lists: nextLists }, change, 'surgical')
  }

  // ── add card ──

  function handleCommitAdd(listIdx: number) {
    const text = newCardText.trim()
    setAddingTo(null)
    setNewCardText('')
    if (!text) return

    const prev = boardRef.current
    const list = prev.lists[listIdx]
    const newLine = cardLine({ text, checked: false })

    const nextLists = prev.lists.map((l, li) =>
      li !== listIdx ? l : { ...l, cards: [...l.cards, { id: nextId(), text, checked: false }] }
    )

    let change: NoteChange
    if (list.cards.length > 0) {
      // Anchor patch on the last card line (unique in the note if possible)
      const lastLine = cardLine(list.cards[list.cards.length - 1])
      change = patchChange(path, lastLine, lastLine + '\n' + newLine)
    } else {
      // Empty column — surgical upsert
      change = upsertChange(path, applyBoardToBaseline(baselineMdRef.current, stripLists(nextLists)))
    }
    applyAndSave({ ...prev, lists: nextLists }, change, 'surgical')
  }

  function handleCancelAdd() {
    setAddingTo(null)
    setNewCardText('')
  }

  // ── list (column) ops ──
  // Structural changes can't be expressed as a per-card patch, so they re-serialise
  // the columns region and upsert it (preserving frontmatter + settings).

  function handleAddList() {
    if (!addingListRef.current) return  // already committed/cancelled this add session
    addingListRef.current = false
    const title = newListTitle.trim()
    setAddingList(false)
    setNewListTitle('')
    if (!title) return

    const prev = boardRef.current
    const lists: AugList[] = [...prev.lists, { id: nextListId(), title, complete: false, cards: [] }]
    const newMd = applyStructuralChange(baselineMdRef.current, stripLists(lists))
    applyAndSave({ ...prev, lists }, upsertChange(path, newMd), 'structural')
  }

  function handleCancelAddList() {
    addingListRef.current = false
    setAddingList(false)
    setNewListTitle('')
  }

  function handleRenameList(listIdx: number, title: string) {
    const prev = boardRef.current
    const lists = prev.lists.map((l, i) => (i === listIdx ? { ...l, title } : l))
    const newMd = applyStructuralChange(baselineMdRef.current, stripLists(lists))
    applyAndSave({ ...prev, lists }, upsertChange(path, newMd), 'structural')
  }

  function handleDeleteList(listIdx: number) {
    const prev = boardRef.current
    const lists = prev.lists.filter((_, i) => i !== listIdx)
    const newMd = applyStructuralChange(baselineMdRef.current, stripLists(lists))
    applyAndSave({ ...prev, lists }, upsertChange(path, newMd), 'structural')
  }

  const title = titleFromPath(path)
  const totalCards = board.lists.reduce((n, l) => n + l.cards.length, 0)

  return (
    <div className="kanban-app">
      {/* ── header ── */}
      <header className="kanban-header" role="banner">
        <div className="kanban-header-inner">
          <div className="kanban-header-left">
            <GridIcon />
            <span className="kanban-header-title">{title}</span>
            <span className="kanban-header-sep" aria-hidden="true">/</span>
            <span className="kanban-header-sub">
              {board.lists.length} {board.lists.length === 1 ? 'column' : 'columns'} · {totalCards} {totalCards === 1 ? 'card' : 'cards'}
            </span>
          </div>
          <div className="kanban-header-right">
            {editable && (
              <span className="kanban-header-tag">
                <span className="kanban-header-tag-dot" aria-hidden="true" />
                Editable
              </span>
            )}
          </div>
        </div>
      </header>

      {/* ── board ── */}
      <main className="kanban-main" role="main">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragEnd={handleDragEnd}
        >
          <div className="kanban-board">
            <SortableContext
              items={board.lists.map(l => l.id)}
              strategy={horizontalListSortingStrategy}
            >
              {board.lists.map((list, listIdx) => (
                <SortableColumn
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
                  onRenameList={title => handleRenameList(listIdx, title)}
                  onDeleteList={() => handleDeleteList(listIdx)}
                />
              ))}
            </SortableContext>

            {editable && (
              <div className="kanban-add-list">
                {addingList ? (
                  <input
                    ref={addListRef}
                    className="kanban-add-list-input"
                    value={newListTitle}
                    placeholder="Column title..."
                    onChange={e => setNewListTitle(e.target.value)}
                    onBlur={handleAddList}
                    onKeyDown={e => {
                      if (e.key === 'Enter') { e.preventDefault(); handleAddList() }
                      if (e.key === 'Escape') { e.preventDefault(); handleCancelAddList() }
                    }}
                  />
                ) : (
                  <button
                    className="kanban-add-list-btn"
                    onClick={() => { addingListRef.current = true; setAddingList(true) }}
                  >
                    + Add list
                  </button>
                )}
              </div>
            )}
          </div>
        </DndContext>
      </main>

      {toast && <div className="kanban-toast" role="alert">{toast}</div>}
    </div>
  )
}
