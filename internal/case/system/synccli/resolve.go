// Package synccli serves the sync CLI bundled in the onboarding vault, so an
// agent pod can fetch the CLI matching its server instead of a frozen copy.
package synccli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg synccli_test . Env

type Env interface {
	// OnboardingVaultZip returns the vault template archive, empty when the
	// vault was not built into this binary.
	OnboardingVaultZip() []byte
}

type Script struct {
	Body []byte
	ETag string
}

// entryPattern matches by plugin layout, not folder name, so renaming the
// plugin folder needs no change here.
const entryPattern = "*/.obsidian/plugins/*/trip2g-sync.mjs"

const maxScriptSize = 10 * 1024 * 1024

// Resolve returns nil when the archive or the CLI inside it is missing.
func Resolve(env Env) (*Script, error) {
	zipData := env.OnboardingVaultZip()
	if len(zipData) == 0 {
		return nil, nil
	}

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read onboarding vault zip: %w", err)
	}

	for _, file := range reader.File {
		matched, _ := path.Match(entryPattern, file.Name)
		if !matched {
			continue
		}

		body, readErr := readFile(file)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file.Name, readErr)
		}

		sum := sha256.Sum256(body)

		return &Script{Body: body, ETag: `"` + hex.EncodeToString(sum[:]) + `"`}, nil
	}

	return nil, nil
}

func readFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(io.LimitReader(rc, maxScriptSize))
}
