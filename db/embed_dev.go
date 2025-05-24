//go:build dev
// +build dev

package db

import "embed"

//go:embed migrations/*.sql
var FS embed.FS
