package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPerparePermalink(t *testing.T) {
	n := NoteView{Path: "/Моя заметка + другая заметка.md"}
	n.PreparePermalink()

	require.Equal(t, n.Permalink, "/моя_заметка_другая_заметка")

	n.Path = "Моя особая + страница"
	n.PreparePermalink()

	require.Equal(t, n.Permalink, "/моя_особая_страница")
}
