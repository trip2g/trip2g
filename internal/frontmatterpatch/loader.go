package frontmatterpatch

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/db"
)

// Env defines the interface for loading frontmatter patches from database.
type Env interface {
	ListEnabledFrontmatterPatches(ctx context.Context) ([]db.NoteFrontmatterPatch, error)
}

// Loader loads and compiles frontmatter patches from database.
type Loader struct {
	env Env
}

// NewLoader creates a new frontmatter patch loader.
func NewLoader(env Env) *Loader {
	return &Loader{
		env: env,
	}
}

// LoadFrontmatterPatches loads enabled patches from database and compiles them.
func (l *Loader) LoadFrontmatterPatches(ctx context.Context) ([]CompiledPatch, error) {
	patches, err := l.env.ListEnabledFrontmatterPatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list frontmatter patches: %w", err)
	}

	compiled := make([]CompiledPatch, len(patches))
	for i, patch := range patches {
		var includePatterns []string
		err = json.Unmarshal([]byte(patch.IncludePatterns), &includePatterns)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal include patterns for patch %d: %w", patch.ID, err)
		}

		var excludePatterns []string
		err = json.Unmarshal([]byte(patch.ExcludePatterns), &excludePatterns)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal exclude patterns for patch %d: %w", patch.ID, err)
		}

		compiled[i] = Compile(
			int(patch.ID),
			includePatterns,
			excludePatterns,
			patch.Jsonnet,
			int(patch.Priority),
			patch.Description,
		)
	}

	return compiled, nil
}
