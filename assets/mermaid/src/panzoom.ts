// Pan/zoom for rendered mermaid SVGs. Zero-dep, Pointer Events based (mouse
// drag, touch pinch + one-finger pan, ctrl/⌘+wheel) — svg-pan-zoom needs
// Hammer.js for pinch, so a small custom implementation is lighter.
//
// Default state is untouched (fit-to-width, scale 1); the transform only kicks
// in when the user zooms. At scale 1 the container keeps `touch-action: pan-y`
// so one-finger page scroll is never hijacked; pinch is intercepted to zoom.
// Once zoomed (or in fullscreen) one-finger drag pans the diagram.

const MIN_SCALE = 1;
const MAX_SCALE = 10;
const STYLE_ID = 'mermaid-panzoom-style';

const CSS = `
.mermaid-panzoom { position: relative; overflow: hidden; }
.mermaid-panzoom--zoomed { cursor: grab; }
.mermaid-panzoom--zoomed:active { cursor: grabbing; }
.mermaid-panzoom__controls {
  position: absolute; top: 0.4rem; right: 0.4rem;
  display: flex; gap: 0.3rem; z-index: 5;
  opacity: 0.55; transition: opacity 0.15s;
}
.mermaid-panzoom:hover .mermaid-panzoom__controls,
.mermaid-panzoom--zoomed .mermaid-panzoom__controls,
.mermaid-panzoom--full .mermaid-panzoom__controls { opacity: 1; }
.mermaid-panzoom__btn {
  width: 2.2rem; height: 2.2rem; padding: 0; margin: 0;
  display: flex; align-items: center; justify-content: center;
  font-size: 1.1rem; line-height: 1; color: inherit; cursor: pointer;
  border: 1px solid rgba(128, 128, 128, 0.4); border-radius: 0.3rem;
  background: rgba(128, 128, 128, 0.15);
}
.mermaid-panzoom__btn:hover { background: rgba(128, 128, 128, 0.3); }
.mermaid-panzoom--full {
  position: fixed; inset: 0; z-index: 1000; margin: 0;
  background: var(--pico-background-color, #fff);
}
`;

function injectStyle() {
  if (document.getElementById(STYLE_ID)) return;
  const s = document.createElement('style');
  s.id = STYLE_ID;
  s.textContent = CSS;
  document.head.appendChild(s);
}

interface Point { x: number; y: number }

export function enhancePanZoom(container: HTMLElement) {
  const found = container.querySelector('svg');
  if (!found) return;

  // Theme change re-renders the diagram (textContent reset wipes the controls
  // and replaces the svg) — rebind instead of stacking a second listener set.
  const existing = (container as any).__panzoom as { rebind: () => void } | undefined;
  if (existing) { existing.rebind(); return; }

  injectStyle();
  container.classList.add('mermaid-panzoom');

  let svg: SVGSVGElement = found;
  let scale = 1;
  let tx = 0;
  let ty = 0;
  let full = false;

  const apply = () => {
    svg.style.transformOrigin = '0 0';
    svg.style.transform = `translate(${tx}px, ${ty}px) scale(${scale})`;
    container.classList.toggle('mermaid-panzoom--zoomed', scale > 1);
    // pan-y keeps one-finger page scroll working at fit; pinch still reaches us.
    container.style.touchAction = full || scale > 1 ? 'none' : 'pan-y';
  };

  const reset = () => { scale = 1; tx = 0; ty = 0; apply(); };

  const zoomAt = (clientX: number, clientY: number, factor: number) => {
    const next = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale * factor));
    const rect = container.getBoundingClientRect();
    const px = clientX - rect.left;
    const py = clientY - rect.top;
    tx = px - (px - tx) * (next / scale);
    ty = py - (py - ty) * (next / scale);
    scale = next;
    if (scale === 1) { tx = 0; ty = 0; }
    apply();
  };

  const zoomCenter = (factor: number) => {
    const rect = container.getBoundingClientRect();
    zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, factor);
  };

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') setFull(false);
  };

  const setFull = (on: boolean) => {
    if (full === on) return;
    full = on;
    container.classList.toggle('mermaid-panzoom--full', on);
    document.documentElement.style.overflow = on ? 'hidden' : '';
    if (on) document.addEventListener('keydown', onKeyDown);
    else document.removeEventListener('keydown', onKeyDown);
    reset();
  };

  const buildControls = () => {
    const bar = document.createElement('div');
    bar.className = 'mermaid-panzoom__controls';
    const btn = (label: string, title: string, onClick: () => void) => {
      const b = document.createElement('button');
      b.type = 'button';
      b.className = 'mermaid-panzoom__btn';
      b.textContent = label;
      b.title = title;
      b.setAttribute('aria-label', title);
      b.addEventListener('click', onClick);
      bar.appendChild(b);
    };
    btn('+', 'Zoom in', () => zoomCenter(1.4));
    btn('−', 'Zoom out', () => zoomCenter(1 / 1.4));
    btn('⟲', 'Reset zoom', reset);
    btn('⛶', 'Fullscreen', () => setFull(!full));
    container.appendChild(bar);
  };

  // --- pointer input (mouse drag, one-finger pan, two-finger pinch) ---
  const pointers = new Map<number, Point>();
  let lastMid: Point | null = null;
  let lastDist = 0;

  const pinchState = () => {
    const [a, b] = Array.from(pointers.values());
    return {
      mid: { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 },
      dist: Math.hypot(a.x - b.x, a.y - b.y),
    };
  };

  container.addEventListener('pointerdown', (e) => {
    if ((e.target as HTMLElement).closest('.mermaid-panzoom__controls')) return;
    pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
    container.setPointerCapture(e.pointerId);
    if (pointers.size === 2) {
      const s = pinchState();
      lastMid = s.mid;
      lastDist = s.dist;
    }
    if (e.pointerType === 'mouse' && (scale > 1 || full)) e.preventDefault();
  });

  container.addEventListener('pointermove', (e) => {
    if (!pointers.has(e.pointerId)) return;
    const prev = pointers.get(e.pointerId)!;
    pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (pointers.size === 2) {
      const s = pinchState();
      if (lastDist > 0) zoomAt(s.mid.x, s.mid.y, s.dist / lastDist);
      if (lastMid) { tx += s.mid.x - lastMid.x; ty += s.mid.y - lastMid.y; apply(); }
      lastMid = s.mid;
      lastDist = s.dist;
      e.preventDefault();
    } else if (pointers.size === 1 && (scale > 1 || full)) {
      tx += e.clientX - prev.x;
      ty += e.clientY - prev.y;
      apply();
      e.preventDefault();
    }
  });

  const drop = (e: PointerEvent) => {
    pointers.delete(e.pointerId);
    lastMid = null;
    lastDist = 0;
  };
  container.addEventListener('pointerup', drop);
  container.addEventListener('pointercancel', drop);

  // ctrl/⌘+wheel zooms (also trackpad pinch, which reports ctrlKey); plain
  // wheel keeps scrolling the page — except in fullscreen, where it zooms too.
  container.addEventListener('wheel', (e) => {
    if (!e.ctrlKey && !e.metaKey && !full) return;
    e.preventDefault();
    zoomAt(e.clientX, e.clientY, Math.exp(-e.deltaY * 0.01));
  }, { passive: false });

  const rebind = () => {
    const next = container.querySelector('svg');
    if (!next) return;
    svg = next;
    setFull(false);
    buildControls();
    reset();
  };

  (container as any).__panzoom = { rebind };
  buildControls();
  apply();
}
