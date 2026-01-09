package similarnotes

import (
	"context"
	"math"
	"sort"

	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

const (
	defaultLimit = 5
	maxLimit     = 20
)

type Env interface {
	Features() features.Features
	LatestNoteViews() *appmodel.NoteViews
	CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error)
}

type similarNoteScore struct {
	noteView *appmodel.NoteView
	score    float64
}

func Resolve(ctx context.Context, env Env, input model.SimilarNotesInput) ([]model.SimilarNote, error) {
	// Return empty if vector search is disabled
	if !env.Features().VectorSearch.Enabled {
		return []model.SimilarNote{}, nil
	}

	// Get the source note
	noteViews := env.LatestNoteViews()
	// Try PathMap first (e.g., "hello world.md"), then Map (Permalink, e.g., "/hello_world")
	sourceNote := noteViews.PathMap[input.Path]
	if sourceNote == nil {
		sourceNote = noteViews.Map[input.Path]
	}
	if sourceNote == nil {
		return []model.SimilarNote{}, nil
	}

	// Check if source note has embedding
	if len(sourceNote.Embedding) == 0 {
		return []model.SimilarNote{}, nil
	}

	// Calculate limit
	limit := defaultLimit
	if input.Limit != nil {
		limit = int(*input.Limit)
		if limit < 1 {
			limit = 1
		}
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	// Calculate similarity scores using cached embeddings
	scores := make([]similarNoteScore, 0, len(noteViews.List))
	for _, note := range noteViews.List {
		if note.VersionID == sourceNote.VersionID {
			continue
		}

		if len(note.Embedding) == 0 {
			continue
		}

		score := cosineSimilarity(sourceNote.Embedding, note.Embedding)
		scores = append(scores, similarNoteScore{
			noteView: note,
			score:    score,
		})
	}

	// Sort by similarity score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Filter by permissions and build result
	result := make([]model.SimilarNote, 0, limit)
	for _, s := range scores {
		if len(result) >= limit {
			break
		}

		canRead, err := env.CanReadNote(ctx, s.noteView)
		if err != nil {
			return nil, err
		}
		if !canRead {
			continue
		}

		result = append(result, model.SimilarNote{
			Score: s.score,
			Note:  model.ConvertNoteToPublic(s.noteView),
		})
	}

	return result, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 means identical direction.
// TODO: Consider replacing with Bleve's FAISS-based vector search when CGO is acceptable.
// See: https://github.com/blevesearch/bleve/blob/master/docs/vectors.md
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
