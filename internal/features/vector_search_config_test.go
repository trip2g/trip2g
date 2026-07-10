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

func TestParseEmbeddingModel(t *testing.T) {
	tests := []struct {
		input     string
		wantModel EmbeddingModel
		wantKnown bool
	}{
		{"text-embedding-3-small", EmbeddingModelSmall, true},
		{"small", EmbeddingModelSmall, true},
		{"text-embedding-3-large", EmbeddingModelLarge, true},
		{"large", EmbeddingModelLarge, true},
		{"text-embedding-ada-002", EmbeddingModelAda, true},
		{"ada", EmbeddingModelAda, true},
		{"multilingual-e5-base", EmbeddingModelMultilingualE5Base, true},
		{"bge-m3", EmbeddingModelBGEM3, true},
		{"intfloat/multilingual-e5-large", EmbeddingModelCustom, false},
		{"custom-model-v1", EmbeddingModelCustom, false},
		{"", EmbeddingModelCustom, false},
	}
	for _, tt := range tests {
		got, known := ParseEmbeddingModel(tt.input)
		if got != tt.wantModel || known != tt.wantKnown {
			t.Errorf("ParseEmbeddingModel(%q) = (%v, %v), want (%v, %v)",
				tt.input, got, known, tt.wantModel, tt.wantKnown)
		}
	}
}

func TestVectorSearchConfigResolved_KnownModelNoOverride(t *testing.T) {
	// Known models with no explicit overrides must produce the same values as
	// today (backward compatible).
	tests := []struct {
		name           string
		cfg            VectorSearchConfig
		wantDims       int
		wantMaxTokens  int
		wantQueryPfx   string
		wantPassagePfx string
		wantModelName  string
	}{
		{
			name:           "small",
			cfg:            VectorSearchConfig{Model: EmbeddingModelSmall, ModelName: "text-embedding-3-small"},
			wantDims:       1536,
			wantMaxTokens:  8192,
			wantQueryPfx:   "",
			wantPassagePfx: "",
			wantModelName:  "text-embedding-3-small",
		},
		{
			name:           "large",
			cfg:            VectorSearchConfig{Model: EmbeddingModelLarge, ModelName: "text-embedding-3-large"},
			wantDims:       3072,
			wantMaxTokens:  8192,
			wantQueryPfx:   "",
			wantPassagePfx: "",
			wantModelName:  "text-embedding-3-large",
		},
		{
			name:           "ada",
			cfg:            VectorSearchConfig{Model: EmbeddingModelAda, ModelName: "text-embedding-ada-002"},
			wantDims:       1536,
			wantMaxTokens:  8192,
			wantQueryPfx:   "",
			wantPassagePfx: "",
			wantModelName:  "text-embedding-ada-002",
		},
		{
			name:           "multilingual-e5-base",
			cfg:            VectorSearchConfig{Model: EmbeddingModelMultilingualE5Base, ModelName: "multilingual-e5-base"},
			wantDims:       768,
			wantMaxTokens:  512,
			wantQueryPfx:   "query: ",
			wantPassagePfx: "passage: ",
			wantModelName:  "multilingual-e5-base",
		},
		{
			name:           "bge-m3",
			cfg:            VectorSearchConfig{Model: EmbeddingModelBGEM3, ModelName: "bge-m3"},
			wantDims:       1024,
			wantMaxTokens:  8192,
			wantQueryPfx:   "",
			wantPassagePfx: "",
			wantModelName:  "bge-m3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ResolvedDimensions(); got != tt.wantDims {
				t.Errorf("ResolvedDimensions() = %d, want %d", got, tt.wantDims)
			}
			if got := tt.cfg.ResolvedMaxInputTokens(); got != tt.wantMaxTokens {
				t.Errorf("ResolvedMaxInputTokens() = %d, want %d", got, tt.wantMaxTokens)
			}
			if got := tt.cfg.ResolvedQueryPrefix(); got != tt.wantQueryPfx {
				t.Errorf("ResolvedQueryPrefix() = %q, want %q", got, tt.wantQueryPfx)
			}
			if got := tt.cfg.ResolvedPassagePrefix(); got != tt.wantPassagePfx {
				t.Errorf("ResolvedPassagePrefix() = %q, want %q", got, tt.wantPassagePfx)
			}
			if got := tt.cfg.ResolvedModelName(); got != tt.wantModelName {
				t.Errorf("ResolvedModelName() = %q, want %q", got, tt.wantModelName)
			}
		})
	}
}

func TestVectorSearchConfigResolved_KnownModelWithOverride(t *testing.T) {
	// A known model name with explicit dimension override must use the override.
	cfg := VectorSearchConfig{
		Model:      EmbeddingModelSmall,
		ModelName:  "text-embedding-3-small",
		Dimensions: 512, // truncated dimension is supported by the API
	}
	if got := cfg.ResolvedDimensions(); got != 512 {
		t.Errorf("ResolvedDimensions() = %d, want 512 (override)", got)
	}
	// Unoverridden fields still come from enum defaults.
	if got := cfg.ResolvedMaxInputTokens(); got != 8192 {
		t.Errorf("ResolvedMaxInputTokens() = %d, want 8192", got)
	}
}

