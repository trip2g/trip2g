package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractMCPName(t *testing.T) {
	tests := []struct {
		name    string
		rawMeta map[string]interface{}
		want    string
	}{
		{
			name:    "mcp_name names the server",
			rawMeta: map[string]interface{}{"mcp_method": "initialize", "mcp_name": "Acme Docs"},
			want:    "Acme Docs",
		},
		{
			name:    "title is not a server name",
			rawMeta: map[string]interface{}{"mcp_method": "initialize", "title": "Acme documentation home"},
		},
		{
			name:    "a non-string mcp_name is ignored",
			rawMeta: map[string]interface{}{"mcp_method": "initialize", "mcp_name": 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &NoteView{RawMeta: tt.rawMeta}

			err := note.ExtractMetaData()

			require.NoError(t, err)
			require.Equal(t, tt.want, note.MCPName)
		})
	}
}
