import React from 'react'

/** Render inline markdown: **bold**, `code`, [[wikilinks]]. Falls back to plain text. */
export function renderMarkdown(
  text: string,
): React.ReactElement {
  const segments: React.ReactElement[] = []
  // Matches **bold**, `code`, [[Page]] or [[Page|alias]]
  const re = /(\*\*[^*]+\*\*|`[^`]+`|\[\[[^\]]+\]\])/g
  let last = 0
  let key = 0
  let m: RegExpExecArray | null

  while ((m = re.exec(text)) !== null) {
    if (m.index > last) {
      segments.push(<span key={key++}>{text.slice(last, m.index)}</span>)
    }

    const token = m[0]
    if (token.startsWith('**')) {
      segments.push(<strong key={key++}>{token.slice(2, -2)}</strong>)
    } else if (token.startsWith('`')) {
      segments.push(<code key={key++}>{token.slice(1, -1)}</code>)
    } else {
      // wikilink: [[Page]] or [[Page|alias]]
      const inner = token.slice(2, -2)
      const pipeIdx = inner.indexOf('|')
      const target = pipeIdx !== -1 ? inner.slice(0, pipeIdx) : inner
      const label = pipeIdx !== -1 ? inner.slice(pipeIdx + 1) : inner
      segments.push(
        <a key={key++} href={`/${target}`}>
          {label}
        </a>
      )
    }

    last = m.index + token.length
  }

  if (last < text.length) {
    segments.push(<span key={key++}>{text.slice(last)}</span>)
  }

  return <>{segments}</>
}
