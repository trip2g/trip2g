package layoutloader

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestDetectPersonalized(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "plain layout is not personalized",
			content: `<h1>{{ note.Title() }}</h1>{{ note.HTML() }}`,
			want:    false,
		},
		{
			name:    "default-template chrome helpers are not personalized",
			content: `{{ defaultTemplate.Header() }}{{ note.HTML() }}{{ defaultTemplate.Footer() }}`,
			want:    false,
		},
		{
			name:    "currentUser.IsAdmin gate is personalized",
			content: `{{ if currentUser.IsAdmin() }}<a>edit</a>{{ end }}{{ note.HTML() }}`,
			want:    true,
		},
		{
			name:    "bare currentUser reference is personalized",
			content: `{{ currentUser }}`,
			want:    true,
		},
		{
			name:    "note.LastEditedByLabel byline is personalized",
			content: `<footer>{{ note.LastEditedByLabel() }}</footer>`,
			want:    true,
		},
		{
			name:    "note.LastEditedBy resolver is personalized",
			content: `{{ note.LastEditedBy().Name }}`,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &testEnv{logger: &logger.TestLogger{}}
			sources := []model.LayoutSourceFile{{
				ID:      "/main",
				Path:    "_layouts/main.html",
				Content: tt.content,
			}}
			layouts, err := Load(env, sources, Options{})
			require.NoError(t, err)
			got := layouts.Map["/main"]
			require.NotNil(t, got.View, "layout must parse: %v", got.Warnings)
			require.Equal(t, tt.want, got.Personalized)
		})
	}
}

// TestDetectPersonalized_ViaImport verifies the detector follows {{ import }}:
// a clean page that imports a sub-template referencing currentUser is still
// flagged personalized.
func TestDetectPersonalized_ViaImport(t *testing.T) {
	env := &testEnv{logger: &logger.TestLogger{}}
	sources := []model.LayoutSourceFile{
		{
			ID:      "/comp",
			Path:    "_layouts/comp.html",
			Content: `{{ block adminBadge() }}{{ if currentUser.IsAdmin() }}admin{{ end }}{{ end }}`,
		},
		{
			ID:      "/page",
			Path:    "_layouts/page.html",
			Content: `{{ import "/comp" }}<h1>{{ note.Title() }}</h1>{{ yield adminBadge() }}`,
		},
	}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	page := layouts.Map["/page"]
	require.NotNil(t, page.View, "page must parse: %v", page.Warnings)
	require.True(t, page.Personalized, "personalization in an imported template must propagate")
}
