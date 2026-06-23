// highlight.js library bundle — wraps highlight.js and exposes it as
// window.hljs, mirroring how mermaid.min.js sets window.mermaid.
// Loaded lazily by codeblock.js only on pages that have code blocks.

import hljs from 'highlight.js/lib/common';

(window as any).hljs = hljs;
