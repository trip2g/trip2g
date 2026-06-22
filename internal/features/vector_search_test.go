package features

import "testing"

func TestEmbeddingModelMaxInputTokens(t *testing.T) {
	tests := []struct {
		model EmbeddingModel
		want  int
	}{
		{EmbeddingModelSmall, 8192},
		{EmbeddingModelLarge, 8192},
		{EmbeddingModelAda, 8192},
		{EmbeddingModelMultilingualE5Base, 512},
		{EmbeddingModelBGEM3, 8192},
	}
	for _, tt := range tests {
		if got := tt.model.MaxInputTokens(); got != tt.want {
			t.Errorf("%s.MaxInputTokens() = %d, want %d", tt.model, got, tt.want)
		}
	}
}
