//go:build !dev
// +build !dev

package assets

import "embed"

//go:embed defaulttemplate.css output.css tiptap/tiptap.js toc/toc.js ui/admin/-/web.js ui/user/-/web.js ui/user/-/web.locale* ui/admin/-/web.locale* *.png *.ico *.svg *.webmanifest langs/*.png
var FS embed.FS
