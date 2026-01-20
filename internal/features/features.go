package features

import (
	"encoding/json"
	"fmt"
	"os"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// Features holds all feature flag configurations.
type Features struct {
	VectorSearch VectorSearchConfig `json:"vector_search"`
}

// DefaultFeatures returns features with all flags disabled by default.
func DefaultFeatures() Features {
	return Features{
		VectorSearch: VectorSearchConfig{
			Enabled:   false,
			ModelName: "text-embedding-3-small",
			Model:     EmbeddingModelSmall,
		},
	}
}

// Parse parses features from JSON string, validates, and checks dependencies.
// Panics if validation fails or required environment variables are missing.
// Returns default features if json is empty.
func Parse(jsonStr string) Features {
	f := DefaultFeatures()

	if jsonStr != "" && jsonStr != "{}" {
		err := json.Unmarshal([]byte(jsonStr), &f)
		if err != nil {
			panic(fmt.Sprintf("failed to parse features JSON: %v", err))
		}
	}

	// Validate all feature configurations
	err := ozzo.ValidateStruct(&f,
		ozzo.Field(&f.VectorSearch),
	)
	if err != nil {
		panic(fmt.Sprintf("features validation failed: %v", err))
	}

	// Check required environment variables and parse models for enabled features
	if f.VectorSearch.Enabled {
		if os.Getenv("OPENAI_API_KEY") == "" {
			panic("OPENAI_API_KEY environment variable is required when vector_search.enabled=true")
		}

		model, modelErr := ParseEmbeddingModel(f.VectorSearch.ModelName)
		if modelErr != nil {
			panic(fmt.Sprintf("invalid vector_search.model: %v", modelErr))
		}
		f.VectorSearch.Model = model
	}

	return f
}
