package onboardingvault

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestZipContents guards the packaging rules of generate.sh: the agent guide
// and the sync CLI must ship, build machinery and credentials must not.
func TestZipContents(t *testing.T) {
	if len(ZipData) == 0 {
		t.Skip("vault.zip not built — run go generate ./onboarding-vault/...")
	}

	reader, err := zip.NewReader(bytes.NewReader(ZipData), int64(len(ZipData)))
	require.NoError(t, err)

	names := make(map[string]bool, len(reader.File))
	for _, file := range reader.File {
		names[file.Name] = true
	}

	required := []string{
		"onboarding-vault/AGENTS.md",
		"onboarding-vault/CLAUDE.md",
		"onboarding-vault/.obsidian/plugins/trip2g/trip2g-sync.mjs",
		"onboarding-vault/.obsidian/plugins/trip2g/main.js",
		"onboarding-vault/.obsidian/community-plugins.json",
	}
	for _, name := range required {
		require.True(t, names[name], "%s must be bundled", name)
	}

	forbidden := []string{
		"onboarding-vault/generate.sh",
		"onboarding-vault/vault.zip",
		// data.json is written per download with real credentials
		"onboarding-vault/.obsidian/plugins/trip2g/data.json",
	}
	for _, name := range forbidden {
		require.False(t, names[name], "%s must not be bundled", name)
	}

	for name := range names {
		require.False(t, strings.HasSuffix(name, ".go"), "build machinery %s must not be bundled", name)
	}
}
