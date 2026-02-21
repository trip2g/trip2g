//go:build !dev
// +build !dev

package assets

import "embed"

//go:embed output.css tiptap/tiptap.js ui/admin/-/web.js ui/user/-/web.js ui/user/-/web.locale* ui/user/space/-/web.locale* ui/admin/-/web.locale* *.png *.ico *.svg *.webmanifest
var FS embed.FS
