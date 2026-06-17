package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOGImageURL(t *testing.T) {
	tests := []struct {
		name   string
		meta   map[string]interface{}
		assets map[string]*NoteAssetReplace
		links  map[string]string
		want   string
	}{
		{
			name: "no key",
			meta: map[string]interface{}{},
			want: "",
		},
		{
			name:   "wikilink resolved directly",
			meta:   map[string]interface{}{"og_image": "[[cover.png]]"},
			assets: map[string]*NoteAssetReplace{"cover.png": {URL: "https://cdn/cover.png"}},
			want:   "https://cdn/cover.png",
		},
		{
			name:   "plain path resolved directly",
			meta:   map[string]interface{}{"og_image": "images/cover.jpg"},
			assets: map[string]*NoteAssetReplace{"images/cover.jpg": {URL: "https://cdn/c.jpg"}},
			want:   "https://cdn/c.jpg",
		},
		{
			name:   "resolved via ResolvedLinks",
			meta:   map[string]interface{}{"og_image": "[[Cover]]"},
			links:  map[string]string{"Cover": "assets/cover.png"},
			assets: map[string]*NoteAssetReplace{"assets/cover.png": {URL: "https://cdn/x.png"}},
			want:   "https://cdn/x.png",
		},
		{
			name:   "alias stripped",
			meta:   map[string]interface{}{"og_image": "[[cover.png|My Cover]]"},
			assets: map[string]*NoteAssetReplace{"cover.png": {URL: "https://cdn/cover.png"}},
			want:   "https://cdn/cover.png",
		},
		{
			name:   "cover fallback key",
			meta:   map[string]interface{}{"cover": "[[c.png]]"},
			assets: map[string]*NoteAssetReplace{"c.png": {URL: "https://cdn/c.png"}},
			want:   "https://cdn/c.png",
		},
		{
			name: "unresolved",
			meta: map[string]interface{}{"og_image": "[[missing.png]]"},
			want: "",
		},
		{
			name:   "list first item",
			meta:   map[string]interface{}{"og_image": []interface{}{"[[a.png]]", "[[b.png]]"}},
			assets: map[string]*NoteAssetReplace{"a.png": {URL: "https://cdn/a.png"}},
			want:   "https://cdn/a.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{
				RawMeta:       tt.meta,
				AssetReplaces: tt.assets,
				ResolvedLinks: tt.links,
			}
			require.Equal(t, tt.want, n.OGImageURL())
		})
	}
}
