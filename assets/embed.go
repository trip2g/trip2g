//go:build !dev
// +build !dev

package assets

import "embed"

//go:embed defaulttemplate.css defaulttemplate.js toc/toc.js chart.js echarts.min.js mermaid.js mermaid.min.js beautifulmermaid.min.js codeblock.js highlight.min.js ui/admin/-/web.js ui/user/-/web.js ui/forms/-/web.js ui/forms/-/web.locale=*.json ui/user/-/web.locale=*.json ui/editor/pane/-/web.js ui/editor/pane/-/web.locale=*.json ui/admin/-/web.locale=*.json *.png *.ico *.svg *.webmanifest langs/*.png
var FS embed.FS
