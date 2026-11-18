import type { MarkdownConfig } from '@lezer/markdown'

const BANG = 33
const OPEN = 91
const NEWLINE = 10

export const wikilinkSyntax: MarkdownConfig = {
  defineNodes: ['WikiLink', 'Embed'],
  parseInline: [{
    name: 'WikiLink',
    before: 'Link',
    parse(cx, next, pos) {
      const embed = next === BANG
      const open = embed ? pos + 1 : pos
      if (cx.char(open) !== OPEN || cx.char(open + 1) !== OPEN) return -1
      for (let i = open + 2; i < cx.end - 1; i++) {
        const c = cx.char(i)
        if (c === NEWLINE) return -1
        if (c === 93 && cx.char(i + 1) === 93) {
          if (i === open + 2) return -1
          return cx.addElement(cx.elt(embed ? 'Embed' : 'WikiLink', pos, i + 2))
        }
      }
      return -1
    },
  }],
}
