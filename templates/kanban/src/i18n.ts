export type Lang = 'en' | 'ru'

export type Params = { count?: number; title?: string }

// Plural-form entry: English uses one/other; Russian uses one/few/many.
type PluralEntry = { one: string; few?: string; other?: string; many?: string }

interface LangStrings {
  addCard: string
  addList: string
  cardPlaceholder: string
  columnPlaceholder: string
  deleteCard: string
  deleteColumn: string
  dragColumn: string
  doubleClickEdit: string
  doubleClickRename: string
  editable: string
  boardChangedReloading: string
  boardConflictReloading: string
  boardUpdated: string
  boardDeleted: string
  liveDisconnected: string
  notABoardTitle: string
  notABoardBody: string
  lastEditedBy: string
  saveFailed: string
  // Plural keys — call t(lang, key, { count }) to get the right form.
  columns: PluralEntry
  cards: PluralEntry
  // Interpolated plural — call t(lang, 'deleteColumnConfirm', { count, title }).
  deleteColumnConfirm: PluralEntry
}

const STRINGS: Record<Lang, LangStrings> = {
  en: {
    addCard: '+ Add card',
    addList: '+ Add list',
    cardPlaceholder: 'Card text...',
    columnPlaceholder: 'Column title...',
    deleteCard: 'Delete card',
    deleteColumn: 'Delete column',
    dragColumn: 'Drag to reorder column',
    doubleClickEdit: 'Double-click to edit',
    doubleClickRename: 'Double-click to rename',
    editable: 'Editable',
    boardChangedReloading: 'Board changed elsewhere — reloading…',
    boardConflictReloading: 'Edit conflict with a change elsewhere — reloading…',
    boardUpdated: 'Board updated',
    boardDeleted: 'Board deleted',
    liveDisconnected: 'Live updates disconnected',
    notABoardTitle: 'Not a Kanban board',
    notABoardBody: 'This note has "layout: kanban" but no columns. Add "## Column" headings to create columns, or remove "layout: kanban" from the frontmatter.',
    lastEditedBy: 'Last edited by',
    saveFailed: 'Save failed: ',
    columns: { one: 'column', other: 'columns' },
    cards: { one: 'card', other: 'cards' },
    deleteColumnConfirm: {
      one: 'Delete column "{title}" and its {count} card?',
      other: 'Delete column "{title}" and its {count} cards?',
    },
  },
  ru: {
    addCard: '+ Добавить карточку',
    addList: '+ Добавить колонку',
    cardPlaceholder: 'Текст карточки…',
    columnPlaceholder: 'Название колонки…',
    deleteCard: 'Удалить карточку',
    deleteColumn: 'Удалить колонку',
    dragColumn: 'Перетащить для изменения порядка',
    doubleClickEdit: 'Двойной клик для редактирования',
    doubleClickRename: 'Двойной клик для переименования',
    editable: 'Редактируется',
    boardChangedReloading: 'Доска изменена в другом месте — перезагрузка…',
    boardConflictReloading: 'Конфликт правок с изменением в другом месте — перезагрузка…',
    boardUpdated: 'Доска обновлена',
    boardDeleted: 'Доска удалена',
    liveDisconnected: 'Живые обновления отключены',
    notABoardTitle: 'Это не Kanban-доска',
    notABoardBody: 'В этой заметке указан «layout: kanban», но нет колонок. Добавьте заголовки «## Колонка», чтобы создать колонки, или удалите «layout: kanban» из метаданных.',
    lastEditedBy: 'Последняя правка:',
    saveFailed: 'Ошибка сохранения: ',
    // Russian plural: one (1, 21, 31…), few (2–4, 22–24…), many (0, 5–20, 25–30…).
    columns: { one: 'колонка', few: 'колонки', many: 'колонок' },
    cards: { one: 'карточка', few: 'карточки', many: 'карточек' },
    deleteColumnConfirm: {
      one: 'Удалить колонку "{title}" и {count} карточку?',
      few: 'Удалить колонку "{title}" и {count} карточки?',
      many: 'Удалить колонку "{title}" и {count} карточек?',
    },
  },
}

/** Select the correct plural form for `n` given the language and form map. */
function selectPlural(lang: Lang, n: number, entry: PluralEntry): string {
  if (lang === 'ru') {
    const m10 = n % 10
    const m100 = n % 100
    if (m10 === 1 && m100 !== 11) return entry.one
    if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return entry.few ?? entry.one
    return entry.many ?? entry.one
  }
  // English and default: 1 → one, everything else → other.
  return n === 1 ? entry.one : (entry.other ?? entry.one)
}

function interpolate(str: string, params?: Params): string {
  let result = str
  if (params?.count !== undefined) result = result.replace('{count}', String(params.count))
  if (params?.title !== undefined) result = result.replace('{title}', params.title)
  return result
}

/** Translate a key for the given language with optional count/title params. */
export function t(lang: Lang, key: keyof LangStrings, params?: Params): string {
  const entry = STRINGS[lang][key]
  if (typeof entry === 'string') return interpolate(entry, params)
  if (params?.count !== undefined) {
    return interpolate(selectPlural(lang, params.count, entry), params)
  }
  return entry.one
}

/**
 * Detect the page language.
 *
 * Priority:
 * 1. `window.__trip2g_settings.ui_lang` (set by the trip2g layout)
 * 2. `document.documentElement.lang`
 * 3. `navigator.language`
 * 4. Falls back to `'en'` for any unsupported language.
 */
export function detectLang(): Lang {
  const w = window as unknown as { __trip2g_settings?: { ui_lang?: string } }
  const raw =
    w.__trip2g_settings?.ui_lang ||
    document.documentElement.lang ||
    navigator.language ||
    'en'
  const code = raw.split('-')[0].toLowerCase()
  return code === 'ru' ? 'ru' : 'en'
}
