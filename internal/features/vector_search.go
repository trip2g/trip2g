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
	case EmbeddingModelCustom:
		return ""
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
	}
	return ""
}

// defaultDimensions returns the built-in vector dimensions for known models.
func (m EmbeddingModel) defaultDimensions() int {
	switch m {
	case EmbeddingModelCustom:
		return 0
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
	}
	return 0
}

// defaultMaxInputTokens returns the built-in token limit for known models.
// Unknown/custom models default to 8192.
func (m EmbeddingModel) defaultMaxInputTokens() int {
	switch m {
	case EmbeddingModelCustom:
		return 8192
	case EmbeddingModelSmall, EmbeddingModelLarge, EmbeddingModelAda, EmbeddingModelBGEM3:
		return 8192
	case EmbeddingModelMultilingualE5Base:
		return 512
	}
	return 8192
}

// defaultQueryPrefix returns the built-in query prefix for known models.
// Only multilingual-e5-base requires a prefix; all others return empty string.
func (m EmbeddingModel) defaultQueryPrefix() string {
	if m == EmbeddingModelMultilingualE5Base {
		return "query: "
	}
	return ""
}

// defaultPassagePrefix returns the built-in passage prefix for known models.
// Only multilingual-e5-base requires a prefix; all others return empty string.
func (m EmbeddingModel) defaultPassagePrefix() string {
	if m == EmbeddingModelMultilingualE5Base {
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
// Returns (EmbeddingModelCustom, false) for model names not in the built-in
// enum; the caller is responsible for requiring explicit dimension overrides
// in that case.
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
// Dimensions and MaxTokens must be set explicitly; the remaining fields default
// to safe values.
//
// Example FEATURES config for a custom local model:
//
//	{
//	  "vector_search": {
//	    "enabled": true,
//	    "model": "intfloat/multilingual-e5-large",
//	    "base_url": "http://localhost:8080/v1",
//	    "dimensions": 1024,
//	    "max_input_tokens": 512,
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
	// Dimensions and MaxTokens must be set; prefixes fall back to empty.
	Dimensions int    `json:"dimensions"`
	MaxTokens  int    `json:"max_input_tokens"`
	QueryPfx   string `json:"query_prefix"`
	PassagePfx string `json:"passage_prefix"`

	Reranker RerankerConfig `json:"reranker"` // optional second-stage cross-encoder reranker (blend mode)
}

// RerankerConfig configures an optional cross-encoder rerank stage applied to
// the fused (RRF) result set. The cross-encoder score is BLENDED with the
// first-stage RRF score, not substituted for it — the CE nudges the order, it
// does not override a strong first stage. When disabled, search returns the RRF
// order. Only candidates whose passage fits the CE window (~512 tokens) are
// rescored; the rest keep their stage-1 score. See docs/dev/reranker.md.
type RerankerConfig struct {
	Enabled        bool    `json:"enabled"`
	BaseURL        string  `json:"base_url"`        // rerank endpoint, e.g. "http://reranker:8000/rerank"
	Model          string  `json:"model"`           // e.g. "BAAI/bge-reranker-v2-m3"
	TopN           int     `json:"top_n"`           // candidates to rerank (default 50)
	OutputK        int     `json:"output_k"`        // results to keep after rerank (default 20)
	BlendWeight    float64 `json:"blend_weight"`    // CE weight in [0,1]: final = (1-w)*rrf_norm + w*ce_norm (default 0.5)
	TimeoutSeconds int     `json:"timeout_seconds"` // per-request rerank timeout (default 10; raise for CPU inference)
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

// validateModelParsed returns an error if the config requires dimensions or
// max_input_tokens for a custom model but none were provided. This is called
// after model parsing so we can distinguish known from unknown models.
func (c VectorSearchConfig) validateModelParsed() error {
	if c.Enabled && c.Model == EmbeddingModelCustom && c.Dimensions <= 0 {
		return fmt.Errorf(
			"vector_search: dimensions must be set to a positive value when model %q is not a built-in model",
			c.ModelName,
		)
	}
	// Silently defaulting the token limit breaks small-window models: every
	// oversized request gets HTTP 400 and the embedding job retries forever.
	if c.Enabled && c.Model == EmbeddingModelCustom && c.MaxTokens <= 0 {
		return fmt.Errorf(
			"vector_search: max_input_tokens must be set to a positive value when model %q is not a built-in model",
			c.ModelName,
		)
	}
	return nil
}
