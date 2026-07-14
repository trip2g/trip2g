// Standalone beautiful-mermaid bundle. beautiful-mermaid ships as ESM; this
// wraps its SVG renderer in an IIFE that exposes window.beautifulMermaid,
// mirroring lib.ts's window.mermaid global so the glue (index.ts) can
// lazy-load it via a plain <script> tag.
//
// Heavy (pulls in ELK.js for flowchart layout, ~1.6 MB minified), so it is
// its own lazy chunk — same pattern as mermaid.min.js — loaded only for
// pages with a diagram type beautiful-mermaid supports.
import { renderMermaidSVG } from 'beautiful-mermaid';

(window as any).beautifulMermaid = { renderMermaidSVG };
