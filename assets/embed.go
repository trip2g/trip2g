//go:build !dev
// +build !dev

package assets

import "embed"

//go:embed output.css turbo.js ui/admin/-/web.js ui/user/-/web.js ui/user/-/web.locale* ui/admin/-/web.locale* *.woff *.woff2 *.png
var FS embed.FS
