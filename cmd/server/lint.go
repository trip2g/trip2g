package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"trip2g/internal/doclint"
	"trip2g/internal/logger"
)

// runLint is the entry-point for the "lint" subcommand.
// It is called before appconfig / DB init so it never opens SQLite.
//
// Usage:
//
//	trip2g lint [--baseline <file>] [--generate-baseline <file>] <dir>
//
// Flags:
//
//	--baseline <file>            Baseline-ratchet file; grandfathered warnings are
//	                             suppressed and do not cause exit 1.
//	--generate-baseline <file>   Write the current warnings as a new baseline and exit 0.
func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	baseline := fs.String("baseline", "", "baseline-ratchet file (stable signatures; pre-existing warnings are suppressed)")
	generateBaseline := fs.String("generate-baseline", "", "write current warnings as a new baseline to this file and exit 0")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "usage: trip2g lint [--baseline <file>] [--generate-baseline <file>] <dir>")
		os.Exit(2)
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "usage: trip2g lint [--baseline <file>] [--generate-baseline <file>] <dir>")
		os.Exit(2)
	}

	dir := remaining[0]

	// Use a quiet logger so lint output is clean (warnings go to stdout only).
	log := &logger.DummyLogger{}
	ctx := context.Background()

	if *generateBaseline != "" {
		if err := doclint.GenerateBaseline(ctx, dir, *generateBaseline, log); err != nil {
			fmt.Fprintf(os.Stderr, "lint: generate-baseline: %v\n", err)
			os.Exit(2)
		}
		os.Exit(0)
	}

	code, err := doclint.RunWithOptions(ctx, dir, os.Stdout, log, doclint.Options{
		BaselineFile: *baseline,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "lint: %v\n", err)
		os.Exit(2)
	}

	os.Exit(code)
}
