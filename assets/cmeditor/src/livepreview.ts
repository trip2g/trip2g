import { syntaxTree } from '@codemirror/language'
import type { EditorState, Range } from '@codemirror/state'
import { Decoration, EditorView, ViewPlugin, WidgetType } from '@codemirror/view'
import type { DecorationSet, ViewUpdate } from '@codemirror/view'
import type { SyntaxNodeRef } from '@lezer/common'

const FRONTMATTER = /^---\r?\n[\s\S]*?\r?\n---[ \t]*(\r?\n|$)/

class BulletWidget extends WidgetType {
  eq() {
    return true
  }

  toDOM() {
    const el = document.createElement('span')
    el.className = 'cm-lp-bullet'
    el.textContent = '•'
    return el
  }
}

class CheckboxWidget extends WidgetType {
  constructor(readonly checked: boolean) {
    super()
  }

  eq(other: CheckboxWidget) {
    return other.checked === this.checked
  }

  toDOM(view: EditorView) {
    const box = document.createElement('input')
    box.type = 'checkbox'
    box.className = 'cm-lp-checkbox'
    box.checked = this.checked
    box.addEventListener('mousedown', e => e.preventDefault())
    box.addEventListener('click', e => {
      e.preventDefault()
      const pos = view.posAtDOM(box)
      const marker = view.state.sliceDoc(pos, pos + 3)
      if (!/^\[[ xX]\]$/.test(marker)) return
      view.dispatch({ changes: { from: pos + 1, to: pos + 2, insert: marker[1] === ' ' ? 'x' : ' ' } })
    })
    return box
  }

  ignoreEvent() {
    return true
  }
}

const hidden = Decoration.replace({})
const bullet = Decoration.replace({ widget: new BulletWidget() })
const checked = Decoration.replace({ widget: new CheckboxWidget(true) })
const unchecked = Decoration.replace({ widget: new CheckboxWidget(false) })
const monoLine = Decoration.line({ class: 'cm-lp-mono' })
const markLine = Decoration.mark({ class: 'cm-lp-marker' })

const inlineClass: Record<string, string> = {
  Emphasis: 'cm-lp-em',
  StrongEmphasis: 'cm-lp-strong',
  Strikethrough: 'cm-lp-strike',
  InlineCode: 'cm-lp-code',
}

const inlineMarks = new Set(['EmphasisMark', 'StrikethroughMark', 'CodeMark'])

function activeLines(state: EditorState): Set<number> {
  const lines = new Set<number>()
  for (const r of state.selection.ranges) {
    const a = state.doc.lineAt(r.from).number
    const b = state.doc.lineAt(r.to).number
    for (let n = a; n <= b; n++) lines.add(n)
  }
  return lines
}

function touches(state: EditorState, from: number, to: number): boolean {
  return state.selection.ranges.some(r => r.from <= to && r.to >= from)
}

function frontmatterEnd(state: EditorState): number {
  const head = state.sliceDoc(0, Math.min(state.doc.length, 20000))
  const m = FRONTMATTER.exec(head)
  return m ? m[0].length : 0
}

function monoLines(state: EditorState, from: number, to: number, out: Range<Decoration>[]) {
  let line = state.doc.lineAt(from)
  for (;;) {
    out.push(monoLine.range(line.from))
    if (line.to >= to || line.number >= state.doc.lines) break
    line = state.doc.line(line.number + 1)
  }
}

