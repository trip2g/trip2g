package defaulttemplate

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsupportedFile_Canvas(t *testing.T) {
	ctx := &Ctx{
		Title:              "demo.canvas",
		UnsupportedFileExt: ".canvas",
	}
	var buf bytes.Buffer
	WriteRender(&buf, ctx)
	body := buf.String()
	require.Contains(t, body, "Canvas files are not supported yet.")
	require.NotContains(t, body, "Bases are not supported yet.")
}

func TestUnsupportedFile_Base(t *testing.T) {
	ctx := &Ctx{
		Title:              "example_base.base",
		UnsupportedFileExt: ".base",
	}
	var buf bytes.Buffer
	WriteRender(&buf, ctx)
	body := buf.String()
	require.Contains(t, body, "Bases are not supported yet.")
	require.NotContains(t, body, "Canvas files are not supported yet.")
}

func TestUnsupportedFile_Excalidraw(t *testing.T) {
	ctx := &Ctx{
		Title:              "example.excalidraw",
		UnsupportedFileExt: ".excalidraw",
	}
	var buf bytes.Buffer
	WriteRender(&buf, ctx)
	body := buf.String()
	require.Contains(t, body, "Excalidraw files are not supported yet.")
	require.NotContains(t, body, "Canvas files are not supported yet.")
	require.NotContains(t, body, "Bases are not supported yet.")
}
