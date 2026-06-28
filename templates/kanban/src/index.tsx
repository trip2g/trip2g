import React from 'react'
import { createRoot } from 'react-dom/client'
import Board from './Board'
// CSS is bundled as text and injected at runtime (esbuild loader: '.css' -> 'text')
import cssText from './styles.css'

// Inject styles once
const styleEl = document.createElement('style')
styleEl.textContent = cssText as unknown as string
document.head.appendChild(styleEl)

// Data island: the layout sets window.__trip2g_kanban via an inline script
// that reads from the textarea + JSON meta elements (see _layouts/kanban.html).
interface KanbanData {
  path: string
  content: string
  editable: boolean
  // Server-escaped display name of the last editor (admin-only; absent otherwise).
  lastEditedBy?: string
}

function getData(): KanbanData {
  // Prefer the window global set by the inline bootstrap script
  const w = window as unknown as { __trip2g_kanban?: KanbanData }
  if (w.__trip2g_kanban) return w.__trip2g_kanban

  // Fallback: raw JSON element (alternate embed strategy)
  const el = document.getElementById('kanban-data')
  if (el) return JSON.parse(el.textContent ?? '{}') as KanbanData

  throw new Error('trip2g-kanban: data island not found')
}

const data = getData()

const mountEl = document.getElementById('trip2g-kanban-root')
if (!mountEl) throw new Error('trip2g-kanban: #trip2g-kanban-root not found')

createRoot(mountEl).render(
  <React.StrictMode>
    <Board
      path={data.path}
      content={data.content}
      editable={data.editable}
      lastEditedBy={data.lastEditedBy}
    />
  </React.StrictMode>
)
