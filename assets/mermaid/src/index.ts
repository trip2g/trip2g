// mermaid widget — finds ```mermaid fenced blocks (rendered by goldmark as
// <pre><code class="language-mermaid">), converts them to <div class="mermaid">
// containers, and renders each with one of two engines depending on the
// diagram type:
//
//   - beautiful-mermaid (github.com/lukilabs/beautiful-mermaid) — a
//     synchronous, from-scratch renderer for flowchart/state/sequence/class/
//     ER/xychart diagrams. Prettier output, no DOM deps. Heavy (pulls in
//     ELK.js), so it is its own lazy chunk (assets/beautifulmermaid.min.js),
//     loaded only when a block needs it.
//   - mermaid.js — the reference implementation, kept as the lazy-loaded
//     (assets/mermaid.min.js) fallback for every diagram type
//     beautiful-mermaid doesn't support (gantt, pie, gitGraph, mindmap, …)
//     and for any beautiful-mermaid parse failure.
//
// Both engines' output is enhanced with pan/zoom, fullscreen lightbox and
// PNG/SVG export via @mostlylucid/mermaid-enhancements (bundled eagerly here
// — it's ~15 KB gzipped, small enough not to need its own lazy chunk).
//
// Diagrams follow the site's light/dark theme: the theme switcher toggles the
// `dark` class on <html>, which we observe and re-render mermaid.js diagrams
// on. beautiful-mermaid diagrams re-theme for free — their colors are CSS
// custom properties on the SVG, set here to `var(--pico-*)`, so the browser's
// normal cascade repaints them without any re-render.

import { enhanceMermaidDiagrams, configure as configureEnhancements } from '@mostlylucid/mermaid-enhancements';

// The default template's global text color (e.g. `p { color: … }`) leaks into
// mermaid's foreignObject HTML labels and, matching the inner <p> directly,
// beats the color mermaid sets on the label span (from the diagram theme or a
// classDef) — washing node/edge/cluster text to the body color regardless of
// contrast. Force label descendants to inherit the span's mermaid color.
const LABEL_STYLE_ID = 'mermaid-label-style';
const LABEL_CSS = `
.mermaid .nodeLabel *,
.mermaid .edgeLabel *,
.mermaid .cluster-label * { color: inherit; }
`;

