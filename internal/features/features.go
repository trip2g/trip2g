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
		// OPENAI_API_KEY is only required when using the OpenAI endpoint (base_url not set).
		// When base_url is set (e.g. a local TEI server), no API key is needed.
		if f.VectorSearch.BaseURL == "" && os.Getenv("OPENAI_API_KEY") == "" {
			panic("OPENAI_API_KEY environment variable is required when vector_search.enabled=true and base_url is not set")
		}

		model, known := ParseEmbeddingModel(f.VectorSearch.ModelName)
		f.VectorSearch.Model = model

		// For unknown (custom) models, dimensions must be supplied explicitly.
		if !known {
			if parseErr := f.VectorSearch.validateModelParsed(); parseErr != nil {
				panic(fmt.Sprintf("features validation failed: %v", parseErr))
			}
		}
	}

	return f
}
