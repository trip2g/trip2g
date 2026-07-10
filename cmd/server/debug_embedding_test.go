package main

import (
	"testing"

	"trip2g/internal/appconfig"
	"trip2g/internal/openai"

	"github.com/stretchr/testify/require"
)

// /debug/embedding is unauthenticated and lets a caller burn embedding
// API/CPU, so it must stay off in production unless explicitly enabled.
func TestDebugEmbeddingEnabled(t *testing.T) {
	client := &openai.Client{}

	tests := []struct {
		name           string
		client         *openai.Client
		devMode        bool
		debugEmbedding bool
		want           bool
	}{
		{"off in production by default", client, false, false, false},
		{"on in dev mode", client, true, false, true},
		{"on with explicit flag", client, false, true, true},
		{"off without vector search even in dev", nil, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{appState: &appState{
				config:       &appconfig.Config{DevMode: tt.devMode, DebugEmbedding: tt.debugEmbedding},
				openaiClient: tt.client,
			}}
			require.Equal(t, tt.want, a.debugEmbeddingEnabled())
		})
	}
}
