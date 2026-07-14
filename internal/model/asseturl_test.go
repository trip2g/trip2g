package model_test

import (
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestAbsoluteURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{"relative asset path", "https://example.com", "/_system/assets/abc/pic.png", "https://example.com/_system/assets/abc/pic.png"},
		{"base with trailing slash", "https://example.com/", "/_system/assets/abc/pic.png", "https://example.com/_system/assets/abc/pic.png"},
		{"already absolute passes through", "https://example.com", "https://cdn.other.com/x.png", "https://cdn.other.com/x.png"},
		{"empty path passes through", "https://example.com", "", ""},
		{"relative path without leading slash", "https://example.com", "assets/x.png", "https://example.com/assets/x.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, model.AbsoluteURL(tt.baseURL, tt.path))
		})
	}
}

func TestNoteAssetURLPath_IsRelative(t *testing.T) {
	// In-page asset URLs must stay relative (cacheable, host-agnostic).
	// External consumers absolutize with AbsoluteURL at the point of use.
	got := model.NoteAssetURLPath("deadbeef", "pic.png")
	require.Equal(t, "/_system/assets/deadbeef/pic.png", got)
	require.NotContains(t, got, "://")
}
