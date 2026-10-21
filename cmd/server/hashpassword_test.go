package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
	}{
		{"echo with newline", "correct horse\n"},
		{"echo -n", "correct horse"},
		{"windows line ending", "correct horse\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := hashPassword(strings.NewReader(tt.stdin), &out)
			require.NoError(t, err)

			hash := strings.TrimSpace(out.String())
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse")))
		})
	}
}

func TestHashPassword_RefusesEmpty(t *testing.T) {
	var out bytes.Buffer
	err := hashPassword(strings.NewReader("\n"), &out)
	require.Error(t, err)
	require.Empty(t, out.String())
}
