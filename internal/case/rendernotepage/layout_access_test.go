package rendernotepage_test

import (
	"strings"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

const secretMeta = "zzmetasecretzz"

func closedNoteFiles() map[string]string {
	return map[string]string{
		"home.md":   "---\nfree: true\nlayout: probe\n---\nbody\n",
		"closed.md": "---\nsecret: " + secretMeta + "\n---\n" + secretText + "\n",
	}
}

func probeLayout(body string) []model.LayoutSourceFile {
	return []model.LayoutSourceFile{
		{ID: "/probe", Path: "_layouts/probe.html", Content: "<main>" + body + "</main>"},
	}
}

func requireNoLeak(t *testing.T, page string) {
	t.Helper()

	require.NotContains(t, page, secretText)
	require.NotContains(t, page, secretMeta)
}

func TestLayoutAccess_ForeignNoteFieldsNeedReadAccess(t *testing.T) {
	cases := map[string]string{
		"content string": `{{ n := nvs.ByPath("closed.md") }}{{ if n }}{{ n.ContentString() }}{{ end }}`,
		"meta":           `{{ n := nvs.ByPath("closed.md") }}{{ if n }}{{ n.M().GetString("secret", "") }}{{ end }}`,
		"introduction":   `{{ n := nvs.ByPath("closed.md") }}{{ if n }}{{ n.PartialRenderer().Introduce().ContentHTML | unsafe }}{{ end }}`,
		"permalink":      `{{ n := nvs.ByPermalink("/closed") }}{{ if n }}{{ n.ContentString() }}{{ end }}`,
		"wikilink":       `{{ n := nvs.ByWikilink("closed") }}{{ if n }}{{ n.ContentString() }}{{ end }}`,
		"list":           `{{ range _, n := nvs.List() }}{{ n.ContentString() }}{{ n.M().GetString("secret", "") }}{{ end }}`,
		"query":          `{{ range _, n := nvs.Query().All() }}{{ n.ContentString() }}{{ n.M().GetString("secret", "") }}{{ end }}`,
		"glob":           `{{ range _, n := nvs.ByGlob("*.md").All() }}{{ n.ContentString() }}{{ n.M().GetString("secret", "") }}{{ end }}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			env, viewer := embedEnv(t, closedNoteFiles(), probeLayout(body))

			requireNoLeak(t, pageBody(t, env, "/home", nil))

			*viewer = viewerMember
			member := pageBody(t, env, "/home", memberToken())
			require.True(t, containsAny(member, secretText, secretMeta), "a member who can read the note must get it")
		})
	}
}

func containsAny(s string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(s, part) {
			return true
		}
	}

	return false
}

func langFiles() map[string]string {
	return map[string]string{
		"home.md":    "---\nfree: true\nlayout: probe\nlang: en\nlang_redirect: \"[[ru/home]]\"\n---\nbody\n",
		"ru/home.md": "---\nlang: ru\nlang_redirect: \"[[home]]\"\n---\n" + secretText + "\n",
	}
}

func TestLayoutAccess_LangAlternativeNeedsReadAccess(t *testing.T) {
	cases := map[string]string{
		"single": `{{ alt := note.LangAlternative("ru") }}{{ if alt }}{{ alt.HTMLString() | unsafe }}{{ alt.ContentString() }}{{ end }}`,
		"list":   `{{ range _, alt := note.LangAlternativesList() }}{{ alt.HTMLString() | unsafe }}{{ alt.ContentString() }}{{ end }}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			env, viewer := embedEnv(t, langFiles(), probeLayout(body))

			require.NotContains(t, pageBody(t, env, "/home", nil), secretText)

			*viewer = viewerMember
			require.Contains(t, pageBody(t, env, "/home", memberToken()), secretText)
		})
	}
}
