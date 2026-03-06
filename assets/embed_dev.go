//go:build dev
// +build dev

package assets

import (
	"io/fs"
	"os"
)

//nolint:gochecknoglobals // it's ok
var FS fs.FS = os.DirFS("./assets")
