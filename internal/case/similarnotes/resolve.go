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
	LatestNoteChunks() []appmodel.NoteChunk
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

	limit := clampLimit(input.Limit)

	chunkMap := buildChunkMap(env.LatestNoteChunks())

	srcChunks := chunkMap[sourceNote.Path]
	useChunks := len(srcChunks) > 0

	if !useChunks && len(sourceNote.Embedding) == 0 {
		return []model.SimilarNote{}, nil
	}

	scores := calculateScores(noteViews, sourceNote, srcChunks, useChunks, chunkMap)

	// Sort by similarity score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return filterByPermissions(ctx, env, scores, limit)
}

func clampLimit(limit *int32) int {
	if limit == nil {
		return defaultLimit
	}
	v := int(*limit)
	if v < 1 {
		return 1
	}
	if v > maxLimit {
		return maxLimit
	}
	return v
}

func buildChunkMap(allChunks []appmodel.NoteChunk) map[string][]appmodel.NoteChunk {
	chunkMap := make(map[string][]appmodel.NoteChunk, len(allChunks)/4+1)
	for _, c := range allChunks {
		if len(c.Embedding) > 0 {
			chunkMap[c.NotePath] = append(chunkMap[c.NotePath], c)
		}
	}
	return chunkMap
}

func calculateScores(
	noteViews *appmodel.NoteViews,
	sourceNote *appmodel.NoteView,
	srcChunks []appmodel.NoteChunk,
	useChunks bool,
	chunkMap map[string][]appmodel.NoteChunk,
) []similarNoteScore {
	scores := make([]similarNoteScore, 0, len(noteViews.List))
	for _, note := range noteViews.List {
		if note.VersionID == sourceNote.VersionID {
			continue
		}
		if note.IsSystem() {
			continue
		}

		tgtChunks := chunkMap[note.Path]
		var score float64
		switch {
		case useChunks && len(tgtChunks) > 0:
			score = maxChunkSimilarity(srcChunks, tgtChunks)
		case useChunks && len(note.Embedding) > 0:
			for _, sc := range srcChunks {
				if s := cosineSimilarity(sc.Embedding, note.Embedding); s > score {
					score = s
				}
			}
		case !useChunks && len(note.Embedding) > 0:
			score = cosineSimilarity(sourceNote.Embedding, note.Embedding)
		default:
			continue
		}

		scores = append(scores, similarNoteScore{noteView: note, score: score})
	}
	return scores
}

func filterByPermissions(ctx context.Context, env Env, scores []similarNoteScore, limit int) ([]model.SimilarNote, error) {
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

// maxChunkSimilarity returns the maximum cosine similarity across all src×tgt chunk pairs.
func maxChunkSimilarity(src, tgt []appmodel.NoteChunk) float64 {
	var best float64
	for _, s := range src {
		for _, t := range tgt {
			if sc := cosineSimilarity(s.Embedding, t.Embedding); sc > best {
				best = sc
			}
		}
	}
	return best
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
