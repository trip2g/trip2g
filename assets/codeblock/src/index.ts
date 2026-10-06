// codeblock widget — syntax highlighting + copy-to-clipboard for fenced code
// blocks rendered by goldmark as <pre><code class="language-XXX">…</code></pre>.
//
// highlight.js is loaded lazily (only when a page actually has a fenced code
// block) from /assets/highlight.min.js, mirroring the mermaid/echarts split.
// The backend only emits this glue on pages that contain a fenced code block.
//
// Copy button: each <pre> is wrapped in a div.code-block that holds the
// button, so it stays put while the code scrolls horizontally. On click it
// copies the raw text (decoded from HTML entities via textContent) and shows
// a brief "Copied" state, also announced through an aria-live region.

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

const LABELS = {
  ru: { copy: 'Копировать код', copied: 'Скопировано' },
  en: { copy: 'Copy code', copied: 'Copied' },
};

const COPY_ICON =
  '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" ' +
  'stroke="currentColor" stroke-width="2" stroke-linecap="round" ' +
  'stroke-linejoin="round" aria-hidden="true">' +
  '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>' +
  '<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>' +
  '</svg>';

const CHECK_ICON =
  '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" ' +
  'stroke="currentColor" stroke-width="2" stroke-linecap="round" ' +
  'stroke-linejoin="round" aria-hidden="true">' +
  '<polyline points="20 6 9 17 4 12"/>' +
  '</svg>';

function labels() {
  return document.documentElement.lang.toLowerCase().startsWith('ru') ? LABELS.ru : LABELS.en;
}

function copyText(text: string): Promise<void> {
  return navigator.clipboard.writeText(text).catch(() => {
    // Fallback for browsers without clipboard API (e.g. HTTP iframes).
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0';
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  });
}

function addCopyButton(pre: HTMLPreElement) {
  const code = pre.querySelector<HTMLElement>('code');
  if (!code) return;
  const l = labels();

  const wrap = document.createElement('div');
  wrap.className = 'code-block';
  pre.parentNode!.insertBefore(wrap, pre);
  wrap.appendChild(pre);

  const btn = document.createElement('button');
  btn.className = 'code-copy-btn';
  btn.type = 'button';
  btn.setAttribute('aria-label', l.copy);
  btn.title = l.copy;
  btn.innerHTML = COPY_ICON;

  const status = document.createElement('span');
  status.className = 'code-copy-status';
  status.setAttribute('aria-live', 'polite');

  let resetTimer: ReturnType<typeof setTimeout> | undefined;

  function showCopied() {
    btn.classList.add('code-copy-btn--copied');
    btn.innerHTML = CHECK_ICON + '<span class="code-copy-btn__text">' + l.copied + '</span>';
    status.textContent = l.copied;
    clearTimeout(resetTimer);
    resetTimer = setTimeout(() => {
      btn.classList.remove('code-copy-btn--copied');
      btn.innerHTML = COPY_ICON;
      status.textContent = '';
    }, 1500);
  }

  btn.addEventListener('click', () => {
    // textContent decodes HTML entities (e.g. &amp; → &) — we want the raw source.
    copyText(code.textContent ?? '').then(showCopied);
  });

  wrap.appendChild(btn);
  wrap.appendChild(status);
}

// Mermaid and datachart blocks are handled by their own widgets, which
// replace the <pre> or read it as a sibling, so leave them untouched.
function isWidgetBlock(code: Element): boolean {
  return code.classList.contains('language-mermaid') ||
    code.classList.contains('language-datachart');
}

function applyHighlighting() {
  const hljs = (window as any).hljs;
  const codes = Array.from(
    document.querySelectorAll<HTMLElement>('pre > code'),
  );
  for (const code of codes) {
    if (isWidgetBlock(code)) continue;

    // highlight.js auto-detects if no language class is present. A language
    // it doesn't know would only produce a console warning, so skip it.
    const lang = Array.from(code.classList).find((c) => c.startsWith('language-'));
    if (lang && !hljs.getLanguage(lang.slice('language-'.length))) {
      code.classList.add('nohighlight');
      continue;
    }
    hljs.highlightElement(code);
  }
}

function initCodeblocks() {
  const pres = Array.from(document.querySelectorAll<HTMLPreElement>('pre'));
  // Filter to code-containing blocks only (skip raw <pre> without <code>).
  const codePres = pres.filter((pre) => {
    const code = pre.querySelector('code');
    return code && !isWidgetBlock(code);
  });
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
