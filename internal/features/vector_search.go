package features

import (
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// EmbeddingModel represents a known embedding model.
type EmbeddingModel int

const (
	// EmbeddingModelCustom is the sentinel value used when the model name is not
	// one of the five built-in models. Its parameters are supplied entirely via
	// the explicit override fields in VectorSearchConfig.
	EmbeddingModelCustom             EmbeddingModel = 0
	EmbeddingModelSmall              EmbeddingModel = 1 // text-embedding-3-small, 1536 dims
	EmbeddingModelLarge              EmbeddingModel = 2 // text-embedding-3-large, 3072 dims
	EmbeddingModelAda                EmbeddingModel = 3 // text-embedding-ada-002, 1536 dims (legacy)
	EmbeddingModelMultilingualE5Base EmbeddingModel = 4 // multilingual-e5-base, 768 dims (HuggingFace)
	EmbeddingModelBGEM3              EmbeddingModel = 5 // bge-m3, 1024 dims (HuggingFace)
)

// String returns the API model name.
func (m EmbeddingModel) String() string {
	switch m {
	case EmbeddingModelSmall:
		return "text-embedding-3-small"
	case EmbeddingModelLarge:
		return "text-embedding-3-large"
	case EmbeddingModelAda:
		return "text-embedding-ada-002"
	case EmbeddingModelMultilingualE5Base:
		return "multilingual-e5-base"
	case EmbeddingModelBGEM3:
		return "bge-m3"
	default:
		return ""
	}
}

// defaultDimensions returns the built-in vector dimensions for known models.
func (m EmbeddingModel) defaultDimensions() int {
	switch m {
	case EmbeddingModelSmall:
		return 1536
	case EmbeddingModelLarge:
		return 3072
	case EmbeddingModelAda:
		return 1536
	case EmbeddingModelMultilingualE5Base:
		return 768
	case EmbeddingModelBGEM3:
		return 1024
	default:
		return 0
	}
}

// defaultMaxInputTokens returns the built-in token limit for known models.
func (m EmbeddingModel) defaultMaxInputTokens() int {
	switch m {
	case EmbeddingModelSmall, EmbeddingModelLarge, EmbeddingModelAda, EmbeddingModelBGEM3:
		return 8192
	case EmbeddingModelMultilingualE5Base:
		return 512
	default:
		return 8192
	}
}

// defaultQueryPrefix returns the built-in query prefix for known models.
func (m EmbeddingModel) defaultQueryPrefix() string {
	switch m {
	case EmbeddingModelMultilingualE5Base:
		return "query: "
	}
	return ""
}

// defaultPassagePrefix returns the built-in passage prefix for known models.
func (m EmbeddingModel) defaultPassagePrefix() string {
	switch m {
	case EmbeddingModelMultilingualE5Base:
		return "passage: "
	}
	return ""
}

// Dimensions returns the embedding vector dimensions for the model.
// Deprecated: use VectorSearchConfig.ResolvedDimensions instead, which
// respects explicit overrides from the FEATURES config.
func (m EmbeddingModel) Dimensions() int {
	return m.defaultDimensions()
}

// MaxInputTokens returns the hard input-token limit the model's embedding
// endpoint accepts in a single request. Inputs beyond this are rejected
// (OpenAI returns HTTP 400 "maximum input length is 8192 tokens"), so callers
// must truncate before embedding whole notes.
// Deprecated: use VectorSearchConfig.ResolvedMaxInputTokens instead.
func (m EmbeddingModel) MaxInputTokens() int {
	return m.defaultMaxInputTokens()
}

// QueryPrefix returns the prefix to prepend to search queries before embedding.
// Some models (e.g. intfloat/multilingual-e5-*) require "query: " prefix.
// Returns empty string for models that don't require it.
// Deprecated: use VectorSearchConfig.ResolvedQueryPrefix instead.
func (m EmbeddingModel) QueryPrefix() string {
	return m.defaultQueryPrefix()
}

// PassagePrefix returns the prefix to prepend to document passages before embedding.
// Some models (e.g. intfloat/multilingual-e5-*) require "passage: " prefix.
// Returns empty string for models that don't require it.
// Deprecated: use VectorSearchConfig.ResolvedPassagePrefix instead.
func (m EmbeddingModel) PassagePrefix() string {
	return m.defaultPassagePrefix()
}

// ParseEmbeddingModel parses a model name string to EmbeddingModel.
// Returns (0, false) for model names not in the built-in enum; the caller
// is responsible for requiring explicit dimension overrides in that case.
func ParseEmbeddingModel(s string) (EmbeddingModel, bool) {
	switch s {
	case "text-embedding-3-small", "small":
		return EmbeddingModelSmall, true
	case "text-embedding-3-large", "large":
		return EmbeddingModelLarge, true
	case "text-embedding-ada-002", "ada":
		return EmbeddingModelAda, true
	case "multilingual-e5-base":
		return EmbeddingModelMultilingualE5Base, true
	case "bge-m3":
		return EmbeddingModelBGEM3, true
	default:
		return EmbeddingModelCustom, false
	}
}

// VectorSearchConfig holds configuration for the vector search feature.
//
// For the five built-in model names (text-embedding-3-small, text-embedding-3-large,
// text-embedding-ada-002, multilingual-e5-base, bge-m3) all parameters default to
// their known values; any explicit field below overrides the default.
//
// For any other model name (e.g. a local TEI server serving an arbitrary model),
// Dimensions must be set explicitly; the remaining fields default to safe values.
//
// Example FEATURES config for a custom local model:
//
//	{
//	  "vector_search": {
//	    "enabled": true,
//	    "model": "intfloat/multilingual-e5-large",
//	    "base_url": "http://localhost:8080/v1",
//	    "dimensions": 1024,
//	    "query_prefix": "query: ",
//	    "passage_prefix": "passage: "
//	  }
//	}
type VectorSearchConfig struct {
	Enabled   bool           `json:"enabled"`
	ModelName string         `json:"model"`
	Model     EmbeddingModel `json:"-"` // Parsed from ModelName; EmbeddingModelCustom for unknown names.

	// BaseURL overrides the default OpenAI endpoint. Use this to point at any
	// OpenAI-compatible embeddings endpoint, e.g. a local TEI server.
	BaseURL string `json:"base_url"`

	// The following optional fields override the per-model defaults.
	// For known models they are not required. For unknown (custom) models,
	// Dimensions must be set; others fall back to safe defaults (empty prefixes,
	// 8192 token limit).
	Dimensions int    `json:"dimensions"`
	MaxTokens  int    `json:"max_input_tokens"`
	QueryPfx   string `json:"query_prefix"`
	PassagePfx string `json:"passage_prefix"`
}

// ResolvedDimensions returns the embedding vector size: explicit override if
// set, otherwise the built-in default for the model.
func (c VectorSearchConfig) ResolvedDimensions() int {
	if c.Dimensions > 0 {
		return c.Dimensions
	}
	return c.Model.defaultDimensions()
}

// ResolvedMaxInputTokens returns the max token input: explicit override if
// set, otherwise the built-in default (8192 for unknown models).
func (c VectorSearchConfig) ResolvedMaxInputTokens() int {
	if c.MaxTokens > 0 {
		return c.MaxTokens
	}
	return c.Model.defaultMaxInputTokens()
}

// ResolvedQueryPrefix returns the query prefix string: explicit override if
// set, otherwise the built-in default (empty for unknown models).
// An explicit empty string ("") is indistinguishable from "not set" in JSON
// — use the override only when you want a non-empty prefix for a custom model.
func (c VectorSearchConfig) ResolvedQueryPrefix() string {
	if c.QueryPfx != "" {
		return c.QueryPfx
	}
	return c.Model.defaultQueryPrefix()
}

// ResolvedPassagePrefix returns the passage prefix string: explicit override
// if set, otherwise the built-in default (empty for unknown models).
func (c VectorSearchConfig) ResolvedPassagePrefix() string {
	if c.PassagePfx != "" {
		return c.PassagePfx
	}
	return c.Model.defaultPassagePrefix()
}

// ResolvedModelName returns the model name string to use in API requests.
// For known models this is the canonical name (e.g. "text-embedding-3-small");
// for custom models it is the raw ModelName from config.
func (c VectorSearchConfig) ResolvedModelName() string {
	if c.Model != EmbeddingModelCustom {
		return c.Model.String()
	}
	return c.ModelName
}

// Validate validates vector search configuration.
func (c VectorSearchConfig) Validate() error {
	return ozzo.ValidateStruct(&c,
		ozzo.Field(&c.ModelName, ozzo.When(c.Enabled, ozzo.Required)),
	)
}

// validateModelParsed returns an error if the config requires dimensions for a
// custom model but none were provided. This is called after model parsing so
// we can distinguish known from unknown models.
func (c VectorSearchConfig) validateModelParsed() error {
	if c.Enabled && c.Model == EmbeddingModelCustom && c.Dimensions == 0 {
		return fmt.Errorf("vector_search: dimensions must be set when model %q is not a built-in model", c.ModelName)
	}
	return nil
}
