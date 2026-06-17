// mermaid widget — finds ```mermaid fenced blocks (rendered by goldmark as
// <pre><code class="language-mermaid">), converts them to <div class="mermaid">
// containers, and renders them with mermaid.
//
// mermaid itself is heavy, so it is loaded lazily (only when a page actually has
// a mermaid block) from /assets/mermaid.min.js — mirroring the datachart/echarts
// split. The backend only emits this glue on pages that contain a mermaid block.
//
// Diagrams follow the site's light/dark theme: the theme switcher toggles the
// `dark` class on <html>, which we observe and re-render on.

interface Block {
  el: HTMLElement;
  src: string; // original diagram source, kept for re-rendering on theme change
}

function collectBlocks(): Block[] {
  const codes = Array.from(
    document.querySelectorAll<HTMLElement>('pre > code.language-mermaid'),
  );
  const blocks: Block[] = [];
  for (const code of codes) {
    const pre = code.parentElement;
    if (!pre) continue;
    const div = document.createElement('div');
    div.className = 'mermaid';
    // textContent decodes the HTML entities goldmark escaped (e.g. --&gt; → -->).
    const src = code.textContent || '';
    div.textContent = src;
    pre.replaceWith(div);
    blocks.push({ el: div, src });
  }
  return blocks;
}

function currentTheme(): string {
  return document.documentElement.classList.contains('dark') ? 'dark' : 'default';
}

function renderAll(blocks: Block[]) {
  const mermaid = (window as any).mermaid;
  mermaid.initialize({
    startOnLoad: false,
    theme: currentTheme(),
    securityLevel: 'strict',
  });
  // Reset each node so mermaid re-processes it (it skips data-processed nodes
  // and reads the diagram source from textContent).
  for (const b of blocks) {
    b.el.removeAttribute('data-processed');
    b.el.textContent = b.src;
  }
  mermaid.run({ nodes: blocks.map((b) => b.el) });
}

// Re-render when the theme switcher changes the theme. $trip2g_theme (in the
// core mol bundle) invokes every registered listener after it flips the theme.
function onThemeChange(cb: () => void) {
  const w = window as any;
  (w.trip2g_theme_listeners ||= []).push(cb);
}

function loadMermaid(cb: () => void) {
  if ((window as any).mermaid) { cb(); return; }
  let s = document.getElementById('mermaid-lib') as HTMLScriptElement | null;
  if (s) { s.addEventListener('load', cb); return; }
  s = document.createElement('script');
  s.id = 'mermaid-lib';
  s.src = '/assets/mermaid.min.js';
  s.onload = cb;
  document.head.appendChild(s);
}

function initMermaid() {
  const blocks = collectBlocks();
  if (blocks.length === 0) return;
  loadMermaid(() => {
    renderAll(blocks);
    onThemeChange(() => renderAll(blocks));
  });
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initMermaid);
} else {
  initMermaid();
}
