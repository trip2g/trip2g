package main

import (
	"github.com/gotd/td/tg"

	"trip2g/internal/tgtd"
)

// Convert wraps tgtd.Convert for local usage
func Convert(msg *tg.Message) string {
	return tgtd.Convert(msg)
}
