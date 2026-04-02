package features

import (
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// EmbeddingModel represents an OpenAI embedding model.
type EmbeddingModel int

const (
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

// Dimensions returns the embedding vector dimensions for the model.
func (m EmbeddingModel) Dimensions() int {
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

// ParseEmbeddingModel parses model name string to EmbeddingModel.
func ParseEmbeddingModel(s string) (EmbeddingModel, error) {
	switch s {
	case "text-embedding-3-small", "small":
		return EmbeddingModelSmall, nil
	case "text-embedding-3-large", "large":
		return EmbeddingModelLarge, nil
	case "text-embedding-ada-002", "ada":
		return EmbeddingModelAda, nil
	case "multilingual-e5-base":
		return EmbeddingModelMultilingualE5Base, nil
	case "bge-m3":
		return EmbeddingModelBGEM3, nil
	default:
		return 0, fmt.Errorf("unknown embedding model: %s", s)
	}
}

// QueryPrefix returns the prefix to prepend to search queries before embedding.
// Some models (e.g. intfloat/multilingual-e5-*) require "query: " prefix.
// Returns empty string for models that don't require it.
func (m EmbeddingModel) QueryPrefix() string {
	switch m {
	case EmbeddingModelSmall, EmbeddingModelLarge, EmbeddingModelAda, EmbeddingModelBGEM3:
		// no-op: these models don't require a query prefix
	case EmbeddingModelMultilingualE5Base:
		return "query: "
	}
	return ""
}

// PassagePrefix returns the prefix to prepend to document passages before embedding.
// Some models (e.g. intfloat/multilingual-e5-*) require "passage: " prefix.
// Returns empty string for models that don't require it.
func (m EmbeddingModel) PassagePrefix() string {
	switch m {
	case EmbeddingModelSmall, EmbeddingModelLarge, EmbeddingModelAda, EmbeddingModelBGEM3:
		// no-op: these models don't require a passage prefix
	case EmbeddingModelMultilingualE5Base:
		return "passage: "
	}
	return ""
}

// VectorSearchConfig holds configuration for vector search feature.
type VectorSearchConfig struct {
	Enabled   bool           `json:"enabled"`
	ModelName string         `json:"model"`
	Model     EmbeddingModel `json:"-"`        // Parsed from ModelName
	BaseURL   string         `json:"base_url"` // optional; empty = OpenAI default; set to Ollama base URL for local embeddings
}

// Validate validates vector search configuration.
func (c VectorSearchConfig) Validate() error {
	return ozzo.ValidateStruct(&c,
		ozzo.Field(&c.ModelName, ozzo.When(c.Enabled, ozzo.Required)),
	)
}
