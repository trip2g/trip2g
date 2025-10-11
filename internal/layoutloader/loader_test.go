package layoutloader

import (
	"bytes"
	"testing"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestResolveAssets(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/trip2g/main",
		VersionID: 27,
		Path:      "_layouts/trip2g/main.html",
		Content:   `{{ asset("style.css") }}, {{ asset("main.js") }}`,
		Assets: map[string]*model.NoteAssetReplace{
			"_layouts/trip2g/style.css": &model.NoteAssetReplace{
				URL:  "https://storage/style.css",
				Hash: "abc123",
			},
			"_layouts/trip2g/main.js": &model.NoteAssetReplace{
				URL:  "https://storage/main.js",
				Hash: "def456",
			},
		},
	}}

	options := Options{}

	layouts, err := Load(sources, options)
	require.NoError(t, err)
	require.Len(t, layouts.Map, 1)

	var buf bytes.Buffer

	err = layouts.Map["/trip2g/main"].View.Execute(&buf, nil, nil)
	require.NoError(t, err)

	require.Equal(t, `https://storage/style.css, https://storage/main.js`, buf.String())
}
