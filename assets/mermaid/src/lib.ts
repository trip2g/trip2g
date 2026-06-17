// Standalone mermaid library bundle. mermaid ships as ESM; this wraps it in an
// IIFE that exposes window.mermaid, mirroring the prebuilt echarts.min.js global
// so the glue (index.ts) can lazy-load it via a plain <script> tag.
import mermaid from 'mermaid';

(window as any).mermaid = mermaid;
