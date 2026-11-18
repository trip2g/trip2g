import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { markdown, markdownKeymap, markdownLanguage } from '@codemirror/lang-markdown'
import { Compartment, EditorState, Prec } from '@codemirror/state'
import { drawSelection, EditorView, keymap } from '@codemirror/view'
import { livePreview } from './livepreview'
import { darkVars, pageRootPx, pageTheme } from './theme'
import { wikilinkSyntax } from './wikilink'

type Options = {
  doc?: string
  dark?: boolean
  onChange?: (doc: string) => void
  onLinkClick?: (offset: number) => boolean
}

function create(opts: Options = {}) {
  const themeSlot = new Compartment()

  const linkClicks = EditorView.domEventHandlers({
    mousedown(e, view) {
      if (e.button !== 0) return false
      const target = e.target as HTMLElement
      const rendered = target.closest('.cm-lp-wikilink')
      if (!rendered && !(e.ctrlKey || e.metaKey)) return false
      const pos = view.posAtCoords({ x: e.clientX, y: e.clientY })
      if (pos === null || !opts.onLinkClick) return false
      if (!opts.onLinkClick(pos)) return false
      e.preventDefault()
      return true
    },
  })

  let dark = !!opts.dark
  const extensions = () => [
    EditorState.lineSeparator.of('\n'),
    history(),
    drawSelection(),
    EditorView.lineWrapping,
    markdown({ base: markdownLanguage, extensions: [wikilinkSyntax], addKeymap: false }),
    keymap.of([...markdownKeymap, ...defaultKeymap, ...historyKeymap, indentWithTab]),
    livePreview,
    pageTheme,
    themeSlot.of(dark ? Prec.high(darkVars) : []),
    linkClicks,
    EditorView.updateListener.of(u => {
      if (!u.docChanged) return
      opts.onChange?.(u.state.doc.toString())
    }),
  ]

  const view = new EditorView({
    state: EditorState.create({ doc: opts.doc ?? '', extensions: extensions() }),
  })

  const fitRoot = () => view.dom.style.setProperty('--lp-root', `${pageRootPx(window.innerWidth)}px`)
  fitRoot()
  window.addEventListener('resize', fitRoot)

  return {
    view,
    dom: view.dom,
    getDoc: () => view.state.doc.toString(),
    setDoc(doc: string) {
      if (doc === view.state.doc.toString()) return
      view.setState(EditorState.create({ doc, extensions: extensions() }))
    },
    setDark(next: boolean) {
      dark = next
      view.dispatch({ effects: themeSlot.reconfigure(next ? Prec.high(darkVars) : []) })
    },
    focus: () => view.focus(),
    destroy() {
      window.removeEventListener('resize', fitRoot)
      view.destroy()
    },
  }
}

export type CMEditor = ReturnType<typeof create>

;(window as any).$trip2g_cmeditor = { create, find: (el: HTMLElement) => EditorView.findFromDOM(el) }
