package graph

import _ "embed"

// UIHTML is the self-contained localhost debug page: it fetches ./graph.json
// and renders the dependency graph with Mermaid (loaded from CDN; the page
// falls back to raw JSON when the CDN is unreachable).
//
//go:embed ui.html
var UIHTML []byte
