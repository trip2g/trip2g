// codeblock widget — syntax highlighting + copy-to-clipboard for fenced code
// blocks rendered by goldmark as <pre><code class="language-XXX">…</code></pre>.
//
// highlight.js is loaded lazily (only when a page actually has a fenced code
// block) from /assets/highlight.min.js, mirroring the mermaid/echarts split.
// The backend only emits this glue on pages that contain a fenced code block.
//
// Copy button: appended to every <pre> container. On click it copies the raw
// text (decoded from HTML entities via textContent) and shows a brief "Copied"
// state with aria feedback.

function loadHLJS(cb: () => void) {
  if ((window as any).hljs) { cb(); return; }
  let s = document.getElementById('hljs-lib') as HTMLScriptElement | null;
  if (s) { s.addEventListener('load', cb); return; }
  s = document.createElement('script');
  s.id = 'hljs-lib';
  s.src = '/assets/highlight.min.js';
  s.onload = cb;
  document.head.appendChild(s);
}

function addCopyButton(pre: HTMLPreElement) {
  const code = pre.querySelector<HTMLElement>('code');
  if (!code) return;

  const btn = document.createElement('button');
  btn.className = 'code-copy-btn';
  btn.setAttribute('aria-label', 'Copy code to clipboard');
  btn.setAttribute('type', 'button');
  btn.innerHTML =
    '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" ' +
    'stroke="currentColor" stroke-width="2" stroke-linecap="round" ' +
    'stroke-linejoin="round" aria-hidden="true">' +
    '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>' +
    '<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>' +
    '</svg>';

  let resetTimer: ReturnType<typeof setTimeout> | undefined;

  btn.addEventListener('click', () => {
    // textContent decodes HTML entities (e.g. &amp; → &) — we want the raw source.
    const text = code.textContent ?? '';
    navigator.clipboard.writeText(text).then(() => {
      btn.classList.add('code-copy-btn--copied');
      btn.setAttribute('aria-label', 'Copied!');
      clearTimeout(resetTimer);
      resetTimer = setTimeout(() => {
        btn.classList.remove('code-copy-btn--copied');
        btn.setAttribute('aria-label', 'Copy code to clipboard');
      }, 2000);
    }).catch(() => {
      // Fallback for browsers without clipboard API (e.g. HTTP iframes).
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
      btn.classList.add('code-copy-btn--copied');
      btn.setAttribute('aria-label', 'Copied!');
      clearTimeout(resetTimer);
      resetTimer = setTimeout(() => {
        btn.classList.remove('code-copy-btn--copied');
        btn.setAttribute('aria-label', 'Copy code to clipboard');
      }, 2000);
    });
  });

  // Wrap pre in a relative container so the button can be positioned inside.
  pre.style.position = 'relative';
  pre.appendChild(btn);
}

function applyHighlighting() {
  const hljs = (window as any).hljs;
  const codes = Array.from(
    document.querySelectorAll<HTMLElement>('pre > code'),
  );
  for (const code of codes) {
    // Skip mermaid and datachart blocks — they are handled by their own widgets.
    if (
      code.classList.contains('language-mermaid') ||
      code.classList.contains('language-datachart')
    ) continue;

    // highlight.js auto-detects if no language class is present; if a
    // `language-XXX` class is present it uses that as a hint.
    hljs.highlightElement(code);
  }
}

function initCodeblocks() {
  const pres = Array.from(document.querySelectorAll<HTMLPreElement>('pre'));
  // Filter to code-containing blocks only (skip raw <pre> without <code>).
  const codePres = pres.filter((pre) => pre.querySelector('code'));
  if (codePres.length === 0) return;

  // Add copy buttons immediately — no dependency on hljs.
  for (const pre of codePres) {
    addCopyButton(pre);
  }

  // Then load hljs and apply highlighting.
  loadHLJS(applyHighlighting);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initCodeblocks);
} else {
  initCodeblocks();
}