func TestVectorSearchConfigResolved_CustomModelWithDimensions(t *testing.T) {
	// An unknown model name with explicit dimensions must work without error.
	cfg := VectorSearchConfig{
		Model:      EmbeddingModelCustom,
		ModelName:  "intfloat/multilingual-e5-large",
		Dimensions: 1024,
		QueryPfx:   "query: ",
		PassagePfx: "passage: ",
		MaxTokens:  512,
	}
	if got := cfg.ResolvedDimensions(); got != 1024 {
		t.Errorf("ResolvedDimensions() = %d, want 1024", got)
	}
	if got := cfg.ResolvedMaxInputTokens(); got != 512 {
		t.Errorf("ResolvedMaxInputTokens() = %d, want 512", got)
	}
	if got := cfg.ResolvedQueryPrefix(); got != "query: " {
		t.Errorf("ResolvedQueryPrefix() = %q, want \"query: \"", got)
	}
	if got := cfg.ResolvedPassagePrefix(); got != "passage: " {
		t.Errorf("ResolvedPassagePrefix() = %q, want \"passage: \"", got)
	}
	if got := cfg.ResolvedModelName(); got != "intfloat/multilingual-e5-large" {
		t.Errorf("ResolvedModelName() = %q, want \"intfloat/multilingual-e5-large\"", got)
	}
}

func TestVectorSearchConfigResolved_CustomModelWithoutDimensions_Error(t *testing.T) {
	// An unknown model name without dimensions must fail validation.
	cfg := VectorSearchConfig{
		Model:     EmbeddingModelCustom,
		ModelName: "my-custom-model",
		Enabled:   true,
		// Dimensions intentionally omitted.
	}
	err := cfg.validateModelParsed()
	if err == nil {
		t.Error("validateModelParsed() expected error for unknown model without dimensions, got nil")
	}
}

func TestVectorSearchConfigResolved_CustomModelWithoutMaxTokens_Error(t *testing.T) {
	// An unknown model without an explicit max_input_tokens must fail validation,
	// same as missing dimensions. Silently defaulting to 8192 breaks small-window
	// models (e.g. e5-large, 512 tokens): every oversized request gets HTTP 400
	// and the embedding job retries forever.
	cfg := VectorSearchConfig{
		Model:      EmbeddingModelCustom,
		ModelName:  "my-custom-model",
		Enabled:    true,
		Dimensions: 768,
		// MaxTokens intentionally omitted.
	}
	err := cfg.validateModelParsed()
	if err == nil {
		t.Error("validateModelParsed() expected error for unknown model without max_input_tokens, got nil")
	}
}

func TestVectorSearchConfigResolved_CustomModelNegativeValues_Error(t *testing.T) {
	// Negative overrides must fail validation the same way as missing ones:
	// the Resolved* getters treat non-positive values as "not set" and silently
	// fall back to defaults, hiding the config mistake.
	t.Run("negative dimensions", func(t *testing.T) {
		cfg := VectorSearchConfig{
			Model:      EmbeddingModelCustom,
			ModelName:  "my-custom-model",
			Enabled:    true,
			Dimensions: -1,
			MaxTokens:  512,
		}
		if err := cfg.validateModelParsed(); err == nil {
			t.Error("validateModelParsed() expected error for negative dimensions, got nil")
		}
	})
	t.Run("negative max_input_tokens", func(t *testing.T) {
		cfg := VectorSearchConfig{
			Model:      EmbeddingModelCustom,
			ModelName:  "my-custom-model",
			Enabled:    true,
			Dimensions: 768,
			MaxTokens:  -512,
		}
		if err := cfg.validateModelParsed(); err == nil {
			t.Error("validateModelParsed() expected error for negative max_input_tokens, got nil")
		}
	})
}

func TestParseFeatures_KnownModel(t *testing.T) {
	// Parse must resolve known model correctly.
	t.Setenv("OPENAI_API_KEY", "sk-test")
	f := Parse(`{"vector_search":{"enabled":true,"model":"bge-m3","base_url":"http://localhost:8080/v1"}}`)
	if f.VectorSearch.Model != EmbeddingModelBGEM3 {
		t.Errorf("Model = %v, want EmbeddingModelBGEM3", f.VectorSearch.Model)
	}
	if got := f.VectorSearch.ResolvedDimensions(); got != 1024 {
		t.Errorf("ResolvedDimensions() = %d, want 1024", got)
	}
}

func TestParseFeatures_CustomModelWithDimensions(t *testing.T) {
	// Parse must accept an unknown model when dimensions are provided.
	customJSON := `{"vector_search":{"enabled":true,"model":"intfloat/multilingual-e5-large",` +
		`"base_url":"http://localhost:8080/v1","dimensions":1024,"max_input_tokens":512,` +
		`"query_prefix":"query: ","passage_prefix":"passage: "}}`
	f := Parse(customJSON)
	if f.VectorSearch.Model != EmbeddingModelCustom {
		t.Errorf("Model = %v, want EmbeddingModelCustom", f.VectorSearch.Model)
	}
	if got := f.VectorSearch.ResolvedDimensions(); got != 1024 {
		t.Errorf("ResolvedDimensions() = %d, want 1024", got)
	}
	if got := f.VectorSearch.ResolvedModelName(); got != "intfloat/multilingual-e5-large" {
		t.Errorf("ResolvedModelName() = %q, want \"intfloat/multilingual-e5-large\"", got)
	}
}

func TestParseFeatures_CustomModelWithoutDimensions_Panics(t *testing.T) {
	// Parse must panic when an unknown model is used without dimensions.
	defer func() {
		if r := recover(); r == nil {
			t.Error("Parse() expected panic for unknown model without dimensions, got none")
		}
	}()
	Parse(`{"vector_search":{"enabled":true,"model":"my-custom-model","base_url":"http://localhost:8080/v1"}}`)
}
