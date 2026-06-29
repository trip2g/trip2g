package main

import (
	"context"
	"fmt"
	"os"

	"trip2g/internal/doclint"
	"trip2g/internal/logger"
)

// runLint is the entry-point for the "lint" subcommand.
// It is called before appconfig / DB init so it never opens SQLite.
func runLint(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: trip2g lint <dir>")
		os.Exit(2)
	}

	dir := args[0]

	// Use a quiet logger so lint output is clean (warnings go to stdout only).
	log := &logger.DummyLogger{}

	code, err := doclint.Run(context.Background(), dir, os.Stdout, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lint: %v\n", err)
		os.Exit(2)
	}

	os.Exit(code)
}
