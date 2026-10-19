package doclint

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var updateLandingHead = flag.Bool("update-landing-head", false, "rewrite testdata/landing_head_*.txt") //nolint:gochecknoglobals // test flag

// headTags keeps the <head> lines a crawler reads: title, meta and link tags,
// and the JSON-LD block. The CSS around them is left out.
func headTags(page string) string {
	head, _, _ := strings.Cut(page, "</head>")
	var out []string
	inLD := false
	for _, line := range strings.Split(head, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, `<script type="application/ld+json">`):
			inLD = true
		case inLD && line == "</script>":
			inLD = false
		case inLD, strings.HasPrefix(line, "<title"), strings.HasPrefix(line, "<meta"), strings.HasPrefix(line, "<link"):
			if line != "" {
				out = append(out, line)
			}
		}
	}
	return strings.Join(out, "\n") + "\n"
}

// TestLandingHead pins what the home pages tell crawlers: canonical, hreflang,
// og:url and the structured data.
func TestLandingHead(t *testing.T) {
	pages := map[string]string{
		"/":          "testdata/landing_head_en.txt",
		"/ru":        "testdata/landing_head_ru.txt",
		"/en/circle": "testdata/landing_head_circle_en.txt",
		"/ru/circle": "testdata/landing_head_circle_ru.txt",
	}
	for permalink, golden := range pages {
		t.Run(permalink, func(t *testing.T) {
			got := headTags(renderDocsLanding(t, permalink))
			if *updateLandingHead {
				require.NoError(t, os.WriteFile(golden, []byte(got), 0o600))
				return
			}
			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			require.Equal(t, string(want), got)
		})
	}
}
