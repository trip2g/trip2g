//go:build !dev
// +build !dev

package assets

import "embed"

//go:embed output.css turbo.js ui/admin/-/web.js ui/user/-/web.js
var FS embed.FS
