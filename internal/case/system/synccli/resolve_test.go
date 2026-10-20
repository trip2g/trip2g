package synccli_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/system/synccli"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for name, content := range files {
		w, err := writer.Create(name)
		require.NoError(t, err)

		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	return buf.Bytes()
}

func etagOf(body string) string {
	sum := sha256.Sum256([]byte(body))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func TestResolve(t *testing.T) {
	const cli = "#!/usr/bin/env node\nconsole.log('sync')\n"

	tests := []struct {
		name    string
		zip     []byte
		want    *synccli.Script
		wantErr bool
	}{
		{
			name: "current plugin folder",
			zip: buildZip(t, map[string]string{
				"onboarding-vault/AGENTS.md":                                "# agents",
				"onboarding-vault/.obsidian/plugins/trip2g/main.js":         "plugin",
				"onboarding-vault/.obsidian/plugins/trip2g/trip2g-sync.mjs": cli,
			}),
			want: &synccli.Script{Body: []byte(cli), ETag: etagOf(cli)},
		},
		{
			name: "renamed plugin folder",
			zip: buildZip(t, map[string]string{
				"onboarding-vault/.obsidian/plugins/ttgsync/trip2g-sync.mjs": cli,
			}),
			want: &synccli.Script{Body: []byte(cli), ETag: etagOf(cli)},
		},
		{
			name: "same name outside a plugin folder is ignored",
			zip: buildZip(t, map[string]string{
				"onboarding-vault/trip2g-sync.mjs":                       cli,
				"onboarding-vault/.obsidian/plugins/a/b/trip2g-sync.mjs": cli,
			}),
		},
		{
			name: "no cli in the archive",
			zip:  buildZip(t, map[string]string{"onboarding-vault/AGENTS.md": "# agents"}),
		},
		{
			name: "build without vault.zip",
			zip:  nil,
		},
		{
			name:    "corrupt archive",
			zip:     []byte("not a zip"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{OnboardingVaultZipFunc: func() []byte { return tt.zip }}

			got, err := synccli.Resolve(env)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
