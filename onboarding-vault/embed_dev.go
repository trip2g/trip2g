//go:build dev
// +build dev

package onboardingvault

import "os"

// ZipData is read from disk under the dev build tag instead of embedded, so a
// fresh worktree compiles (for `make lint` and `make air`) without first
// running generate.sh. It stays nil until onboarding-vault/vault.zip exists;
// the download endpoint already treats empty ZipData as "unavailable".
//
//nolint:gochecknoglobals // mirrors the embedded prod var (see embed.go)
var ZipData = readZipFromDisk()

func readZipFromDisk() []byte {
	data, err := os.ReadFile("onboarding-vault/vault.zip")
	if err != nil {
		return nil
	}
	return data
}
