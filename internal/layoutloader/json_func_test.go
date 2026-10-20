package layoutloader

import (
	"bytes"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

func yamlMetaNote() *templateviews.Note {
	return templateviews.NewNote(&model.NoteView{
		Title: "Talk",
		RawMeta: map[string]interface{}{
			"title": "Talk",
			"speakers": map[interface{}]interface{}{
				"host": map[interface{}]interface{}{"name": "Ann"},
			},
			"slots": []interface{}{
				map[interface{}]interface{}{"at": "10:00"},
			},
			"rooms": map[interface{}]interface{}{1: "Hall", true: "Yard"},
		},
	})
}

func TestJSONEncodesYAMLMaps(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "nested map",
			content: `{{ json(note.M().Get("speakers")) }}`,
			want:    `{"host":{"name":"Ann"}}`,
		},
		{
			name:    "list of maps",
			content: `{{ json(note.M().Get("slots")) }}`,
			want:    `[{"at":"10:00"}]`,
		},
		{
			name:    "non-string keys",
			content: `{{ json(note.M().Get("rooms")) }}`,
			want:    `{"1":"Hall","true":"Yard"}`,
		},
		{
			name:    "whole frontmatter",
			content: `{{ json(note.M().Raw()) }}`,
			want: `{"rooms":{"1":"Hall","true":"Yard"},` +
				`"slots":[{"at":"10:00"}],` +
				`"speakers":{"host":{"name":"Ann"}},"title":"Talk"}`,
		},
		{
			name:    "writeJson",
			content: `{{ writeJson(note.M().Get("speakers")) }}`,
			want:    "{\"host\":{\"name\":\"Ann\"}}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := jet.VarMap{}
			vars.Set("note", yamlMetaNote())
			got := renderEscapeLayout(t, tt.content, nil, vars)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestJSONPanicNamesUnencodableType(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:      "/page",
		Path:    "_layouts/page.html",
		Content: `{{ json(f) }}`,
	}}
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	vars := jet.VarMap{}
	vars.Set("f", func() {})

	var buf bytes.Buffer
	err = layouts.Map["/page"].View.Execute(&buf, vars, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot encode func()")
}
