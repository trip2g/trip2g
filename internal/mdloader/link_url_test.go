package mdloader_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func TestWikilinkDangerousSchemes(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"javascript", "[[javascript:alert(1)|click]]"},
		{"uppercase", "[[JavaScript:alert(1)|click]]"},
		{"entity", "[[&#106;avascript:alert(1)|click]]"},
		{"vbscript", "[[vbscript:msgbox|click]]"},
		{"data", "[[data:text/html,hi|click]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := mdloader.Load(mdloader.Options{
				Log:     &logger.TestLogger{},
				Sources: []mdloader.SourceFile{{Path: "a.md", Content: []byte(tt.content)}},
			})
			require.NoError(t, err)

			html := string(notes.PathMap["a.md"].HTML)
			require.Contains(t, html, "click")
			require.NotContains(t, html, `href="javascript`)
			require.NotContains(t, html, `href="JavaScript`)
			require.NotContains(t, html, `href="vbscript`)
			require.NotContains(t, html, `href="data`)
		})
	}
}

func TestWikilinkSafeSchemesKept(t *testing.T) {
	notes, err := mdloader.Load(mdloader.Options{
		Log: &logger.TestLogger{},
		Sources: []mdloader.SourceFile{
			{Path: "a.md", Content: []byte("[[b|note]] [[https://example.com|web]] [[mailto:x@example.com|mail]]")},
			{Path: "b.md", Content: []byte("B")},
		},
	})
	require.NoError(t, err)

	html := string(notes.PathMap["a.md"].HTML)
	require.Contains(t, html, `href="/b"`)
	require.Contains(t, html, `href="https://example.com"`)
	require.Contains(t, html, `href="mailto:x@example.com"`)
}