// @mostlylucid/mermaid-enhancements ships its own stylesheet + Boxicons-based
// toolbar; both would mean a second self-hosted asset (CSS file + icon font)
// for a self-hosted-only site. Instead we inject an equivalent stylesheet
// here: same class names (`.mermaid-wrapper`, `.mermaid-controls`, …) so the
// library's DOM wiring is untouched, but colors mapped to trip2g's Pico
// variables and icons as plain glyphs (no icon font).
//
// `max-height: 80vh` bounds the diagram column: mermaid-enhancements' own
// `fit: true` / `center: true` pan-zoom setup scales each diagram to fill its
// wrapper, so a large diagram fits the (viewport-bounded) box on load instead
// of shrinking to unreadable size the way mermaid's default `useMaxWidth`
// did — and the same fit-to-box math also fixes wide-but-short diagrams
// (previously a sliver): scale is bounded by whichever dimension is tighter,
// so a wide diagram fits by width and sits centered in the box rather than
// being stretched to fill the height. Full detail beyond fit level is one
// zoom/pan or fullscreen away.
const ENH_STYLE_ID = 'mermaid-enhancements-style';
const ENH_CSS = `
.mermaid { visibility: hidden; }
.mermaid[data-processed="true"] { visibility: visible; }

.mermaid-wrapper {
  position: relative;
  border-radius: 0.5rem;
  overflow: hidden;
  width: 100%;
  max-height: 80vh;
  margin: 1rem 0;
}
.mermaid-wrapper .mermaid {
  margin: 0;
  width: 100%;
  min-height: 300px;
  max-height: 80vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
}
.mermaid-wrapper .mermaid svg {
  width: 100% !important;
  height: auto !important;
  max-height: 80vh !important;
}

.mermaid-controls {
  position: absolute; top: 0.5rem; right: 0.5rem;
  display: flex; gap: 0.25rem; z-index: 10;
  background: var(--pico-card-background-color);
  border: 1px solid var(--pico-muted-border-color);
  border-radius: 0.4rem; padding: 0.25rem;
  opacity: 0.7; transition: opacity 0.15s;
}
.mermaid-wrapper:hover .mermaid-controls,
.mermaid-lightbox-diagram-wrapper .mermaid-controls { opacity: 1; }
.mermaid-control-btn {
  width: 2.2rem; height: 2.2rem; padding: 0; margin: 0;
  display: flex; align-items: center; justify-content: center;
  border: none; border-radius: 0.3rem; background: transparent;
  color: var(--pico-color); cursor: pointer; font-size: 1.05rem; line-height: 1;
}
.mermaid-control-btn:hover { background: var(--pico-muted-border-color); }
.mm-fullscreen::before { content: '⛶'; }
.mm-zoom-in::before { content: '+'; }
.mm-zoom-out::before { content: '−'; }
.mm-reset::before { content: '⟲'; }
.mm-export-png::before { content: 'PNG'; font-size: 0.6rem; font-weight: 600; }
.mm-export-svg::before { content: 'SVG'; font-size: 0.6rem; font-weight: 600; }
.mermaid-control-btn.bx.bx-x::before { content: '✕'; }

/* beautiful-mermaid sets \`text { font-family: '<font>', … }\` inside the SVG;
   override to match the site's own font instead of shipping/naming another. */
.mermaid-wrapper .mermaid svg text { font-family: var(--pico-font-family) !important; }

.mermaid-lightbox {
  position: fixed; inset: 0; z-index: 1000;
  display: flex; align-items: center; justify-content: center;
  background: rgba(0, 0, 0, 0.7); backdrop-filter: blur(4px);
}
.mermaid-lightbox-content {
  position: relative; width: 92%; height: 88%;
  background: var(--pico-background-color);
  border: 1px solid var(--pico-muted-border-color);
  border-radius: 0.5rem;
}
.mermaid-lightbox-diagram-wrapper { width: 100%; height: 100%; padding: 2rem; box-sizing: border-box; }
.mermaid-lightbox-diagram { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }
.mermaid-lightbox-diagram svg {
  width: 100% !important; height: 100% !important;
  max-width: 100% !important; max-height: 100% !important;
}
`;

function injectStyle(id: string, css: string) {
  if (document.getElementById(id)) return;
  const s = document.createElement('style');
  s.id = id;
  s.textContent = css;
  document.head.appendChild(s);
}

configureEnhancements({
  icons: {
    fullscreen: 'mm-fullscreen',
    zoomIn: 'mm-zoom-in',
    zoomOut: 'mm-zoom-out',
    reset: 'mm-reset',
    exportPng: 'mm-export-png',
    exportSvg: 'mm-export-svg',
  },
});

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

// beautiful-mermaid's colors are CSS custom properties on the <svg> element;
// passing `var(--pico-*)` (rather than a resolved hex) means the diagram
// re-themes for free whenever Pico's own variables flip on the [data-theme]
// attribute — no re-render, no theme-change listener needed for this path.
//
// `font` is deliberately omitted (defaults to 'Inter'): it's baked into a
// quoted CSS string (`font-family: '<font>', …`), so a `var(--pico-*)`
// reference wouldn't resolve there — we override the real font via CSS
// (`.mermaid-wrapper .mermaid svg text` above) instead.
function beautifulThemeOptions() {
  return {
    bg: 'var(--pico-background-color)',
    fg: 'var(--pico-color)',
    accent: 'var(--pico-primary)',
    muted: 'var(--pico-muted-color)',
    surface: 'var(--pico-card-background-color)',
    border: 'var(--pico-muted-border-color)',
    line: 'var(--pico-muted-border-color)',
    transparent: true,
  };
}

