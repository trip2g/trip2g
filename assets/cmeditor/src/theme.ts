import { EditorView } from '@codemirror/view'

const sans = 'system-ui, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, Helvetica, Arial, "Helvetica Neue", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"'
const mono = 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace'

export const pageTheme = EditorView.theme({
  '&': {
    '--lp-color': '#373c44',
    '--lp-h1': '#2d3138',
    '--lp-h2': '#373c44',
    '--lp-h3': '#424751',
    '--lp-muted': '#646b79',
    '--lp-primary': '#0172ad',
    '--lp-code-bg': 'rgb(243, 244.5, 246.75)',
    '--lp-code-color': '#646b79',
    '--lp-del': 'rgb(136, 56.5, 53)',
    '--lp-bg': '#fff',
    colorScheme: 'light',
    color: 'var(--lp-color)',
    backgroundColor: 'var(--lp-bg)',
    flexGrow: '1',
    '--lp-root': '16px',
    height: '100%',
    fontSize: 'var(--lp-root)',
  },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': {
    fontFamily: sans,
    lineHeight: '1.75',
  },
  '.cm-content': {
    fontSize: 'var(--lp-root)',
    maxWidth: '80ch',
    margin: '0 auto',
    padding: '1.5rem 1rem 40vh',
    caretColor: 'var(--lp-color)',
  },
  '.cm-line': {
    padding: '0',
    tabSize: '4',
    fontSize: 'clamp(calc(0.9 * var(--lp-root)), calc(0.81 * var(--lp-root) + 0.225vw), calc(1.0125 * var(--lp-root)))',
  },
  '.cm-cursor': { borderLeftColor: 'var(--lp-color)' },
  '.cm-lp-h': { fontWeight: '700', lineHeight: '1.25' },
  '.cm-lp-h1': { fontSize: 'calc(2 * var(--lp-root))', color: 'var(--lp-h1)', paddingTop: '0.6em', paddingBottom: '0.3em' },
  '.cm-lp-h2': { fontSize: 'calc(1.5 * var(--lp-root))', color: 'var(--lp-h2)', paddingTop: '0.9em', paddingBottom: '0.25em' },
  '.cm-lp-h3': { fontSize: 'calc(1.25 * var(--lp-root))', color: 'var(--lp-h3)', paddingTop: '0.8em', paddingBottom: '0.2em' },
  '.cm-lp-h4': { fontSize: 'calc(1.125 * var(--lp-root))', paddingTop: '0.6em' },
  '.cm-lp-h5': { fontSize: 'calc(1 * var(--lp-root))', paddingTop: '0.5em' },
  '.cm-lp-h6': { fontSize: 'calc(0.875 * var(--lp-root))', paddingTop: '0.5em' },
  '.cm-lp-marker': { color: 'var(--lp-muted)', opacity: '0.7' },
  '.cm-lp-em': { fontStyle: 'italic' },
  '.cm-lp-strong': { fontWeight: '700' },
  '.cm-lp-strike': { textDecoration: 'line-through', color: 'var(--lp-del)' },
  '.cm-lp-code': {
    fontFamily: mono,
    fontSize: '0.875em',
    backgroundColor: 'var(--lp-code-bg)',
    color: 'var(--lp-code-color)',
    borderRadius: '0.25rem',
    padding: '0.1em 0.25em',
  },
  '.cm-lp-link, .cm-lp-wikilink': {
    color: 'var(--lp-primary)',
    textDecoration: 'underline',
    textDecorationColor: 'rgba(1, 114, 173, 0.5)',
    textUnderlineOffset: '0.125rem',
  },
  '.cm-lp-wikilink': { cursor: 'pointer' },
  '.cm-lp-wikilink-src': { color: 'var(--lp-primary)' },
  '.cm-line.cm-lp-mono': {
    fontFamily: mono,
    fontSize: 'calc(0.875 * var(--lp-root))',
    lineHeight: '1.6',
    color: 'var(--lp-code-color)',
    backgroundColor: 'var(--lp-code-bg)',
  },
  '.cm-lp-bullet': { display: 'inline-block', width: '1ch', color: 'var(--lp-muted)' },
  '.cm-lp-checkbox': {
    width: '1em',
    height: '1em',
    margin: '0 0.4em 0 0',
    verticalAlign: '-0.15em',
    cursor: 'pointer',
    accentColor: 'var(--lp-primary)',
  },
})

const picoRoot: [number, number][] = [[1536, 18], [1280, 20], [1024, 19], [768, 18], [576, 17]]

export function pageRootPx(width: number): number {
  return picoRoot.find(([min]) => width >= min)?.[1] ?? 16
}

export const darkVars = EditorView.theme({
  '&': {
    '--lp-color': '#c2c7d0',
    '--lp-h1': '#f0f1f3',
    '--lp-h2': '#e0e3e7',
    '--lp-h3': '#c2c7d0',
    '--lp-muted': '#7b8495',
    '--lp-primary': '#01aaff',
    '--lp-code-bg': 'rgb(26, 30.5, 40.25)',
    '--lp-code-color': '#8891a4',
    '--lp-del': 'rgb(205.5, 126, 123)',
    '--lp-bg': 'rgb(19, 22.5, 30.5)',
    colorScheme: 'dark',
  },
}, { dark: true })
