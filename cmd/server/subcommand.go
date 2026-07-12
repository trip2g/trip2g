package main

import "strings"

// knownSubcommands are bare-word os.Args[1] values dispatched to a subcommand
// handler instead of booting the server.
var knownSubcommands = map[string]bool{ //nolint:gochecknoglobals // small static dispatch table
	"lint":       true,
	"login-link": true,
}

// unrecognizedSubcommand reports the bare subcommand name from an os.Args-shaped
// slice when it must block the server from booting: present, not a flag (doesn't
// start with "-"), and not one of knownSubcommands. Without this check a typo'd
// or unsupported subcommand (e.g. an older binary that doesn't know a newer one)
// would silently fall through to starting a second server process against the
// live SQLite DB next to an already-running instance.
func unrecognizedSubcommand(args []string) (string, bool) {
	if len(args) < 2 {
		return "", false
	}

	arg := args[1]
	if strings.HasPrefix(arg, "-") || knownSubcommands[arg] {
		return "", false
	}

	return arg, true
}
