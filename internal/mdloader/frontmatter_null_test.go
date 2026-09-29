package mdloader_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
)

// A YAML key with no value (or null / ~) is a common way to leave a field
// blank; it must read as "not set" instead of failing the whole load.
func TestNullFrontmatterScalarIsUnset(t *testing.T) {
	tests := []struct {
		name  string
		front string
	}{
		{name: "empty description", front: "description:\n"},
		{name: "null description", front: "description: null\n"},
		{name: "tilde description", front: "description: ~\n"},
		{name: "empty redirect", front: "redirect:\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.TestLogger{}
			pages, err := mdloader.Load(mdloader.Options{
				Sources: []mdloader.SourceFile{
					{Path: "blank.md", Content: []byte("---\ntitle: Blank\n" + tt.front + "---\nBody.")},
					{Path: "other.md", Content: []byte("Other note.")},
				},
				Log: &log,
			})
			require.NoError(t, err)
			note := pages.PathMap["blank.md"]
			require.NotNil(t, note)
			require.Nil(t, note.Description)
			require.Nil(t, note.Redirect)
			require.NotNil(t, pages.PathMap["other.md"], "one blank field must not drop the rest of the vault")
		})
	}
}

func TestNonStringDescriptionStillFails(t *testing.T) {
	log := logger.TestLogger{}
	_, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{
			{Path: "bad.md", Content: []byte("---\ndescription: [a, b]\n---\nBody.")},
		},
		Log: &log,
	})
	require.Error(t, err)
}
