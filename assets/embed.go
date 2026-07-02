//go:build !dev
// +build !dev

package assets

import "embed"

//go:embed defaulttemplate.css defaulttemplate.js output.css toc/toc.js chart.js echarts.min.js mermaid.js mermaid.min.js codeblock.js highlight.min.js tasklist.js ui/admin/-/web.js ui/user/-/web.js ui/forms/-/web.js ui/forms/-/web.locale* ui/user/-/web.locale* ui/admin/-/web.locale* *.png *.ico *.svg *.webmanifest langs/*.png
var FS embed.FS
