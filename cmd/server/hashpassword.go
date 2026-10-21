package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// runHashPassword is the entry-point for the "hash-password" subcommand. It
// reads a password from the first line of stdin and prints the bcrypt hash
// OWNER_PASSWORD_HASH takes. It never loads configuration or opens SQLite.
//
// Usage:
//
//	echo -n 'password' | trip2g-server hash-password
func runHashPassword() {
	err := hashPassword(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash-password: %v\n", err)
		os.Exit(1)
	}
}

func hashPassword(in io.Reader, out io.Writer) error {
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to read password from stdin: %w", err)
	}

	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		return errors.New("empty password: pass it on stdin")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = fmt.Fprintln(out, string(hash))
	return err
}