// renderMermaidSVG always embeds a Google Fonts @import for whatever `font`
// name is in effect (default 'Inter'), regardless of options — a CDN call
// this self-hosted-only site must not make. Strip it; the site's own font
// (applied via CSS above) replaces it visually anyway.
function stripGoogleFontsImport(svg: string): string {
  return svg.replace(/@import\s+url\('https:\/\/fonts\.googleapis\.com\/[^']*'\);?\n?/g, '');
}

// Diagram-type routing: beautiful-mermaid supports 6 types (flowchart,
// state, sequence, class, ER, xychart) detected from the first non-empty,
// non-comment line's leading keyword. Everything else (gantt, pie, gitGraph,
// mindmap, timeline, journey, sankey, C4, requirement, quadrantChart, …)
// falls through to mermaid.js.
const BEAUTIFUL_KEYWORDS = new Set([
  'graph', 'flowchart',
  'statediagram', 'statediagram-v2',
  'sequencediagram',
  'classdiagram',
  'erdiagram',
  'xychart-beta',
]);

function firstKeyword(src: string): string {
  for (const raw of src.split('\n')) {
    const line = raw.trim();
    if (!line || line.startsWith('%%')) continue;
    const m = line.match(/^[A-Za-z][A-Za-z0-9_-]*/);
    return m ? m[0].toLowerCase() : '';
  }
  return '';
}

function usesBeautifulMermaid(src: string): boolean {
  return BEAUTIFUL_KEYWORDS.has(firstKeyword(src));
}

function loadScript(id: string, globalName: string, src: string, cb: () => void) {
  const w = window as any;
  if (w[globalName]) { cb(); return; }
  let s = document.getElementById(id) as HTMLScriptElement | null;
  if (s) { s.addEventListener('load', cb); return; }
  s = document.createElement('script');
  s.id = id;
  s.src = src;
  s.onload = cb;
  document.head.appendChild(s);
}

// Re-render when the theme switcher changes the theme. $trip2g_theme (in the
// core mol bundle) invokes every registered listener after it flips the theme.
function onThemeChange(cb: () => void) {
  const w = window as any;
  (w.trip2g_theme_listeners ||= []).push(cb);
}

function renderAll(blocks: Block[], done: () => void) {
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
  Promise.resolve(mermaid.run({ nodes: blocks.map((b) => b.el) })).then(done);
}

function runMermaidJsPath(mmBlocks: Block[]) {
  injectStyle(LABEL_STYLE_ID, LABEL_CSS);
  loadScript('mermaid-lib', 'mermaid', '/assets/mermaid.min.js', () => {
    renderAll(mmBlocks, () => enhanceMermaidDiagrams());
    onThemeChange(() => renderAll(mmBlocks, () => enhanceMermaidDiagrams()));
  });
}

function runBeautifulPath(bmBlocks: Block[], onDone: (fallback: Block[]) => void) {
  loadScript('beautiful-mermaid-lib', 'beautifulMermaid', '/assets/beautifulmermaid.min.js', () => {
    const bm = (window as any).beautifulMermaid;
    const opts = beautifulThemeOptions();
    const fallback: Block[] = [];
    for (const b of bmBlocks) {
      try {
        b.el.innerHTML = stripGoogleFontsImport(bm.renderMermaidSVG(b.src, opts));
        // Mirrors what mermaid.run() does to a node once rendered — this is
        // what enhanceMermaidDiagrams() and our own visibility CSS key off.
        b.el.setAttribute('data-processed', 'true');
      } catch (err) {
        fallback.push(b);
      }
    }
    onDone(fallback);
  });
}

function initMermaid() {
  const blocks = collectBlocks();
  if (blocks.length === 0) return;

  injectStyle(ENH_STYLE_ID, ENH_CSS);

  const bmBlocks: Block[] = [];
  let mmBlocks: Block[] = [];
  for (const b of blocks) {
    (usesBeautifulMermaid(b.src) ? bmBlocks : mmBlocks).push(b);
  }

  if (bmBlocks.length === 0) {
    runMermaidJsPath(mmBlocks);
    return;
  }

  runBeautifulPath(bmBlocks, (fallback) => {
    mmBlocks = mmBlocks.concat(fallback);
    if (mmBlocks.length > 0) {
      runMermaidJsPath(mmBlocks);
    } else {
      enhanceMermaidDiagrams();
    }
  });
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initMermaid);
} else {
  initMermaid();
}
