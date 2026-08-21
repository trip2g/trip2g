package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"trip2g/cmd/codellm/internal/codellm"
)

// runSeal implements `codellm seal [--env-key NAME]`: it reads the plaintext
// from stdin and prints the value to paste into a role note's frontmatter.
//
// The secret arrives on stdin rather than as a flag on purpose — an argv secret
// is visible in the process table and lands in shell history. The key is named,
// not passed: it is resolved from codellm's own environment, the same place the
// running service resolves it from, so sealing and unsealing cannot disagree
// about which key was used.
func runSeal(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("seal", flag.ContinueOnError)
	envKey := fs.String("env-key", codellm.DefaultSealEnvKey,
		"name of the env var holding the 32-byte seal key")
	if err := fs.Parse(args); err != nil {
		return err
	}

	key := os.Getenv(*envKey)
	if key == "" {
		return fmt.Errorf("%s is not set", *envKey)
	}

	plaintext, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	// A trailing newline is what a shell heredoc or `echo` adds, never part of
	// the credential; a secret that really ends in one can be piped with printf.
	blob, err := codellm.Seal(key, strings.TrimRight(string(plaintext), "\n"))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, blob)
	return err
}
