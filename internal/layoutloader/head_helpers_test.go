package layoutloader

import (
	"bytes"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestHeadHelpersRender(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"abs_url joins base and root path", `{{ abs_url("https://trip2g.com", "/") }}`, "https://trip2g.com/"},
		{"abs_url trims base slash", `{{ abs_url("https://trip2g.com/", "/ru") }}`, "https://trip2g.com/ru"},
		{"abs_url keeps absolute path", `{{ abs_url("https://trip2g.com", "https://cdn.example/og.png") }}`, "https://cdn.example/og.png"},
		{"abs_url empty base stays relative", `{{ abs_url("", "/ru") }}`, "/ru"},
		{"json_str quotes and escapes", `{{ json_str("a \"b\" <c> & d's") | unsafe }}`, `"a \"b\" \u003cc\u003e \u0026 d's"`},
		{"json_str bool", `{{ json_str(true) | unsafe }}`, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := []model.LayoutSourceFile{{
				ID:      "/page",
				Path:    "_layouts/page.html",
				Content: tt.content,
			}}

			layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
			require.NoError(t, err)

			var buf bytes.Buffer
			err = layouts.Map["/page"].View.Execute(&buf, nil, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, buf.String())
		})
	}
}
