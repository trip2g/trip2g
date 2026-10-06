package defaulttemplate

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var docEmbedRe = regexp.MustCompile(`!\[\[([^|\]]+)[^\]]*\]\]`)

func TestDefaultTemplateDocsEmbedLayoutDiagram(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			page := filepath.Join("../../docs", lang, "user/default-template.md")
			data, err := os.ReadFile(page)
			require.NoError(t, err)

			var diagram string
			for _, m := range docEmbedRe.FindAllStringSubmatch(string(data), -1) {
				if strings.HasSuffix(m[1], "default-template-layout.svg") {
					diagram = filepath.Join(filepath.Dir(page), m[1])
				}
			}
			require.NotEmpty(t, diagram, "%s must embed default-template-layout.svg", page)

			f, err := os.Open(diagram)
			require.NoError(t, err)
			defer f.Close()

			dec := xml.NewDecoder(f)
			for {
				_, tokErr := dec.Token()
				if errors.Is(tokErr, io.EOF) {
					break
				}
				require.NoError(t, tokErr, "%s must be well-formed SVG", diagram)
			}
		})
	}
}
