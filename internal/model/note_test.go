package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPerparePermalink(t *testing.T) {
	n := NoteView{Path: "/Моя заметка + другая заметка.md"}
	n.PreparePermalink()

	require.Equal(t, n.Permalink, "/moya_zametka_drugaya_zametka")

	n.Path = "Моя особая + страница"
	n.PreparePermalink()

	require.Equal(t, n.Permalink, "/moya_osobaya_stranica")
}
