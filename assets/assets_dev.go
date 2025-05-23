//go:build dev
// +build dev

package assets

import (
	"io/fs"
	"os"
)

var FS fs.FS = os.DirFS("./assets")
