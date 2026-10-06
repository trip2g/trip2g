//go:build !dev

package assets

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fasthttp serves a *.fasthttp.gz sibling instead of the file itself, and embedded
// files carry a zero mtime, so a stale one left in the working tree would win forever.
func TestFSHasNoPrecompressedFiles(t *testing.T) {
	err := fs.WalkDir(FS, ".", func(path string, _ fs.DirEntry, err error) error {
		require.NoError(t, err)
		require.False(t, strings.HasSuffix(path, ".fasthttp.gz") || strings.HasSuffix(path, ".fasthttp.br"), path)
		return nil
	})
	require.NoError(t, err)
}
