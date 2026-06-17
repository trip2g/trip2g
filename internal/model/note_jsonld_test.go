package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractJSONLDFields(t *testing.T) {
	tests := []struct {
		name        string
		meta        map[string]interface{}
		wantAuthor  string
		wantTags    []string
		wantUpdated string // RFC3339 date portion, or "" for zero
	}{
		{
			name:       "empty",
			meta:       map[string]interface{}{},
			wantAuthor: "",
			wantTags:   nil,
		},
		{
			name:       "author trimmed",
			meta:       map[string]interface{}{"author": "  Jane Doe  "},
			wantAuthor: "Jane Doe",
		},
		{
			name:     "tags as list",
			meta:     map[string]interface{}{"tags": []interface{}{"go", " web ", ""}},
			wantTags: []string{"go", "web"},
		},
		{
			name:     "tags as comma string",
			meta:     map[string]interface{}{"tags": "go, web , seo"},
			wantTags: []string{"go", "web", "seo"},
		},
		{
			name:     "keywords fallback when no tags",
			meta:     map[string]interface{}{"keywords": []interface{}{"a", "b"}},
			wantTags: []string{"a", "b"},
		},
		{
			name:     "tags win over keywords",
			meta:     map[string]interface{}{"tags": []interface{}{"x"}, "keywords": []interface{}{"y"}},
			wantTags: []string{"x"},
		},
		{
			name:        "updated date",
			meta:        map[string]interface{}{"updated": "2026-06-17"},
			wantUpdated: "2026-06-17",
		},
		{
			name:        "modified alias",
			meta:        map[string]interface{}{"modified": "2026-01-02"},
			wantUpdated: "2026-01-02",
		},
		{
			name:        "invalid date ignored",
			meta:        map[string]interface{}{"updated": "not-a-date"},
			wantUpdated: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{RawMeta: tt.meta}
			n.extractJSONLDFields()
			require.Equal(t, tt.wantAuthor, n.Author)
			require.Equal(t, tt.wantTags, n.Tags)
			if tt.wantUpdated == "" {
				require.True(t, n.UpdatedAt.IsZero())
			} else {
				require.Equal(t, tt.wantUpdated, n.UpdatedAt.Format("2006-01-02"))
			}
		})
	}
}
