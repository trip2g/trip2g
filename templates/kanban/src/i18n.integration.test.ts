import { describe, test, expect, beforeEach, afterEach } from 'vitest'
import { detectLang, t } from './i18n'

// ── detectLang ────────────────────────────────────────────────────────────────

describe('detectLang', () => {
  const w = window as unknown as { __trip2g_settings?: { ui_lang?: string } }

  beforeEach(() => {
    // Clear all signals before each test.
    delete w.__trip2g_settings
    document.documentElement.lang = ''
  })

  afterEach(() => {
    // Restore the global set by test-setup.ts for any test that follows.
    w.__trip2g_settings = { ui_lang: 'en' }
    document.documentElement.lang = ''
  })

  test('returns ru for ru-RU in __trip2g_settings', () => {
    w.__trip2g_settings = { ui_lang: 'ru-RU' }
    expect(detectLang()).toBe('ru')
  })

  test('returns en for en-US in __trip2g_settings', () => {
    w.__trip2g_settings = { ui_lang: 'en-US' }
    expect(detectLang()).toBe('en')
  })

  test('returns en for unsupported language (falls back)', () => {
    w.__trip2g_settings = { ui_lang: 'fr' }
    expect(detectLang()).toBe('en')
  })

  test('falls back to document.documentElement.lang when no settings', () => {
    document.documentElement.lang = 'ru'
    expect(detectLang()).toBe('ru')
  })

  test('__trip2g_settings takes priority over document.lang', () => {
    document.documentElement.lang = 'ru'
    w.__trip2g_settings = { ui_lang: 'en' }
    expect(detectLang()).toBe('en')
  })
})

// ── t() – simple strings ─────────────────────────────────────────────────────

describe('t() simple strings', () => {
  test('returns English addCard', () => {
    expect(t('en', 'addCard')).toBe('+ Add card')
  })

  test('returns Russian addCard', () => {
    expect(t('ru', 'addCard')).toBe('+ Добавить карточку')
  })

  test('returns English addList', () => {
    expect(t('en', 'addList')).toBe('+ Add list')
  })

  test('returns Russian addList', () => {
    expect(t('ru', 'addList')).toBe('+ Добавить колонку')
  })

  test('returns English saveFailed prefix', () => {
    expect(t('en', 'saveFailed')).toBe('Save failed: ')
  })

  test('returns Russian saveFailed prefix', () => {
    expect(t('ru', 'saveFailed')).toBe('Ошибка сохранения: ')
  })
})

// ── t() – English plurals ─────────────────────────────────────────────────────

describe('t() English plurals – columns', () => {
  test('1 → singular', () => expect(t('en', 'columns', { count: 1 })).toBe('column'))
  test('0 → plural', () => expect(t('en', 'columns', { count: 0 })).toBe('columns'))
  test('2 → plural', () => expect(t('en', 'columns', { count: 2 })).toBe('columns'))
  test('5 → plural', () => expect(t('en', 'columns', { count: 5 })).toBe('columns'))
})

describe('t() English plurals – cards', () => {
  test('1 → singular', () => expect(t('en', 'cards', { count: 1 })).toBe('card'))
  test('2 → plural', () => expect(t('en', 'cards', { count: 2 })).toBe('cards'))
  test('5 → plural', () => expect(t('en', 'cards', { count: 5 })).toBe('cards'))
})

// ── t() – Russian plurals ─────────────────────────────────────────────────────

describe('t() Russian plurals – columns', () => {
  test('1 → one (колонка)', () => expect(t('ru', 'columns', { count: 1 })).toBe('колонка'))
  test('2 → few (колонки)', () => expect(t('ru', 'columns', { count: 2 })).toBe('колонки'))
  test('5 → many (колонок)', () => expect(t('ru', 'columns', { count: 5 })).toBe('колонок'))
  test('11 → many (колонок)', () => expect(t('ru', 'columns', { count: 11 })).toBe('колонок'))
  test('21 → one (колонка)', () => expect(t('ru', 'columns', { count: 21 })).toBe('колонка'))
  test('22 → few (колонки)', () => expect(t('ru', 'columns', { count: 22 })).toBe('колонки'))
})

describe('t() Russian plurals – cards', () => {
  test('1 → one (карточка)', () => expect(t('ru', 'cards', { count: 1 })).toBe('карточка'))
  test('2 → few (карточки)', () => expect(t('ru', 'cards', { count: 2 })).toBe('карточки'))
  test('5 → many (карточек)', () => expect(t('ru', 'cards', { count: 5 })).toBe('карточек'))
  test('11 → many (карточек)', () => expect(t('ru', 'cards', { count: 11 })).toBe('карточек'))
  test('21 → one (карточка)', () => expect(t('ru', 'cards', { count: 21 })).toBe('карточка'))
})

// ── t() – deleteColumnConfirm interpolation ───────────────────────────────────

describe('t() deleteColumnConfirm', () => {
  test('en: 1 card', () => {
    expect(t('en', 'deleteColumnConfirm', { count: 1, title: 'My Board' }))
      .toBe('Delete column "My Board" and its 1 card?')
  })

  test('en: 2 cards', () => {
    expect(t('en', 'deleteColumnConfirm', { count: 2, title: 'My Board' }))
      .toBe('Delete column "My Board" and its 2 cards?')
  })

  test('ru: 1 карточку', () => {
    expect(t('ru', 'deleteColumnConfirm', { count: 1, title: 'Доска' }))
      .toBe('Удалить колонку "Доска" и 1 карточку?')
  })

  test('ru: 2 карточки (few)', () => {
    expect(t('ru', 'deleteColumnConfirm', { count: 2, title: 'Доска' }))
      .toBe('Удалить колонку "Доска" и 2 карточки?')
  })

  test('ru: 5 карточек (many)', () => {
    expect(t('ru', 'deleteColumnConfirm', { count: 5, title: 'Доска' }))
      .toBe('Удалить колонку "Доска" и 5 карточек?')
  })
})