function isCallout(state: EditorState, from: number): boolean {
  const line = state.doc.lineAt(from)
  return /^\s*>\s*\[!/.test(line.text)
}

function build(view: EditorView): DecorationSet {
  const { state } = view
  const out: Range<Decoration>[] = []
  const active = activeLines(state)
  const fmEnd = frontmatterEnd(state)
  const lineOf = (pos: number) => state.doc.lineAt(pos).number

  if (fmEnd > 0) monoLines(state, 0, fmEnd - 1, out)

  for (const { from, to } of view.visibleRanges) {
    syntaxTree(state).iterate({
      from,
      to,
      enter(node: SyntaxNodeRef) {
        if (node.to <= fmEnd) return false
        const name = node.name

        if (name === 'FencedCode' || name === 'CodeBlock' || name === 'Table' || name === 'HTMLBlock' || name === 'CommentBlock') {
          monoLines(state, Math.max(node.from, from), Math.min(node.to, to), out)
          return false
        }
        if (name === 'Blockquote' && isCallout(state, node.from)) {
          monoLines(state, Math.max(node.from, from), Math.min(node.to, to), out)
          return false
        }
        if (name === 'Embed') {
          out.push(Decoration.mark({ class: 'cm-lp-code' }).range(node.from, node.to))
          return false
        }

        const heading = /^ATXHeading(\d)$/.exec(name)
        if (heading) {
          const line = state.doc.lineAt(node.from)
          out.push(Decoration.line({ class: `cm-lp-h cm-lp-h${heading[1]}` }).range(line.from))
          if (active.has(line.number)) {
            const mark = node.node.getChild('HeaderMark')
            if (mark) out.push(markLine.range(mark.from, mark.to))
            return
          }
          for (const mark of node.node.getChildren('HeaderMark')) {
            if (mark.from === node.from) {
              const after = state.sliceDoc(mark.to, mark.to + 1) === ' ' ? mark.to + 1 : mark.to
              out.push(hidden.range(mark.from, after))
            } else {
              out.push(hidden.range(mark.from, mark.to))
            }
          }
          return
        }

        if (inlineClass[name]) {
          out.push(Decoration.mark({ class: inlineClass[name] }).range(node.from, node.to))
          const inside = touches(state, node.from, node.to)
          let child = node.node.firstChild
          while (child) {
            if (inlineMarks.has(child.name) && child.from < child.to) {
              out.push((inside ? markLine : hidden).range(child.from, child.to))
            }
            child = child.nextSibling
          }
          return name !== 'InlineCode'
        }

        if (name === 'Link') {
          const marks = node.node.getChildren('LinkMark')
          const url = node.node.getChild('URL')
          if (marks.length < 2 || !url) return
          const textFrom = marks[0].to
          const textTo = marks[1].from
          if (textTo > textFrom) out.push(Decoration.mark({ class: 'cm-lp-link' }).range(textFrom, textTo))
          if (!touches(state, node.from, node.to)) {
            out.push(hidden.range(marks[0].from, marks[0].to))
            out.push(hidden.range(marks[1].from, node.to))
          } else {
            out.push(markLine.range(marks[0].from, marks[0].to))
            out.push(markLine.range(marks[1].from, node.to))
          }
          return
        }

        if (name === 'WikiLink') {
          const inner = state.sliceDoc(node.from + 2, node.to - 2)
          const pipe = inner.indexOf('|')
          const labelFrom = pipe >= 0 ? node.from + 2 + pipe + 1 : node.from + 2
          const labelTo = node.to - 2
          const inside = touches(state, node.from, node.to)
          if (inside || labelTo <= labelFrom) {
            out.push(Decoration.mark({ class: 'cm-lp-wikilink-src' }).range(node.from, node.to))
            return false
          }
          out.push(hidden.range(node.from, labelFrom))
          out.push(Decoration.mark({ class: 'cm-lp-wikilink' }).range(labelFrom, labelTo))
          out.push(hidden.range(labelTo, node.to))
          return false
        }

        if (name === 'ListMark') {
          const item = node.node.parent
          const list = item?.parent
          if (!item || !list) return
          if (active.has(lineOf(node.from))) return
          const task = item.getChild('Task')
          if (task) {
            const after = state.sliceDoc(node.to, node.to + 1) === ' ' ? node.to + 1 : node.to
            out.push(hidden.range(node.from, after))
            return
          }
          if (list.name === 'BulletList') out.push(bullet.range(node.from, node.to))
          return
        }

        if (name === 'TaskMarker') {
          if (active.has(lineOf(node.from))) return
          const mark = state.sliceDoc(node.from, node.to)
          out.push((mark[1] === ' ' ? unchecked : checked).range(node.from, node.to))
          return
        }
      },
    })
  }

  return Decoration.set(out, true)
}

export const livePreview = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet

    constructor(view: EditorView) {
      this.decorations = build(view)
    }

    update(u: ViewUpdate) {
      if (u.docChanged || u.selectionSet || u.viewportChanged || syntaxTree(u.startState) !== syntaxTree(u.state)) {
        this.decorations = build(u.view)
      }
    }
  },
  { decorations: v => v.decorations },
)
