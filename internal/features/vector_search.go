package features

import (
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// EmbeddingModel represents an OpenAI embedding model.
type EmbeddingModel int

const (
	EmbeddingModelSmall EmbeddingModel = 1 // text-embedding-3-small, 1536 dims
	EmbeddingModelLarge EmbeddingModel = 2 // text-embedding-3-large, 3072 dims
	EmbeddingModelAda   EmbeddingModel = 3 // text-embedding-ada-002, 1536 dims (legacy)
)

// String returns the OpenAI API model name.
func (m EmbeddingModel) String() string {
	switch m {
	case EmbeddingModelSmall:
		return "text-embedding-3-small"
	case EmbeddingModelLarge:
		return "text-embedding-3-large"
	case EmbeddingModelAda:
		return "text-embedding-ada-002"
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
	default:
		return 0, fmt.Errorf("unknown embedding model: %s", s)
	}
}

// VectorSearchConfig holds configuration for vector search feature.
type VectorSearchConfig struct {
	Enabled   bool           `json:"enabled"`
	ModelName string         `json:"model"`
	Model     EmbeddingModel `json:"-"` // Parsed from ModelName
}

// Validate validates vector search configuration.
func (c VectorSearchConfig) Validate() error {
	return ozzo.ValidateStruct(&c,
		ozzo.Field(&c.ModelName, ozzo.When(c.Enabled, ozzo.Required)),
	)
}
